package lexer

//go:generate go run ../../../tools/tokengen -in tokens.def -out tokens_gen.go

type Token struct {
	Type   TokenType
	Lexeme string
	Value  any
	Pos    int
}

func IsNameToken(token Token) bool {
	switch token.Type {
	case TokenEOF, TokenDot, TokenEquals, TokenValue:
		return false
	default:
		return token.Lexeme != ""
	}
}
