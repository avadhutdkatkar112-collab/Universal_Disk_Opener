package services

import "fmt"

type TokenType string

const (
	TokenVerb     TokenType = "VERB"
	TokenWhere    TokenType = "WHERE"
	TokenField    TokenType = "FIELD"
	TokenOperator TokenType = "OPERATOR"
	TokenValue    TokenType = "VALUE"
	TokenAnd      TokenType = "AND"
	TokenOr       TokenType = "OR"
	TokenOrderBy  TokenType = "ORDER_BY"
	TokenAsc      TokenType = "ASC"
	TokenDesc     TokenType = "DESC"
	TokenLimit    TokenType = "LIMIT"
	TokenEOF      TokenType = "EOF"
)

type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type QueryAST struct {
	Verb         string      `json:"verb"`
	CapabilityID string      `json:"capability_id"`
	Conditions   []Condition `json:"conditions"`
	OrderBy      string      `json:"order_by,omitempty"`
	OrderDir     string      `json:"order_dir,omitempty"`
	Limit        int         `json:"limit,omitempty"`
	RawQuery     string      `json:"raw_query"`
}

// ToParamMap translates the AST into a parameter map for Capability.ExecutionContext.
func (ast *QueryAST) ToParamMap() map[string]any {
	params := map[string]any{
		"verb":      ast.Verb,
		"raw_query": ast.RawQuery,
	}

	for _, cond := range ast.Conditions {
		params[cond.Field] = cond.Value
		params[fmt.Sprintf("%s_op", cond.Field)] = cond.Operator
	}

	if ast.OrderBy != "" {
		params["order_by"] = ast.OrderBy
		params["order_dir"] = ast.OrderDir
	}
	if ast.Limit > 0 {
		params["limit"] = ast.Limit
	}

	return params
}
