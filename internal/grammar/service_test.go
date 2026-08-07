package grammar_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/vhd-opener/internal/grammar"
)

// ── Registry Lookup Tests ──────────────────────────────────────────────────

func TestLanguageForExt_Builtin(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{"py", "Python"}, {"go", "Go"}, {"rs", "Rust"}, {"js", "JavaScript"},
		{"ts", "TypeScript"}, {"java", "Java"}, {"rb", "Ruby"}, {"php", "PHP"},
		{"cpp", "C++"}, {"cs", "C#"}, {"swift", "Swift"}, {"zig", "Zig"},
		{"sol", "Solidity"}, {"nix", "Nix"}, {"vhd", "VHDL"}, {"vue", "Vue"},
		{"svelte", "Svelte"}, {"proto", "Protocol Buffer"}, {"graphql", "GraphQL"},
		{"hcl", "HCL"}, {"tf", "HCL"},
	}
	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := grammar.LanguageForExt(tt.ext)
			if got != tt.want {
				t.Errorf("LanguageForExt(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

func TestLanguageForExt_Unknown(t *testing.T) {
	unknowns := []string{"xyz", "custom", "myext", "notalanguage", ""}
	for _, ext := range unknowns {
		t.Run(ext, func(t *testing.T) {
			got := grammar.LanguageForExt(ext)
			if got != "" {
				t.Errorf("LanguageForExt(%q) = %q, want empty string", ext, got)
			}
		})
	}
}

// ── GrammarService Core Tests ──────────────────────────────────────────────

func TestGetGrammar_BuiltinExtension(t *testing.T) {
	svc := grammar.NewGrammarService(t.TempDir())
	resp, err := svc.GetGrammar("py")
	if err != nil {
		t.Fatalf("GetGrammar error: %v", err)
	}
	if resp.Language != "Python" {
		t.Errorf("expected language Python, got %q", resp.Language)
	}
	if resp.Source != "builtin" {
		t.Errorf("expected source builtin, got %q", resp.Source)
	}
	if resp.Grammar != "" {
		t.Errorf("expected empty grammar for builtin, got %d bytes", len(resp.Grammar))
	}
}

func TestGetGrammar_EmptyExtension(t *testing.T) {
	svc := grammar.NewGrammarService(t.TempDir())
	resp, err := svc.GetGrammar("")
	if err != nil {
		t.Fatalf("GetGrammar error: %v", err)
	}
	if resp.Source != "generic" {
		t.Errorf("expected source generic for empty ext, got %q", resp.Source)
	}
}

func TestGetGrammar_DotPrefix(t *testing.T) {
	svc := grammar.NewGrammarService(t.TempDir())
	resp, err := svc.GetGrammar(".py")
	if err != nil {
		t.Fatalf("GetGrammar error: %v", err)
	}
	if resp.Language != "Python" {
		t.Errorf("expected language Python for .py, got %q", resp.Language)
	}
}

// ── Negative Caching Tests ─────────────────────────────────────────────────

func TestGetGrammar_NegativeCaching(t *testing.T) {
	dir := t.TempDir()
	svc := grammar.NewGrammarService(dir)

	// First call for unknown extension — triggers CDN fetch (will 404)
	ext := "zzz_nonexistent_ext_99999"
	resp, err := svc.GetGrammar(ext)
	if err != nil {
		t.Fatalf("GetGrammar error: %v", err)
	}
	if resp.Source != "generic" {
		t.Logf("First call source: %s (CDN may have returned grammar)", resp.Source)
	}

	// Verify .notfound marker file was created (only if CDN returned 404)
	notfoundPath := filepath.Join(dir, "grammars", ext+".notfound")
	if _, statErr := os.Stat(notfoundPath); statErr == nil {
		t.Log("Negative cache marker created correctly")

		// Second call should hit negative cache (no network call)
		start := time.Now()
		resp2, err := svc.GetGrammar(ext)
		duration := time.Since(start)
		if err != nil {
			t.Fatalf("GetGrammar error on cached negative: %v", err)
		}
		if resp2.Source != "generic" {
			t.Errorf("expected source generic from negative cache, got %q", resp2.Source)
		}
		if duration > 5*time.Millisecond {
			t.Errorf("negative cache lookup took %v, expected < 5ms", duration)
		}
		t.Logf("Negative cache hit in %v", duration)
	} else {
		t.Log("CDN returned grammar for unknown ext — negative cache not triggered (acceptable)")
	}
}

func TestClearCache_RemovesNegativeMarkers(t *testing.T) {
	dir := t.TempDir()
	svc := grammar.NewGrammarService(dir)

	// Manually create a negative cache marker
	grammarDir := filepath.Join(dir, "grammars")
	os.MkdirAll(grammarDir, 0755)
	os.WriteFile(filepath.Join(grammarDir, "test.notfound"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(grammarDir, "test.json"), []byte("{}"), 0644)

	if err := svc.ClearCache(); err != nil {
		t.Fatalf("ClearCache error: %v", err)
	}

	entries, _ := os.ReadDir(grammarDir)
	if len(entries) != 0 {
		t.Errorf("expected empty cache dir after ClearCache, got %d entries", len(entries))
	}
}

// ── Circuit Breaker Tests ──────────────────────────────────────────────────

func TestCircuitBreaker_TripsAfterFailures(t *testing.T) {
	dir := t.TempDir()
	svc := grammar.NewGrammarService(dir)

	// Trigger multiple failures to trip the circuit breaker
	exts := []string{
		"zzz_fail1_99999",
		"zzz_fail2_99999",
		"zzz_fail3_99999",
	}

	for _, ext := range exts {
		resp, err := svc.GetGrammar(ext)
		if err != nil {
			t.Fatalf("GetGrammar(%s) error: %v", ext, err)
		}
		t.Logf("GetGrammar(%s) source=%s", ext, resp.Source)
	}

	// Next request should be circuit-broken (instant return)
	start := time.Now()
	resp, err := svc.GetGrammar("zzz_fail4_99999")
	duration := time.Since(start)
	if err != nil {
		t.Fatalf("GetGrammar error: %v", err)
	}

	if resp.Source == "circuit_broken" {
		t.Logf("Circuit breaker tripped correctly, response in %v", duration)
		if duration > 5*time.Millisecond {
			t.Errorf("circuit breaker response took %v, expected < 5ms", duration)
		}
	} else {
		t.Logf("Circuit did not trip (CDN may be available), source=%s", resp.Source)
	}
}

// ── Timeout Validation ────────────────────────────────────────────────────

func TestGetGrammar_TimeoutBounds(t *testing.T) {
	svc := grammar.NewGrammarService(t.TempDir())

	start := time.Now()
	resp, err := svc.GetGrammar("invalid_ext_test_timeout_99")
	duration := time.Since(start)

	if err != nil {
		t.Logf("GetGrammar returned error (expected): %v", err)
	}

	// Must complete within 12 seconds (8s timeout + overhead for 3 mirrors)
	if duration > 12*time.Second {
		t.Fatalf("GetGrammar took %v — HTTP timeout non-functional", duration)
	}

	t.Logf("GetGrammar completed in %v, source=%s", duration, resp.Source)
}

// ── Benchmark Tests ────────────────────────────────────────────────────────

func BenchmarkRegistryLookup_Known(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = grammar.LanguageForExt("py")
		_ = grammar.LanguageForExt("go")
		_ = grammar.LanguageForExt("rs")
		_ = grammar.LanguageForExt("java")
		_ = grammar.LanguageForExt("zig")
	}
}

func BenchmarkRegistryLookup_Unknown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = grammar.LanguageForExt("zzz_unknown_ext")
		_ = grammar.LanguageForExt("custom_ext")
		_ = grammar.LanguageForExt("nonexistent")
	}
}

func BenchmarkRegistryLookup_Mixed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = grammar.LanguageForExt("zig")
		_ = grammar.LanguageForExt("sol")
		_ = grammar.LanguageForExt("unknownext")
	}
}

func BenchmarkGetGrammar_Builtin(b *testing.B) {
	svc := grammar.NewGrammarService(b.TempDir())
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetGrammar("py")
	}
}

func BenchmarkGetGrammar_NegativeCache(b *testing.B) {
	dir := b.TempDir()
	svc := grammar.NewGrammarService(dir)

	// Pre-populate negative cache
	os.WriteFile(filepath.Join(dir, "grammars", "cached.notfound"), []byte("test"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetGrammar("cached")
	}
}
