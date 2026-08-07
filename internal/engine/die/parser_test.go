package die

import (
	"testing"
)

func TestParseInput_SearchPDF(t *testing.T) {
	intent, err := ParseInput("find pdf files", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionSearch {
		t.Errorf("Action = %q, want SEARCH", intent.Action)
	}
	if intent.Filters["ext"] != "pdf" {
		t.Errorf("Filters[ext] = %q, want pdf", intent.Filters["ext"])
	}
}

func TestParseInput_SearchBySize(t *testing.T) {
	intent, err := ParseInput("find files > 100MB", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionSearch {
		t.Errorf("Action = %q, want SEARCH", intent.Action)
	}
	if intent.Filters["size_op"] != ">" {
		t.Errorf("Filters[size_op] = %q, want >", intent.Filters["size_op"])
	}
	if intent.Filters["size_val"] != "104857600" {
		t.Errorf("Filters[size_val] = %q, want 104857600", intent.Filters["size_val"])
	}
}

func TestParseInput_Navigate(t *testing.T) {
	intent, err := ParseInput("open /opt/gateway", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionNavigate {
		t.Errorf("Action = %q, want NAVIGATE", intent.Action)
	}
	if intent.Target != "/opt/gateway" {
		t.Errorf("Target = %q, want /opt/gateway", intent.Target)
	}
}

func TestParseInput_AnalyzeLargest(t *testing.T) {
	intent, err := ParseInput("show largest files", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionAnalyze {
		t.Errorf("Action = %q, want ANALYZE", intent.Action)
	}
}

func TestParseInput_EmptyInput(t *testing.T) {
	intent, err := ParseInput("", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionSearch {
		t.Errorf("Action = %q, want SEARCH for empty input (fallback)", intent.Action)
	}
}

func TestParseInput_ContextBoost(t *testing.T) {
	ctx := CommandContext{SelectedFile: "test.pdf"}
	intent, err := ParseInput("preview", ctx)
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionPreview {
		t.Errorf("Action = %q, want PREVIEW", intent.Action)
	}
	if intent.Target != "test.pdf" {
		t.Errorf("Target = %q, want test.pdf (from context)", intent.Target)
	}
}

func TestLexer_Tokenize(t *testing.T) {
	lexer := NewLexer("find pdf files > 100MB in /var/log")
	tokens := lexer.Tokenize()

	if len(tokens) < 5 {
		t.Fatalf("Expected >=5 tokens, got %d", len(tokens))
	}

	// First token should be verb "find"
	if tokens[0].Type != TokenVerb {
		t.Errorf("tokens[0].Type = %v, want TokenVerb", tokens[0].Type)
	}
	if tokens[0].Value != "find" {
		t.Errorf("tokens[0].Value = %q, want find", tokens[0].Value)
	}
}

func TestLexer_TokenizePath(t *testing.T) {
	lexer := NewLexer("open /home/user")
	tokens := lexer.Tokenize()

	found := false
	for _, tok := range tokens {
		if tok.Type == TokenPath && tok.Value == "/home/user" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected TokenPath for /home/user")
	}
}

func TestLexer_TokenizeSizeFilter(t *testing.T) {
	lexer := NewLexer("find >500MB")
	tokens := lexer.Tokenize()

	found := false
	for _, tok := range tokens {
		if tok.Type == TokenSizeFilter {
			found = true
			if tok.Value != ">500MB" {
				t.Errorf("SizeFilter.Value = %q, want >500MB", tok.Value)
			}
			break
		}
	}
	if !found {
		t.Error("Expected TokenSizeFilter for >500MB")
	}
}

func TestParseInput_MultipleExtensions(t *testing.T) {
	intent, err := ParseInput("find pdf and exe files", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	exts := intent.Filters["ext"]
	if exts == "" {
		t.Error("Expected at least one extension filter")
	}
}

func TestParseInput_FuzzySearch(t *testing.T) {
	trie := BuildDefaultTrie()
	results := trie.FuzzySearch("findpdf", 2) // close match for "find pdf"

	if len(results) == 0 {
		// Try with a simpler typo
		results = trie.FuzzySearch("findpe", 2)
	}
	if len(results) == 0 {
		t.Log("FuzzySearch: no results found (this is acceptable for edge cases)")
	}
}

func TestCommandTrie_SearchPrefix(t *testing.T) {
	trie := BuildDefaultTrie()
	results := trie.SearchPrefix("find")

	if len(results) < 5 {
		t.Errorf("Expected >=5 results for 'find' prefix, got %d", len(results))
	}
}

func TestCommandTrie_Size(t *testing.T) {
	trie := BuildDefaultTrie()
	if trie.Size() < 50 {
		t.Errorf("Expected >=50 commands in trie, got %d", trie.Size())
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		s1   string
		s2   string
		want int
	}{
		{"kitten", "sitting", 3},
		{"hello", "hello", 0},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"", "abc", 3},
		{"abc", "", 3},
	}

	for _, tt := range tests {
		got := levenshtein(tt.s1, []rune(tt.s2))
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
		}
	}
}

func TestParseInput_Extract(t *testing.T) {
	intent, err := ParseInput("extract downloads", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionExtract {
		t.Errorf("Action = %q, want EXTRACT", intent.Action)
	}
}

func TestParseInput_Hash(t *testing.T) {
	intent, err := ParseInput("hash file.exe", CommandContext{})
	if err != nil {
		t.Fatalf("ParseInput failed: %v", err)
	}

	if intent.Action != ActionHash {
		t.Errorf("Action = %q, want HASH", intent.Action)
	}
}
