package ucl

import (
	"testing"
)

func TestParseUCLQuery(t *testing.T) {
	query := "FIND WHERE extension=pdf AND size>10MB LIMIT 20"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error parsing UCL: %v", err)
	}

	if ast.Verb != "FIND" {
		t.Errorf("Expected Verb FIND, got %s", ast.Verb)
	}

	if ast.CapabilityID != "cap.disk.search" {
		t.Errorf("Expected CapabilityID cap.disk.search, got %s", ast.CapabilityID)
	}

	if len(ast.Conditions) != 2 {
		t.Fatalf("Expected 2 conditions, got %d", len(ast.Conditions))
	}

	if ast.Conditions[0].Field != "extension" || ast.Conditions[0].Value != "pdf" {
		t.Errorf("Condition 0 mismatch: %+v", ast.Conditions[0])
	}

	if ast.Limit != 20 {
		t.Errorf("Expected Limit 20, got %d", ast.Limit)
	}
}

func TestParseWildcardSearch(t *testing.T) {
	query := "*.pdf"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ast.Verb != "FIND" {
		t.Errorf("Expected FIND verb, got %s", ast.Verb)
	}

	if len(ast.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(ast.Conditions))
	}

	if ast.Conditions[0].Field != "pattern" {
		t.Errorf("Expected field pattern, got %s", ast.Conditions[0].Field)
	}

	if ast.Conditions[0].Value != "*.pdf" {
		t.Errorf("Expected value *.pdf, got %s", ast.Conditions[0].Value)
	}
}

func TestParseAnalyzeVerb(t *testing.T) {
	query := "ANALYZE WHERE type=users"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ast.Verb != "ANALYZE" {
		t.Errorf("Expected ANALYZE, got %s", ast.Verb)
	}

	if ast.CapabilityID != "cap.analysis.system" {
		t.Errorf("Expected cap.analysis.system, got %s", ast.CapabilityID)
	}

	if len(ast.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(ast.Conditions))
	}

	if ast.Conditions[0].Field != "type" || ast.Conditions[0].Value != "users" {
		t.Errorf("Condition mismatch: %+v", ast.Conditions[0])
	}
}

func TestParseToParamMap(t *testing.T) {
	query := "FIND WHERE extension=pdf AND size>10MB ORDER BY modified DESC LIMIT 50"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	params := ast.ToParamMap()

	if params["verb"] != "FIND" {
		t.Errorf("Expected verb FIND, got %v", params["verb"])
	}

	if params["extension"] != "pdf" {
		t.Errorf("Expected extension=pdf, got %v", params["extension"])
	}

	if params["size"] != "10MB" {
		t.Errorf("Expected size=10MB, got %v", params["size"])
	}

	if params["order_dir"] != "DESC" {
		t.Errorf("Expected order_dir DESC, got %v", params["order_dir"])
	}

	if params["limit"] != 50 {
		t.Errorf("Expected limit 50, got %v", params["limit"])
	}
}

func TestParseOpenVerb(t *testing.T) {
	query := "OPEN /Windows/System32"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ast.Verb != "OPEN" {
		t.Errorf("Expected OPEN, got %s", ast.Verb)
	}

	if ast.CapabilityID != "cap.vfs.navigate" {
		t.Errorf("Expected cap.vfs.navigate, got %s", ast.CapabilityID)
	}

	if len(ast.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(ast.Conditions))
	}

	if ast.Conditions[0].Value != "/Windows/System32" {
		t.Errorf("Expected path /Windows/System32, got %s", ast.Conditions[0].Value)
	}
}

func TestParseExtractVerb(t *testing.T) {
	query := "EXTRACT /etc/passwd"

	ast, err := Parse(query)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if ast.Verb != "EXTRACT" {
		t.Errorf("Expected EXTRACT, got %s", ast.Verb)
	}

	if ast.CapabilityID != "cap.vfs.extract" {
		t.Errorf("Expected cap.vfs.extract, got %s", ast.CapabilityID)
	}
}
