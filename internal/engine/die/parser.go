package die

import (
	"strconv"
	"strings"
	"unicode"
)

// TokenType classifies input fragments for the lexer.
type TokenType int

const (
	TokenVerb       TokenType = iota // find, search, show, open, extract, etc.
	TokenModifier                    // all, hidden, largest, duplicate, etc.
	TokenExtension                   // pdf, exe, log, png, etc.
	TokenSizeFilter                  // >100MB, <1GB, =500KB
	TokenPath                        // /var/log, ~/Desktop, C:\Users
	TokenKeyword                     // anything else
)

// Token is a classified fragment from the input.
type Token struct {
	Type  TokenType
	Value string
	Pos   int // Position in original input
}

// ASTNode is a node in the parsed command tree.
type ASTNode struct {
	Action      string
	Modifiers   []string
	Extensions  []string
	SearchTerm  string
	Path        string
	SizeFilter  *SizeFilter
	DateFilter  *DateFilter
	OwnerFilter string
	Flags       map[string]bool
}

// SizeFilter represents a size comparison filter.
type SizeFilter struct {
	Op    string // >, <, >=, <=, =
	Value uint64
	Unit  string
}

// DateFilter represents a modified-date filter.
type DateFilter struct {
	Op    string
	Value int
	Unit  string // d, h, m, w, mo, y
}

// Noise words stripped during tokenization.
var noiseWords = map[string]bool{
	"all": true, "please": true, "the": true, "a": true,
	"an": true, "file": true, "files": true, "folder": true,
	"folders": true, "that": true, "which": true, "with": true,
	"from": true, "in": true, "on": true, "at": true,
}

// Known verbs mapped to action types.
var verbMap = map[string]string{
	"find":      "SEARCH",
	"search":    "SEARCH",
	"locate":    "SEARCH",
	"look":      "SEARCH",
	"lookup":    "SEARCH",
	"query":     "SEARCH",
	"filter":    "SEARCH",
	"open":      "NAVIGATE",
	"cd":        "NAVIGATE",
	"go":        "NAVIGATE",
	"navigate":  "NAVIGATE",
	"goto":      "NAVIGATE",
	"browse":    "NAVIGATE",
	"enter":     "NAVIGATE",
	"switch":    "NAVIGATE",
	"show":      "ANALYZE",
	"list":      "ANALYZE",
	"display":   "ANALYZE",
	"analyze":   "ANALYZE",
	"inspect":   "ANALYZE",
	"examine":   "ANALYZE",
	"info":      "ANALYZE",
	"details":   "ANALYZE",
	"status":    "ANALYZE",
	"extract":   "EXTRACT",
	"export":    "EXTRACT",
	"copy":      "EXTRACT",
	"save":      "EXTRACT",
	"dump":      "EXTRACT",
	"pull":      "EXTRACT",
	"preview":   "PREVIEW",
	"view":      "PREVIEW",
	"peek":      "PREVIEW",
	"compare":   "COMPARE",
	"diff":      "COMPARE",
	"hash":      "HASH",
	"checksum":  "HASH",
	"verify":    "HASH",
	"recover":   "RECOVERY",
	"restore":   "RECOVERY",
	"undelete":  "RECOVERY",
	"report":    "REPORT",
	"summary":   "REPORT",
	"generate":  "REPORT",
	"settings":  "SETTINGS",
	"config":    "SETTINGS",
}

// Known file extensions for classification.
var knownExtensions = map[string]bool{
	"pdf": true, "png": true, "jpg": true, "jpeg": true, "gif": true,
	"bmp": true, "svg": true, "ico": true, "webp": true,
	"json": true, "xml": true, "yaml": true, "yml": true,
	"log": true, "txt": true, "csv": true,
	"exe": true, "dll": true, "sys": true, "drv": true,
	"ini": true, "cfg": true, "conf": true, "toml": true,
	"zip": true, "tar": true, "gz": true, "7z": true, "rar": true,
	"mp3": true, "mp4": true, "avi": true, "mkv": true, "mov": true,
	"doc": true, "docx": true, "xls": true, "xlsx": true, "ppt": true,
	"pptx": true, "odt": true, "ods": true,
	"sql": true, "db": true, "sqlite": true, "mdb": true,
	"key": true, "pem": true, "crt": true, "cer": true,
	"ssh": true, "pub": true, "p12": true,
	"go": true, "py": true, "js": true, "ts": true, "rs": true,
	"c": true, "cpp": true, "h": true, "java": true, "rb": true,
	"sh": true, "bash": true, "ps1": true, "bat": true, "cmd": true,
	"so": true, "dylib": true,
	"iso": true, "img": true, "raw": true, "dd": true, "bin": true,
	"vhd": true, "vhdx": true, "vmdk": true, "qcow2": true, "vdi": true,
	"dockerfile": true, "makefile": true, "readme": true,
}

// Modifiers that change search scope or behavior.
var modifierWords = map[string]bool{
	"hidden": true, "system": true, "large": true, "largest": true,
	"biggest": true, "big": true, "small": true, "smallest": true,
	"duplicate": true, "dupes": true, "empty": true, "zero": true,
	"recent": true, "old": true, "new": true, "today": true,
	"yesterday": true, "this": true, "last": true, "week": true,
	"month": true, "year": true, "user": true, "users": true,
	"owners": true, "by": true, "under": true, "recursive": true,
}

// Lexer performs deterministic tokenization of raw input.
type Lexer struct {
	input []rune
	pos   int
}

// NewLexer creates a Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(input), pos: 0}
}

// Tokenize converts raw input into a token stream.
func (l *Lexer) Tokenize() []Token {
	var tokens []Token

	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		// Path token (starts with / or ~ or drive letter)
		if ch == '/' || ch == '~' || ch == '\\' || (ch >= 'A' && ch <= 'Z' && l.pos+1 < len(l.input) && l.input[l.pos+1] == ':') {
			tokens = append(tokens, l.readPath())
			continue
		}

		// Size filter (>100MB, <1GB, etc.)
		if ch == '>' || ch == '<' || ch == '=' {
			tok := l.readSizeFilter()
			if tok != nil {
				tokens = append(tokens, *tok)
				continue
			}
		}

		// Word token
		word := l.readWord()
		if word == "" {
			l.pos++
			continue
		}

		tok := l.classifyWord(word, len(tokens)-1, tokens)
		tokens = append(tokens, tok)
	}

	return tokens
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) readPath() Token {
	start := l.pos
	for l.pos < len(l.input) && !unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
	return Token{Type: TokenPath, Value: string(l.input[start:l.pos]), Pos: start}
}

func (l *Lexer) readSizeFilter() *Token {
	start := l.pos
	op := string(l.input[l.pos])
	l.pos++

	// Check for >= or <=
	if l.pos < len(l.input) && l.input[l.pos] == '=' {
		op += "="
		l.pos++
	}

	// Skip optional whitespace after operator
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}

	// Read numeric value
	numStart := l.pos
	for l.pos < len(l.input) && (unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	if l.pos == numStart {
		return nil // No number after operator
	}

	val := string(l.input[numStart:l.pos])

	// Read unit (KB, MB, GB, TB)
	unitStart := l.pos
	for l.pos < len(l.input) && unicode.IsLetter(l.input[l.pos]) {
		l.pos++
	}
	unit := strings.ToUpper(string(l.input[unitStart:l.pos]))

	return &Token{Type: TokenSizeFilter, Value: op + val + unit, Pos: start}
}

func (l *Lexer) readWord() string {
	start := l.pos
	for l.pos < len(l.input) && !unicode.IsSpace(l.input[l.pos]) && l.input[l.pos] != '>' && l.input[l.pos] != '<' && l.input[l.pos] != '=' {
		l.pos++
	}
	return string(l.input[start:l.pos])
}

func (l *Lexer) classifyWord(word string, prevIdx int, tokens []Token) Token {
	lower := strings.ToLower(word)

	// Check if it's a known verb
	if action, ok := verbMap[lower]; ok {
		_ = action
		return Token{Type: TokenVerb, Value: lower, Pos: l.pos - len(word)}
	}

	// Check if it's a known extension (with or without dot)
	cleanExt := strings.TrimPrefix(lower, ".")
	if knownExtensions[cleanExt] {
		return Token{Type: TokenExtension, Value: cleanExt, Pos: l.pos - len(word)}
	}

	// Check if it's a modifier
	if modifierWords[lower] {
		return Token{Type: TokenModifier, Value: lower, Pos: l.pos - len(word)}
	}

	// Check if previous token was a verb that expects a path (open, cd)
	if prevIdx >= 0 && tokens[prevIdx].Type == TokenVerb {
		v := tokens[prevIdx].Value
		if v == "open" || v == "cd" || v == "goto" || v == "navigate" || v == "browse" || v == "enter" || v == "switch" {
			return Token{Type: TokenPath, Value: lower, Pos: l.pos - len(word)}
		}
	}

	return Token{Type: TokenKeyword, Value: lower, Pos: l.pos - len(word)}
}

// Parse converts a token stream into an AST (ExecutableCommand).
func Parse(tokens []Token) *ASTNode {
	node := &ASTNode{
		Flags: make(map[string]bool),
	}

	for i, tok := range tokens {
		switch tok.Type {
		case TokenVerb:
			if action, ok := verbMap[tok.Value]; ok {
				if node.Action == "" {
					node.Action = action
				}
			}
			// Handle "go to" as compound verb
			if tok.Value == "go" && i+1 < len(tokens) && tokens[i+1].Value == "to" {
				node.Action = "NAVIGATE"
			}

		case TokenExtension:
			node.Extensions = append(node.Extensions, tok.Value)

		case TokenModifier:
			mod := tok.Value
			// Interpret modifiers into flags or search terms
			switch mod {
			case "hidden", "system":
				node.Flags[mod] = true
			case "largest", "biggest", "large", "big":
				node.Modifiers = append(node.Modifiers, "largest")
			case "smallest", "small":
				node.Modifiers = append(node.Modifiers, "smallest")
			case "duplicate", "dupes":
				node.Flags["duplicate"] = true
			case "empty", "zero":
				node.Flags["empty"] = true
			case "user", "users", "owners":
				node.Modifiers = append(node.Modifiers, "users")
			default:
				node.Modifiers = append(node.Modifiers, mod)
			}

		case TokenSizeFilter:
			node.SizeFilter = parseSizeFilter(tok.Value)

		case TokenPath:
			// If path is after "open/cd/etc.", it's the target path
			if i > 0 && tokens[i-1].Type == TokenVerb {
				if _, ok := verbMap[tokens[i-1].Value]; ok {
					node.Path = tok.Value
				}
			} else if node.Path == "" {
				node.Path = tok.Value
			}

		case TokenKeyword:
			// Accumulate keywords as search term
			if node.SearchTerm != "" {
				node.SearchTerm += " " + tok.Value
			} else {
				node.SearchTerm = tok.Value
			}
		}
	}

	// Clean up search term (remove noise)
	if node.SearchTerm != "" {
		words := strings.Fields(node.SearchTerm)
		var clean []string
		for _, w := range words {
			if !noiseWords[w] {
				clean = append(clean, w)
			}
		}
		node.SearchTerm = strings.Join(clean, " ")
	}

	// Default action
	if node.Action == "" {
		if node.SearchTerm != "" || len(node.Extensions) > 0 {
			node.Action = "SEARCH"
		} else {
			node.Action = "SEARCH" // Fallback to search
		}
	}

	return node
}

// ParseInput is the main entry point: tokenizes input, builds AST, returns Intent.
func ParseInput(input string, cmdCtx CommandContext) (*Intent, error) {
	lexer := NewLexer(input)
	tokens := lexer.Tokenize()
	ast := Parse(tokens)

	// Convert AST to Intent
	intent := &Intent{
		Action:     ActionType(ast.Action),
		Query:      ast.SearchTerm,
		Target:     ast.Path,
		Filters:    make(map[string]string),
		Params:     make(map[string]string),
		RawCommand: input,
		Confidence: 0.5,
	}

	// Map extensions
	if len(ast.Extensions) > 0 {
		intent.Filters["ext"] = strings.Join(ast.Extensions, ",")
	}

	// Map size filter
	if ast.SizeFilter != nil {
		intent.Filters["size_op"] = ast.SizeFilter.Op
		intent.Filters["size_val"] = strconv.FormatUint(ast.SizeFilter.Value, 10)
		intent.Filters["size_unit"] = ast.SizeFilter.Unit
	}

	// Map flags
	for flag, val := range ast.Flags {
		if val {
			intent.Filters[flag] = "true"
		}
	}

	// Map modifiers
	for _, mod := range ast.Modifiers {
		intent.Filters["modifier"] = mod
	}

	// Context boost
	applyContext(intent, cmdCtx)

	return intent, nil
}

func parseSizeFilter(raw string) *SizeFilter {
	// Parse ">100MB" into SizeFilter
	f := &SizeFilter{}
	remaining := raw

	// Extract operator
	if strings.HasPrefix(remaining, ">=") {
		f.Op = ">="
		remaining = remaining[2:]
	} else if strings.HasPrefix(remaining, "<=") {
		f.Op = "<="
		remaining = remaining[2:]
	} else if strings.HasPrefix(remaining, ">") {
		f.Op = ">"
		remaining = remaining[1:]
	} else if strings.HasPrefix(remaining, "<") {
		f.Op = "<"
		remaining = remaining[1:]
	} else if strings.HasPrefix(remaining, "=") || strings.HasPrefix(remaining, "==") {
		f.Op = "="
		remaining = strings.TrimPrefix(remaining, "=")
	}

	// Extract number
	i := 0
	for i < len(remaining) && (remaining[i] >= '0' && remaining[i] <= '9' || remaining[i] == '.') {
		i++
	}
	if i == 0 {
		return nil
	}

	val, _ := strconv.ParseFloat(remaining[:i], 64)
	f.Unit = strings.ToUpper(remaining[i:])

	switch f.Unit {
	case "KB":
		f.Value = uint64(val * 1024)
	case "MB":
		f.Value = uint64(val * 1024 * 1024)
	case "GB":
		f.Value = uint64(val * 1024 * 1024 * 1024)
	case "TB":
		f.Value = uint64(val * 1024 * 1024 * 1024 * 1024)
	default:
		f.Value = uint64(val)
	}

	return f
}

func applyContext(intent *Intent, cmdCtx CommandContext) {
	// Boost confidence based on context
	if cmdCtx.ActiveTab == "explorer" && intent.Action == "SEARCH" {
		intent.Confidence += 0.1
	}

	if cmdCtx.SelectedFile != "" && (intent.Action == "PREVIEW" || intent.Action == "EXTRACT") {
		intent.Confidence += 0.15
	}

	// Fill missing target from context
	if intent.Target == "" && cmdCtx.CurrentPath != "" {
		intent.Target = cmdCtx.CurrentPath
	}

	if intent.Target == "" && cmdCtx.SelectedFile != "" {
		intent.Target = cmdCtx.SelectedFile
	}

	// Cap confidence
	if intent.Confidence > 1.0 {
		intent.Confidence = 1.0
	}
}
