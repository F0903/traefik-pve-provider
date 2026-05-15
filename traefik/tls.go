package traefik

import (
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/types"
)

func buildRouterTLS(source *labelcfg.Resource) *dynamic.RouterTLSConfig {
	if source == nil {
		return nil
	}

	tlsEnabled, hasTLSLabel := source.BoolValue(lexer.TokenTLS)
	certResolver, _ := source.StringValue(lexer.TokenTLS, lexer.TokenCertResolver)
	options, _ := source.StringValue(lexer.TokenTLS, lexer.TokenOptions)
	domains := routerTLSDomains(source, lexer.TokenTLS)

	if !tlsEnabled && !hasTLSLabel && certResolver == "" && options == "" && len(domains) == 0 {
		return nil
	}

	return &dynamic.RouterTLSConfig{
		CertResolver: certResolver,
		Options:      options,
		Domains:      domains,
	}
}

func buildTCPRouterTLS(source *labelcfg.Resource) *dynamic.RouterTCPTLSConfig {
	if source == nil {
		return nil
	}

	tlsEnabled, hasTLSLabel := source.BoolValue(lexer.TokenTLS)
	passthrough, hasPassthrough := source.BoolValue(lexer.TokenTLS, lexer.TokenPassthrough)
	certResolver, _ := source.StringValue(lexer.TokenTLS, lexer.TokenCertResolver)
	options, _ := source.StringValue(lexer.TokenTLS, lexer.TokenOptions)
	domains := routerTLSDomains(source, lexer.TokenTLS)

	if !tlsEnabled && !hasTLSLabel && !hasPassthrough && certResolver == "" && options == "" && len(domains) == 0 {
		return nil
	}

	return &dynamic.RouterTCPTLSConfig{
		Passthrough:  passthrough,
		CertResolver: certResolver,
		Options:      options,
		Domains:      domains,
	}
}

func routerTLSDomains(source *labelcfg.Resource, path ...lexer.TokenType) []types.Domain {
	if source == nil {
		return nil
	}

	indexed := indexedTLSDomains(source, path...)
	if len(indexed) > 0 {
		return indexed
	}

	domainPath := append(append([]lexer.TokenType{}, path...), lexer.TokenDomains)
	rawDomains, ok := source.ListValue(domainPath...)
	if !ok {
		return nil
	}

	domains := make([]types.Domain, 0, len(rawDomains))
	for _, domain := range rawDomains {
		domains = append(domains, types.Domain{Main: domain})
	}
	return domains
}

func indexedTLSDomains(source *labelcfg.Resource, path ...lexer.TokenType) []types.Domain {
	labelDomains := source.TLSDomains(path...)
	domains := make([]types.Domain, 0, len(labelDomains))
	for _, source := range labelDomains {
		domains = append(domains, types.Domain{
			Main: source.Main,
			SANs: source.SANs,
		})
	}
	return domains
}
