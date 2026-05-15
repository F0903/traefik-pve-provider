package lexer

import (
	"regexp"
	"strconv"
	"strings"
)

var domainsIndexPattern = regexp.MustCompile(`^domains\[(\d+)\]$`)

func Lex(key, value string) []Token {
	segments := strings.Split(strings.ToLower(strings.TrimSpace(key)), ".")
	tokens := make([]Token, 0, len(segments)*2+3)
	for index, segment := range segments {
		if index > 0 {
			tokens = append(tokens, Token{Type: TokenDot, Lexeme: ".", Pos: index})
		}
		tokens = append(tokens, tokenForSegment(segment, index))
	}
	tokens = append(tokens,
		Token{Type: TokenEquals, Lexeme: "="},
		Token{Type: TokenValue, Lexeme: value, Value: value},
		Token{Type: TokenEOF},
	)
	return tokens
}

func tokenForSegment(segment string, pos int) Token {
	if matches := domainsIndexPattern.FindStringSubmatch(segment); matches != nil {
		index, _ := strconv.Atoi(matches[1])
		return Token{Type: TokenDomains, Lexeme: segment, Value: index, Pos: pos}
	}

	tokenType := StringToTokenType[segment]
	if tokenType == 0 {
		tokenType = TokenIdentifier
	}
	return Token{Type: tokenType, Lexeme: segment, Pos: pos}
}
