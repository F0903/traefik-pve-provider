package parser

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

type labelParser struct {
	tokens []lexer.Token
	index  int
	value  string
}

type parsedTarget struct {
	segments  []ast.Segment
	valueType labelValueType
}

func Parse(key, value string, context Context) (ast.Node, *ParseError) {
	return ParseTokens(lexer.Lex(key, value), value, context)
}

func ParseTokens(tokens []lexer.Token, value string, context Context) (ast.Node, *ParseError) {
	parser := labelParser{tokens: tokens, value: value}
	node, err := parser.parse(context)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *labelParser) parse(context Context) (ast.Node, *ParseError) {
	if !p.consume(lexer.TokenTraefik) || !p.consume(lexer.TokenDot) {
		return nil, unsupportedLabel()
	}

	target, err := p.parseRootTarget(context)
	if err != nil {
		return nil, err
	}
	return p.assignment(target)
}

func (p *labelParser) parseRootTarget(context Context) (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenEnable:
		p.advance()
		return target(valueBool, segment(lexer.TokenEnable)), nil
	case lexer.TokenName:
		p.advance()
		return target(valueString, segment(lexer.TokenName)), nil
	case lexer.TokenPort:
		p.advance()
		return p.httpDefaultService(context.DefaultName, valueInt, segment(lexer.TokenServer), segment(lexer.TokenPort)), nil
	case lexer.TokenScheme:
		p.advance()
		return p.httpDefaultService(context.DefaultName, valueString, segment(lexer.TokenServer), segment(lexer.TokenScheme)), nil
	case lexer.TokenInsecureSkipVerify:
		p.advance()
		return p.httpDefaultService(context.DefaultName, valueBool, segment(lexer.TokenInsecureSkipVerify)), nil
	case lexer.TokenMiddlewares:
		p.advance()
		return p.httpDefaultRouter(context.DefaultName, valueCSV, segment(lexer.TokenMiddlewares)), nil
	case lexer.TokenEntryPoints:
		p.advance()
		return p.httpDefaultRouter(context.DefaultName, valueCSV, segment(lexer.TokenEntryPoints)), nil
	case lexer.TokenEntryPoint:
		p.advance()
		return p.httpDefaultRouter(context.DefaultName, valueString, segment(lexer.TokenEntryPoint)), nil
	case lexer.TokenServersTransport:
		p.advance()
		return p.httpDefaultService(context.DefaultName, valueString, segment(lexer.TokenServersTransport)), nil
	case lexer.TokenHTTP:
		p.advance()
		return p.parseHTTP(context)
	case lexer.TokenTCP:
		p.advance()
		return p.parseTCP(context)
	case lexer.TokenUDP:
		p.advance()
		return p.parseUDP(context)
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseNamedObject(collection lexer.TokenType, parseChild func() (parsedTarget, *ParseError)) (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenDot) || !lexer.IsNameToken(p.peek()) {
		return parsedTarget{}, unsupportedLabel()
	}
	name := p.advance().Lexeme
	if !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}
	child, err := parseChild()
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(collection), ast.Identifier(name)), nil
}

func (p *labelParser) parseRouterOption(tcp bool) (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenRule:
		p.advance()
		return target(valueString, segment(lexer.TokenRule)), nil
	case lexer.TokenService:
		p.advance()
		return target(valueString, segment(lexer.TokenService)), nil
	case lexer.TokenEntryPoints:
		p.advance()
		return target(valueCSV, segment(lexer.TokenEntryPoints)), nil
	case lexer.TokenEntryPoint:
		p.advance()
		return target(valueString, segment(lexer.TokenEntryPoint)), nil
	case lexer.TokenMiddlewares:
		p.advance()
		return target(valueCSV, segment(lexer.TokenMiddlewares)), nil
	case lexer.TokenPriority:
		p.advance()
		return target(valueInt, segment(lexer.TokenPriority)), nil
	case lexer.TokenTLS:
		p.advance()
		return p.parseTLSOption(tcp)
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseServerOption(http bool) (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenURL:
		if !http {
			return parsedTarget{}, unsupportedLabel()
		}
		p.advance()
		return target(valueString, segment(lexer.TokenURL)), nil
	case lexer.TokenScheme:
		if !http {
			return parsedTarget{}, unsupportedLabel()
		}
		p.advance()
		return target(valueString, segment(lexer.TokenScheme)), nil
	case lexer.TokenPort:
		p.advance()
		return target(valueInt, segment(lexer.TokenPort)), nil
	case lexer.TokenIP:
		p.advance()
		return target(valueString, segment(lexer.TokenIP)), nil
	case lexer.TokenAddress:
		if http {
			return parsedTarget{}, unsupportedLabel()
		}
		p.advance()
		return target(valueString, segment(lexer.TokenAddress)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) assignment(target parsedTarget) (ast.Node, *ParseError) {
	if len(target.segments) == 0 || !p.consume(lexer.TokenEquals) || p.peek().Type != lexer.TokenValue {
		return nil, unsupportedLabel()
	}
	token := p.advance()
	if !p.consume(lexer.TokenEOF) {
		return nil, unsupportedLabel()
	}

	value, err := parseLabelValue(token.Lexeme, target.valueType)
	if err != nil {
		return nil, err
	}
	return ast.Assignment{
		Target: ast.NewTarget(target.segments...),
		Value:  value,
	}, nil
}

func target(valueType labelValueType, segments ...ast.Segment) parsedTarget {
	return parsedTarget{
		segments:  segments,
		valueType: valueType,
	}
}

func (t parsedTarget) prepend(segments ...ast.Segment) parsedTarget {
	next := make([]ast.Segment, 0, len(segments)+len(t.segments))
	next = append(next, segments...)
	next = append(next, t.segments...)
	t.segments = next
	return t
}

func segment(tokenType lexer.TokenType) ast.Segment {
	return ast.SegmentFor(tokenType)
}

func (p *labelParser) peek() lexer.Token {
	if p.index >= len(p.tokens) {
		return lexer.Token{Type: lexer.TokenEOF}
	}
	return p.tokens[p.index]
}

func (p *labelParser) advance() lexer.Token {
	token := p.peek()
	if p.index < len(p.tokens) {
		p.index++
	}
	return token
}

func (p *labelParser) consume(tokenType lexer.TokenType) bool {
	if p.peek().Type != tokenType {
		return false
	}
	p.advance()
	return true
}
