package parser

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (p *labelParser) parseHTTP(context Context) (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}

	var child parsedTarget
	var err *ParseError
	switch p.peek().Type {
	case lexer.TokenRouters:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenRouters, p.parseHTTPRouterOption)
	case lexer.TokenServices:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenServices, p.parseHTTPServiceOption)
	case lexer.TokenServersTransports:
		p.advance()
		child, err = p.parseNamedObject(lexer.TokenServersTransports, p.parseHTTPServersTransportOption)
	default:
		return parsedTarget{}, unsupportedLabel()
	}
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenHTTP)), nil
}

func (p *labelParser) parseHTTPRouterOption() (parsedTarget, *ParseError) {
	return p.parseRouterOption(false)
}

func (p *labelParser) parseHTTPServiceOption() (parsedTarget, *ParseError) {
	if !p.consume(lexer.TokenLoadBalancer) || !p.consume(lexer.TokenDot) {
		return parsedTarget{}, unsupportedLabel()
	}
	child, err := p.parseHTTPLoadBalancerOption()
	if err != nil {
		return parsedTarget{}, err
	}
	return child.prepend(segment(lexer.TokenLoadBalancer)), nil
}

func (p *labelParser) parseHTTPLoadBalancerOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenServer:
		p.advance()
		if !p.consume(lexer.TokenDot) {
			return parsedTarget{}, unsupportedLabel()
		}
		child, err := p.parseServerOption(true)
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenServer)), nil
	case lexer.TokenInsecureSkipVerify:
		p.advance()
		return target(valueBool, segment(lexer.TokenInsecureSkipVerify)), nil
	case lexer.TokenPassHostHeader:
		p.advance()
		return target(valueBool, segment(lexer.TokenPassHostHeader)), nil
	case lexer.TokenHealthCheck:
		p.advance()
		if !p.consume(lexer.TokenDot) {
			return parsedTarget{}, unsupportedLabel()
		}
		child, err := p.parseHealthCheckOption()
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenHealthCheck)), nil
	case lexer.TokenSticky:
		p.advance()
		if !p.consume(lexer.TokenDot) || !p.consume(lexer.TokenCookie) || !p.consume(lexer.TokenDot) {
			return parsedTarget{}, unsupportedLabel()
		}
		child, err := p.parseCookieOption()
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenSticky), segment(lexer.TokenCookie)), nil
	case lexer.TokenResponseForwarding:
		p.advance()
		if !p.consume(lexer.TokenDot) || !p.consume(lexer.TokenFlushInterval) {
			return parsedTarget{}, unsupportedLabel()
		}
		return target(valueString, segment(lexer.TokenResponseForwarding), segment(lexer.TokenFlushInterval)), nil
	case lexer.TokenServersTransport:
		p.advance()
		return target(valueString, segment(lexer.TokenServersTransport)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseHTTPServersTransportOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenServerName:
		p.advance()
		return target(valueString, segment(lexer.TokenServerName)), nil
	case lexer.TokenInsecureSkipVerify:
		p.advance()
		return target(valueBool, segment(lexer.TokenInsecureSkipVerify)), nil
	case lexer.TokenRootCAs:
		p.advance()
		return target(valueCSV, segment(lexer.TokenRootCAs)), nil
	case lexer.TokenMaxIdleConnsPerHost:
		p.advance()
		return target(valueInt, segment(lexer.TokenMaxIdleConnsPerHost)), nil
	case lexer.TokenDisableHTTP2:
		p.advance()
		return target(valueBool, segment(lexer.TokenDisableHTTP2)), nil
	case lexer.TokenPeerCertURI:
		p.advance()
		return target(valueString, segment(lexer.TokenPeerCertURI)), nil
	case lexer.TokenForwardingTimeouts:
		p.advance()
		if !p.consume(lexer.TokenDot) {
			return parsedTarget{}, unsupportedLabel()
		}
		child, err := p.parseForwardingTimeoutsOption()
		if err != nil {
			return parsedTarget{}, err
		}
		return child.prepend(segment(lexer.TokenForwardingTimeouts)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseForwardingTimeoutsOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenDialTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenDialTimeout)), nil
	case lexer.TokenResponseHeaderTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenResponseHeaderTimeout)), nil
	case lexer.TokenIdleConnTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenIdleConnTimeout)), nil
	case lexer.TokenReadIdleTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenReadIdleTimeout)), nil
	case lexer.TokenPingTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenPingTimeout)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseHealthCheckOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenPath:
		p.advance()
		return target(valueString, segment(lexer.TokenPath)), nil
	case lexer.TokenInterval:
		p.advance()
		return target(valueString, segment(lexer.TokenInterval)), nil
	case lexer.TokenTimeout:
		p.advance()
		return target(valueString, segment(lexer.TokenTimeout)), nil
	case lexer.TokenScheme:
		p.advance()
		return target(valueString, segment(lexer.TokenScheme)), nil
	case lexer.TokenMethod:
		p.advance()
		return target(valueString, segment(lexer.TokenMethod)), nil
	case lexer.TokenHostname:
		p.advance()
		return target(valueString, segment(lexer.TokenHostname)), nil
	case lexer.TokenPort:
		p.advance()
		return target(valueInt, segment(lexer.TokenPort)), nil
	case lexer.TokenFollowRedirects:
		p.advance()
		return target(valueBool, segment(lexer.TokenFollowRedirects)), nil
	case lexer.TokenHeaders:
		p.advance()
		if !p.consume(lexer.TokenDot) || !lexer.IsNameToken(p.peek()) {
			return parsedTarget{}, unsupportedLabel()
		}
		name := p.advance().Lexeme
		return target(valueString, segment(lexer.TokenHeaders), ast.Identifier(name)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) parseCookieOption() (parsedTarget, *ParseError) {
	switch p.peek().Type {
	case lexer.TokenName:
		p.advance()
		return target(valueString, segment(lexer.TokenName)), nil
	case lexer.TokenSecure:
		p.advance()
		return target(valueBool, segment(lexer.TokenSecure)), nil
	case lexer.TokenHTTPOnly:
		p.advance()
		return target(valueBool, segment(lexer.TokenHTTPOnly)), nil
	case lexer.TokenSameSite:
		p.advance()
		return target(valueString, segment(lexer.TokenSameSite)), nil
	default:
		return parsedTarget{}, unsupportedLabel()
	}
}

func (p *labelParser) httpDefaultRouter(defaultName string, valueType labelValueType, suffix ...ast.Segment) parsedTarget {
	return target(valueType, suffix...).prepend(
		segment(lexer.TokenHTTP),
		segment(lexer.TokenRouters),
		ast.Identifier(defaultName),
	)
}

func (p *labelParser) httpDefaultService(defaultName string, valueType labelValueType, suffix ...ast.Segment) parsedTarget {
	return target(valueType, suffix...).prepend(
		segment(lexer.TokenHTTP),
		segment(lexer.TokenServices),
		ast.Identifier(defaultName),
		segment(lexer.TokenLoadBalancer),
	)
}
