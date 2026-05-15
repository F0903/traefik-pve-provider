package parser

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (p *labelParser) parseTCP(context Context) (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}

	var child parsedTarget
	var err *ParseError
	switch p.peek().Type {
	case lexer.TokenRouters:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenRouters, p.parseTCPRouterOption)
	case lexer.TokenServices:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenServices, p.parseTCPServiceOption)
	default:
		child, err = p.parseTCPShorthand(context.DefaultName)
	}
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenTCP)), nil
}

func (p *labelParser) parseTCPRouterOption() (parsedTarget, *ParseError) {
	return p.parseRouterOption(true)
}

func (p *labelParser) parseTCPServiceOption() (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenLoadBalancer) || !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}
	child, err := p.parseTCPLoadBalancerOption()
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenLoadBalancer)), nil
}

func (p *labelParser) parseTCPLoadBalancerOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenServer:
		p.advance()
		if !p.consume(lexer.TokenDot) {
			return parsedTarget{}, unsupportedLabel()
		}
		child, err := p.parseServerOption(false)
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenServer)), nil
	case lexer.TokenProxyProtocol:
		p.advance()
		if !p.consume(lexer.TokenDot) || !p.consume(lexer.TokenVersion) {
			return parsedTarget{}, unsupportedLabel()
		}
		return target(valueInt, segment(lexer.TokenProxyProtocol), segment(lexer.TokenVersion)), nil
	case lexer.TokenTerminationDelay:
		p.advance()
		return target(valueInt, segment(lexer.TokenTerminationDelay)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseTCPShorthand(defaultName string) (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenRule, lexer.TokenService, lexer.TokenEntryPoints, lexer.TokenEntryPoint, lexer.TokenMiddlewares, lexer.TokenPriority, lexer.TokenTLS:
		child, err := p.parseTCPRouterOption()
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
	case lexer.TokenProxyProtocol, lexer.TokenTerminationDelay:
		child, err := p.parseTCPLoadBalancerOption()
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenServices), ast.Identifier(defaultName), segment(lexer.TokenLoadBalancer)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}
