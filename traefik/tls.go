package traefik

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/types"
)

func buildRouterTLS(labels map[string]string, routerPrefix string) *dynamic.RouterTLSConfig {
	tlsEnabled, hasTLSLabel := parseBool(labels[routerPrefix+"tls"])
	certResolver := strings.TrimSpace(labels[routerPrefix+"tls.certresolver"])
	options := strings.TrimSpace(labels[routerPrefix+"tls.options"])
	domains := routerTLSDomains(labels, routerPrefix)

	if !tlsEnabled && !hasTLSLabel && certResolver == "" && options == "" && len(domains) == 0 {
		return nil
	}

	return &dynamic.RouterTLSConfig{
		CertResolver: certResolver,
		Options:      options,
		Domains:      domains,
	}
}

func buildTCPRouterTLS(labels map[string]string, routerPrefix string) *dynamic.RouterTCPTLSConfig {
	tlsEnabled, hasTLSLabel := firstBoolLabel(labels, routerPrefix+"tls", "traefik.tcp.tls")
	passthrough, hasPassthrough := firstBoolLabel(labels, routerPrefix+"tls.passthrough", "traefik.tcp.tls.passthrough")
	certResolver := firstLabel(labels, routerPrefix+"tls.certresolver", "traefik.tcp.tls.certresolver")
	options := firstLabel(labels, routerPrefix+"tls.options", "traefik.tcp.tls.options")
	domains := routerTLSDomains(labels, routerPrefix)
	if len(domains) == 0 {
		domains = routerTLSDomains(labels, "traefik.tcp.")
	}

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

func firstBoolLabel(labels map[string]string, keys ...string) (bool, bool) {
	for _, key := range keys {
		raw, exists := labels[key]
		if !exists {
			continue
		}
		return parseBool(raw)
	}
	return false, false
}

func routerTLSDomains(labels map[string]string, routerPrefix string) []types.Domain {
	indexed := indexedTLSDomains(labels, routerPrefix)
	if len(indexed) > 0 {
		return indexed
	}

	rawDomains := splitCSV(labels[routerPrefix+"tls.domains"])
	if len(rawDomains) == 0 {
		return nil
	}

	domains := make([]types.Domain, 0, len(rawDomains))
	for _, domain := range rawDomains {
		domains = append(domains, types.Domain{Main: domain})
	}
	return domains
}

func indexedTLSDomains(labels map[string]string, routerPrefix string) []types.Domain {
	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(routerPrefix) + `tls\.domains\[(\d+)\]\.(main|sans)$`)
	domainMap := make(map[int]*types.Domain)

	for key, value := range labels {
		matches := pattern.FindStringSubmatch(key)
		if matches == nil {
			continue
		}
		index, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if domainMap[index] == nil {
			domainMap[index] = &types.Domain{}
		}
		if matches[2] == "main" {
			domainMap[index].Main = strings.TrimSpace(value)
		} else {
			domainMap[index].SANs = splitCSV(value)
		}
	}

	indices := make([]int, 0, len(domainMap))
	for index := range domainMap {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	domains := make([]types.Domain, 0, len(indices))
	for _, index := range indices {
		domains = append(domains, *domainMap[index])
	}
	return domains
}
