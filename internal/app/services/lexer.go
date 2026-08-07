package services

import (
	"strings"
	"unicode"
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	input []rune
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: []rune(strings.TrimSpace(input))}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Value: ""}
	}

	ch := l.input[l.pos]

	if ch == '=' || ch == '>' || ch == '<' || ch == '!' {
		start := l.pos
		l.pos++
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos++
		}
		return Token{Type: TokenOperator, Value: string(l.input[start:l.pos])}
	}

	start := l.pos
	for l.pos < len(l.input) && !unicode.IsSpace(l.input[l.pos]) &&
		l.input[l.pos] != '=' && l.input[l.pos] != '>' && l.input[l.pos] != '<' && l.input[l.pos] != '!' {
		l.pos++
	}

	val := string(l.input[start:l.pos])
	upperVal := strings.ToUpper(val)

	switch upperVal {
	case "FIND", "ANALYZE", "EXTRACT", "OPEN", "SHOW":
		return Token{Type: TokenVerb, Value: upperVal}
	case "WHERE":
		return Token{Type: TokenWhere, Value: upperVal}
	case "AND":
		return Token{Type: TokenAnd, Value: upperVal}
	case "OR":
		return Token{Type: TokenOr, Value: upperVal}
	case "ORDER":
		return Token{Type: TokenOrderBy, Value: upperVal}
	case "BY":
		return Token{Type: TokenValue, Value: upperVal}
	case "ASC":
		return Token{Type: TokenAsc, Value: upperVal}
	case "DESC":
		return Token{Type: TokenDesc, Value: upperVal}
	case "LIMIT":
		return Token{Type: TokenLimit, Value: upperVal}
	default:
		return Token{Type: TokenValue, Value: val}
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}
