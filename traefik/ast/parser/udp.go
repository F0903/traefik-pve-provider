package parser

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (p *labelParser) parseUDP(context Context) (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}

	var child parsedTarget
	var err *ParseError
	switch p.peek().Type {
	case lexer.TokenRouters:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenRouters, p.parseUDPRouterOption)
	case lexer.TokenServices:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenServices, p.parseUDPServiceOption)
	default:
		child, err = p.parseUDPShorthand(context.DefaultName)
	}
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenUDP)), nil
}

func (p *labelParser) parseUDPServiceOption() (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenLoadBalancer) || !p.consume(lexer.TokenDot) || !p.consume(lexer.TokenServer) || !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}
	child, err := p.parseServerOption(false)
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenLoadBalancer), segment(lexer.TokenServer)), nil
}

func (p *labelParser) parseUDPRouterOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenEntryPoints:
		p.advance()
		return target(valueCSV, segment(lexer.TokenEntryPoints)), nil
	case lexer.TokenEntryPoint:
		p.advance()
		return target(valueString, segment(lexer.TokenEntryPoint)), nil
	case lexer.TokenService:
		p.advance()
		return target(valueString, segment(lexer.TokenService)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseUDPShorthand(defaultName string) (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenService, lexer.TokenEntryPoints, lexer.TokenEntryPoint:
		child, err := p.parseUDPRouterOption()
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenRouters), ast.Identifier(defaultName)), nil
	case lexer.TokenAddress, lexer.TokenIP, lexer.TokenPort:
		child, err := p.parseServerOption(false)
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(
			segment(lexer.TokenServices),
			ast.Identifier(defaultName),
			segment(lexer.TokenLoadBalancer),
			segment(lexer.TokenServer),
		), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}
