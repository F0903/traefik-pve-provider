package parser

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (p *labelParser) parseTLSOption(tcp bool) (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenDot) {
		return target(valueBool, segment(lexer.TokenTLS)), nil
	}

	var child parsedTarget
	var err *ParseError
	switch p.peek().Type {
	case lexer.TokenPassthrough:
		if !tcp {
			return parsedTarget{}, unsupportedLabel()
		}
		p.advance()
		child = target(valueBool, segment(lexer.TokenPassthrough))
	case lexer.TokenCertResolver:
		p.advance()
		child = target(valueString, segment(lexer.TokenCertResolver))
	case lexer.TokenOptions:
		p.advance()
		child = target(valueString, segment(lexer.TokenOptions))
	case lexer.TokenDomains:
		token := p.advance()
		child, err = p.parseTLSDomainsOption(token)
	default:
		return parsedTarget{}, unsupportedLabel()
	}
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenTLS)), nil
}

func (p *labelParser) parseTLSDomainsOption(domains lexer.Token) (parsedTarget, *ParseError) {
	domainsSegment := ast.TokenSegment(domains)
	if !p.consume(lexer.TokenDot) {
		return target(valueCSV, domainsSegment), nil
	}

	if _, ok := domains.Value.(int); !ok {
		return parsedTarget{}, unsupportedLabel()
	}

	switch p.peek().Type {
	case lexer.TokenMain:
		p.advance()
		return target(valueString, domainsSegment, segment(lexer.TokenMain)), nil
	case lexer.TokenSANs:
		p.advance()
		return target(valueCSV, domainsSegment, segment(lexer.TokenSANs)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}
