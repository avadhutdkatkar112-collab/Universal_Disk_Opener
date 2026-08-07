package services

import (
	"strconv"
	"strings"
)

var verbMap = map[string]string{
	"FIND":    "cap.disk.search",
	"ANALYZE": "cap.analysis.system",
	"EXTRACT": "cap.vfs.extract",
	"OPEN":    "cap.vfs.navigate",
	"SHOW":    "cap.analysis.system",
}

func Parse(input string) (*QueryAST, error) {
	lexer := NewLexer(input)
	ast := &QueryAST{
		RawQuery:   input,
		Conditions: []Condition{},
	}

	token := lexer.NextToken()
	if token.Type != TokenVerb {
		ast.Verb = "FIND"
		ast.CapabilityID = verbMap["FIND"]
		ast.Conditions = append(ast.Conditions, Condition{
			Field:    "pattern",
			Operator: "LIKE",
			Value:    input,
		})
		return ast, nil
	}

	ast.Verb = token.Value
	if capID, exists := verbMap[ast.Verb]; exists {
		ast.CapabilityID = capID
	} else {
		ast.CapabilityID = "cap.disk.search"
	}

	for {
		tok := lexer.NextToken()
		if tok.Type == TokenEOF {
			break
		}

		switch tok.Type {
		case TokenWhere, TokenAnd:
			fieldTok := lexer.NextToken()
			opTok := lexer.NextToken()
			valTok := lexer.NextToken()

			if fieldTok.Type != TokenValue || opTok.Type != TokenOperator || valTok.Type != TokenValue {
				if fieldTok.Type == TokenValue {
					ast.Conditions = append(ast.Conditions, Condition{
						Field:    "pattern",
						Operator: "LIKE",
						Value:    fieldTok.Value,
					})
				}
				continue
			}

			ast.Conditions = append(ast.Conditions, Condition{
				Field:    strings.ToLower(fieldTok.Value),
				Operator: opTok.Value,
				Value:    valTok.Value,
			})

		case TokenLimit:
			valTok := lexer.NextToken()
			if limitVal, err := strconv.Atoi(valTok.Value); err == nil {
				ast.Limit = limitVal
			}

		case TokenOrderBy:
			fieldTok := lexer.NextToken()
			if fieldTok.Type == TokenValue {
				ast.OrderBy = strings.ToLower(fieldTok.Value)
			}

		case TokenAsc, TokenDesc:
			ast.OrderDir = tok.Value

		case TokenValue:
			if strings.ToUpper(tok.Value) == "ORDER" {
				byTok := lexer.NextToken()
				if byTok.Type == TokenValue && strings.ToUpper(byTok.Value) == "BY" {
					fieldTok := lexer.NextToken()
					if fieldTok.Type == TokenValue {
						ast.OrderBy = strings.ToLower(fieldTok.Value)
					}
				}
			} else {
				ast.Conditions = append(ast.Conditions, Condition{
					Field:    "path",
					Operator: "=",
					Value:    tok.Value,
				})
			}
		}
	}

	return ast, nil
}
