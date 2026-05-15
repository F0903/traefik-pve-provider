package labels

import (
	"sort"

	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
	"github.com/F0903/traefik-pve-provider/traefik/ast/parser"
)

func Enabled(labels map[string]string) bool {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := labels[key]
		tokens := lexer.Lex(key, value)
		if !isTraefikLabel(tokens) {
			continue
		}
		node, err := parser.ParseTokens(tokens, value, parser.Context{})
		if err != nil {
			continue
		}
		assignment, ok := node.(ast.Assignment)
		if !ok {
			continue
		}
		segments := assignment.Target.Segments()
		if len(segments) != 1 || segments[0].Type != lexer.TokenEnable {
			continue
		}
		enabled, ok := assignment.Value.(ast.BoolValue)
		return ok && enabled.Value
	}
	return false
}

func Parse(labels map[string]string, defaultName string) (*Set, []Diagnostic) {
	set := newLabelSet()
	context := parser.Context{DefaultName: defaultName}
	diagnostics := make([]Diagnostic, 0)

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := labels[key]
		tokens := lexer.Lex(key, value)
		if !isTraefikLabel(tokens) {
			continue
		}
		set.observeTokens(tokens)
		node, err := parser.ParseTokens(tokens, value, context)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Key:   key,
				Value: value,
				Err:   err,
			})
			continue
		}
		set.apply(node, labelAssignmentOriginForTokens(tokens))
	}

	set.applyNameOverride(defaultName)
	return set, diagnostics
}

func isTraefikLabel(tokens []lexer.Token) bool {
	return tokenTypeAt(tokens, 0) == lexer.TokenTraefik && tokenTypeAt(tokens, 1) == lexer.TokenDot
}

func labelAssignmentOriginForTokens(tokens []lexer.Token) labelAssignmentOrigin {
	switch tokenTypeAt(tokens, 2) {
	case lexer.TokenTCP, lexer.TokenUDP:
		switch tokenTypeAt(tokens, 4) {
		case lexer.TokenRouters, lexer.TokenServices:
			return labelAssignmentOriginExplicit
		default:
			return labelAssignmentOriginShorthand
		}
	case lexer.TokenPort,
		lexer.TokenScheme,
		lexer.TokenServersTransport,
		lexer.TokenInsecureSkipVerify,
		lexer.TokenMiddlewares,
		lexer.TokenEntryPoints,
		lexer.TokenEntryPoint:
		return labelAssignmentOriginShorthand
	default:
		return labelAssignmentOriginExplicit
	}
}

func isNamedProtocolObject(tokens []lexer.Token) bool {
	switch tokenTypeAt(tokens, 4) {
	case lexer.TokenRouters, lexer.TokenServices, lexer.TokenServersTransports:
		return tokenTypeAt(tokens, 5) == lexer.TokenDot &&
			lexer.IsNameToken(tokenAt(tokens, 6)) &&
			tokenTypeAt(tokens, 7) == lexer.TokenDot
	default:
		return false
	}
}

func tokenAt(tokens []lexer.Token, index int) lexer.Token {
	if index < 0 || index >= len(tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return tokens[index]
}

func tokenTypeAt(tokens []lexer.Token, index int) lexer.TokenType {
	return tokenAt(tokens, index).Type
}
