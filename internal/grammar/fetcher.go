package grammar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// GrammarResponse is returned to the frontend with the fetched grammar.
type GrammarResponse struct {
	Grammar   string `json:"grammar"`   // TextMate JSON or empty
	Language  string `json:"language"`  // Canonical language name
	Extension string `json:"extension"` // Original extension
	Source    string `json:"source"`    // "builtin", "cache", "cdn", "negative", "generic", "circuit_broken"
}

// grammarCacheEntry stores a grammar with metadata for ETag validation.
type grammarCacheEntry struct {
	Grammar    string `json:"grammar"`
	ETag       string `json:"etag,omitempty"`
	FetchedAt  int64  `json:"fetched_at"`
	SHA256Hash string `json:"sha256,omitempty"`
}

// ── 3-State Circuit Breaker ────────────────────────────────────────────────

type breakerState int

const (
	breakerClosed   breakerState = iota // Normal operation
	breakerOpen                         // Failing fast, no network calls
	breakerHalfOpen                     // Trial request to probe upstream
)

const (
	breakerFailThreshold = 5                // failures to trip open
	breakerOpenDuration  = 30 * time.Second // cooldown before half-open
)

type circuitBreaker struct {
	mu            sync.Mutex
	state         breakerState
	failCount     int
	successCount  int
	lastFailTime  time.Time
	lastStateTime time.Time
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{state: breakerClosed}
}

func (cb *circuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case breakerClosed:
		return true
	case breakerOpen:
		if time.Since(cb.lastStateTime) > breakerOpenDuration {
			cb.state = breakerHalfOpen
			return true // Allow one trial request
		}
		return false
	case breakerHalfOpen:
		return true // Allow the trial request
	}
	return false
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failCount = 0
	cb.successCount++
	if cb.state == breakerHalfOpen {
		cb.state = breakerClosed
		cb.lastStateTime = time.Now()
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failCount++
	cb.successCount = 0
	cb.lastFailTime = time.Now()
	cb.lastStateTime = time.Now()

	if cb.state == breakerHalfOpen {
		// Trial request failed — reopen
		cb.state = breakerOpen
		return
	}
	if cb.failCount >= breakerFailThreshold {
		cb.state = breakerOpen
	}
}

// ── CDN Mirror Definition ──────────────────────────────────────────────────

type cdnMirror struct {
	name    string
	urlTmpl string // %s = extension
}

var mirrors = []cdnMirror{
	{
		name:    "jsdelivr-shikijs",
		urlTmpl: "https://cdn.jsdelivr.net/gh/shikijs/textmate-grammars@main/grammars/%s.tmLanguage.json",
	},
	{
		name:    "github-raw-shikijs",
		urlTmpl: "https://raw.githubusercontent.com/shikijs/textmate-grammars/main/grammars/%s.tmLanguage.json",
	},
	{
		name:    "unpkg-tm-grammars",
		urlTmpl: "https://unpkg.com/tm-grammars/grammars/%s.tmLanguage.json",
	},
}

const (
	negativeCacheSuffix = ".notfound"
	hedgeDelay          = 150 * time.Millisecond // wait before hedged request
	maxGrammarSize      = 5 * 1024 * 1024        // 5MB max grammar
	etagFileSuffix      = ".etag"
)

// ── GrammarService ─────────────────────────────────────────────────────────

// GrammarService manages fetching, caching, and serving TextMate grammars.
type GrammarService struct {
	cacheDir   string
	httpClient *http.Client
	mu         sync.RWMutex
	breaker    *circuitBreaker
	flight     singleflight.Group // request collapsing
}

// NewGrammarService creates a new grammar service.
func NewGrammarService(appDataDir string) *GrammarService {
	dir := filepath.Join(appDataDir, "grammars")
	os.MkdirAll(dir, 0755)
	return &GrammarService{
		cacheDir: dir,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
		breaker: newCircuitBreaker(),
	}
}

// GetGrammar resolves a file extension to a TextMate grammar.
// Implements 3-tier resolution with singleflight dedup, hedged requests,
// negative caching, ETag validation, and 3-state circuit breaker.
func (g *GrammarService) GetGrammar(ext string) (*GrammarResponse, error) {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "" {
		return &GrammarResponse{Source: "generic"}, nil
	}

	// ── Tier 1: Built-in mapping ──────────────────────────────────────────
	if lang := LanguageForExt(ext); lang != "" {
		return &GrammarResponse{
			Language:  lang,
			Extension: ext,
			Source:    "builtin",
		}, nil
	}

	// ── Tier 2: Check negative cache (known-missing) ──────────────────────
	if g.isNegativeCached(ext) {
		return &GrammarResponse{
			Extension: ext,
			Source:    "generic",
		}, nil
	}

	// ── Tier 2: Check local grammar cache with ETag ───────────────────────
	g.mu.RLock()
	cached, etag, err := g.readCacheWithETag(ext)
	g.mu.RUnlock()
	if err == nil && cached != "" {
		return &GrammarResponse{
			Grammar:   cached,
			Extension: ext,
			Source:    "cache",
		}, nil
	}

	// ── Circuit breaker check ──────────────────────────────────────────────
	if !g.breaker.allowRequest() {
		return &GrammarResponse{
			Extension: ext,
			Source:    "circuit_broken",
		}, nil
	}

	// ── Tier 2: CDN fetch with singleflight dedup ─────────────────────────
	// If multiple goroutines request the same extension simultaneously,
	// only one HTTP call is made.
	resultKey := "grammar:" + ext
	result, err, _ := g.flight.Do(resultKey, func() (interface{}, error) {
		return g.fetchWithHedgedRequests(ext, etag)
	})

	if err != nil {
		g.breaker.recordFailure()
		return &GrammarResponse{
			Extension: ext,
			Source:    "generic",
		}, nil
	}

	resp := result.(*GrammarResponse)
	g.breaker.recordSuccess()
	return resp, nil
}

// ── Hedged Requests (Happy Eyeballs) ───────────────────────────────────────

// fetchWithHedgedRequests dispatches to Mirror 1, and after hedgeDelay
// dispatches a concurrent request to Mirror 2 if Mirror 1 hasn't responded.
// Returns the first successful result.
func (g *GrammarService) fetchWithHedgedRequests(ext string, existingETag string) (*GrammarResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	type mirrorResult struct {
		grammar string
		source  string
		err     error
	}

	resultCh := make(chan mirrorResult, len(mirrors))

	// Launch Mirror 1 immediately
	go func() {
		grammar, source, err := g.fetchSingleMirror(ctx, mirrors[0], ext, existingETag)
		resultCh <- mirrorResult{grammar, source, err}
	}()

	// Launch hedged Mirror 2 after delay
	timer := time.NewTimer(hedgeDelay)
	defer timer.Stop()

	select {
	case <-timer.C:
		// No response yet — launch hedge
		go func() {
			grammar, source, err := g.fetchSingleMirror(ctx, mirrors[1], ext, existingETag)
			resultCh <- mirrorResult{grammar, source, err}
		}()
	case res := <-resultCh:
		// Mirror 1 responded before hedge delay
		if res.err == nil && res.grammar != "" {
			return &GrammarResponse{
				Grammar:   res.grammar,
				Extension: ext,
				Source:    res.source,
			}, nil
		}
		// Mirror 1 failed — fall through to try remaining mirrors
		timer.Stop()
	}

	// Collect results from all mirrors
	allNotFound := true
	for i := 0; i < len(mirrors); i++ {
		select {
		case res := <-resultCh:
			if res.err == nil && res.grammar != "" {
				return &GrammarResponse{
					Grammar:   res.grammar,
					Extension: ext,
					Source:    res.source,
				}, nil
			}
			if res.err == nil && res.source != "" {
				allNotFound = false
			}
		case <-ctx.Done():
			break
		}
	}

	// Try remaining mirrors if first two failed
	for i := 2; i < len(mirrors); i++ {
		grammar, source, err := g.fetchSingleMirror(ctx, mirrors[i], ext, existingETag)
		if err == nil && grammar != "" {
			return &GrammarResponse{
				Grammar:   grammar,
				Extension: ext,
				Source:    source,
			}, nil
		}
	}

	if allNotFound {
		g.writeNegativeCache(ext)
	}

	return &GrammarResponse{
		Extension: ext,
		Source:    "generic",
	}, fmt.Errorf("all mirrors failed for .%s", ext)
}

// fetchSingleMirror attempts to download a grammar from a single CDN mirror.
// Supports ETag-based conditional GET to avoid re-downloading unchanged grammars.
func (g *GrammarService) fetchSingleMirror(ctx context.Context, mirror cdnMirror, ext string, existingETag string) (string, string, error) {
	url := fmt.Sprintf(mirror.urlTmpl, ext)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", err
	}

	// Conditional GET: If-None-Match
	if existingETag != "" {
		req.Header.Set("If-None-Match", existingETag)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxGrammarSize))
		if err != nil {
			return "", "", err
		}

		// Validate JSON
		var parsed interface{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return "", "", fmt.Errorf("invalid JSON from %s: %w", mirror.name, err)
		}

		// Save to cache with ETag
		etag := resp.Header.Get("ETag")
		g.writeCacheWithETag(ext, string(body), etag)

		return string(body), "cdn", nil

	case http.StatusNotModified:
		// 304 — our cached version is still valid
		// Re-read from cache (it's already there)
		cached, _, _ := g.readCacheWithETag(ext)
		if cached != "" {
			return cached, "cache", nil
		}
		return "", "", fmt.Errorf("304 but no cache for .%s", ext)

	case http.StatusNotFound:
		return "", "not_found", nil

	case http.StatusTooManyRequests:
		return "", "rate_limited", nil

	default:
		if resp.StatusCode >= 500 {
			return "", "server_error", nil
		}
		return "", "", fmt.Errorf("%s: HTTP %d", mirror.name, resp.StatusCode)
	}
}

// ── Local Cache with ETag Support ──────────────────────────────────────────

func (g *GrammarService) readCacheWithETag(ext string) (grammar string, etag string, err error) {
	// Read grammar JSON
	jsonPath := filepath.Join(g.cacheDir, ext+".json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return "", "", err
	}

	// Read ETag
	etagPath := filepath.Join(g.cacheDir, ext+etagFileSuffix)
	etagData, _ := os.ReadFile(etagPath)

	return string(data), string(etagData), nil
}

func (g *GrammarService) writeCacheWithETag(ext, grammar, etag string) {
	jsonPath := filepath.Join(g.cacheDir, ext+".json")
	os.WriteFile(jsonPath, []byte(grammar), 0644)

	if etag != "" {
		etagPath := filepath.Join(g.cacheDir, ext+etagFileSuffix)
		os.WriteFile(etagPath, []byte(etag), 0644)
	}

	// Compute and store SHA-256 for integrity verification
	hash := sha256.Sum256([]byte(grammar))
	hashPath := filepath.Join(g.cacheDir, ext+".sha256")
	os.WriteFile(hashPath, []byte(hex.EncodeToString(hash[:])), 0644)
}

// VerifyCacheIntegrity checks if a cached grammar matches its stored SHA-256.
func (g *GrammarService) VerifyCacheIntegrity(ext string) bool {
	jsonPath := filepath.Join(g.cacheDir, ext+".json")
	hashPath := filepath.Join(g.cacheDir, ext+".sha256")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return false
	}
	expectedHash, err := os.ReadFile(hashPath)
	if err != nil {
		return false
	}

	actual := sha256.Sum256(data)
	return hex.EncodeToString(actual[:]) == string(expectedHash)
}

// ── Negative Caching ───────────────────────────────────────────────────────

func (g *GrammarService) isNegativeCached(ext string) bool {
	path := filepath.Join(g.cacheDir, ext+negativeCacheSuffix)
	_, err := os.Stat(path)
	return err == nil
}

func (g *GrammarService) writeNegativeCache(ext string) {
	path := filepath.Join(g.cacheDir, ext+negativeCacheSuffix)
	os.WriteFile(path, []byte(time.Now().Format(time.RFC3339)), 0644)
}

// ── Cache Management ───────────────────────────────────────────────────────

func (g *GrammarService) readCache(ext string) (string, error) {
	grammar, _, err := g.readCacheWithETag(ext)
	return grammar, err
}

// ClearCache removes all cached grammars and markers.
func (g *GrammarService) ClearCache() error {
	entries, err := os.ReadDir(g.cacheDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		os.Remove(filepath.Join(g.cacheDir, e.Name()))
	}
	return nil
}
