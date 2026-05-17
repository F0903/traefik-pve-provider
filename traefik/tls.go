package traefik

import (
	labelcfg "github.com/F0903/traefik-pve-provider/traefik/labels"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/types"
)

func buildRouterTLS(source *labelcfg.Resource) *dynamic.RouterTLSConfig {
	if source == nil {
		return nil
	}

	tlsEnabled, hasTLSLabel := source.BoolValue("tls")
	certResolver, _ := source.StringValue("tls.certresolver")
	options, _ := source.StringValue("tls.options")
	domains := routerTLSDomains(source, "tls")

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

	tlsEnabled, hasTLSLabel := source.BoolValue("tls")
	passthrough, hasPassthrough := source.BoolValue("tls.passthrough")
	certResolver, _ := source.StringValue("tls.certresolver")
	options, _ := source.StringValue("tls.options")
	domains := routerTLSDomains(source, "tls")

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

func routerTLSDomains(source *labelcfg.Resource, key string) []types.Domain {
	if source == nil {
		return nil
	}

	indexed := indexedTLSDomains(source, key)
	if len(indexed) > 0 {
		return indexed
	}

	rawDomains, ok := source.ListValue(key + ".domains")
	if !ok {
		return nil
	}

	domains := make([]types.Domain, 0, len(rawDomains))
	for _, domain := range rawDomains {
		domains = append(domains, types.Domain{Main: domain})
	}
	return domains
}

func indexedTLSDomains(source *labelcfg.Resource, key string) []types.Domain {
	labelDomains := source.TLSDomains(key)
	domains := make([]types.Domain, 0, len(labelDomains))
	for _, source := range labelDomains {
		domains = append(domains, types.Domain{
			Main: source.Main,
			SANs: source.SANs,
		})
	}
	return domains
}
