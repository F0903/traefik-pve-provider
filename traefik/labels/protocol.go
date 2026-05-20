package labels

import "sort"

type labelProtocol int

const (
	labelProtocolHTTP labelProtocol = iota
	labelProtocolTCP
	labelProtocolUDP
)

type ProtocolSet struct {
	protocol labelProtocol

	Routers           map[string]*Resource
	Services          map[string]*Resource
	ServersTransports map[string]*Resource

	explicitRouters           bool
	explicitServices          bool
	explicitServersTransports bool
}

func newLabelProtocolSet(protocol labelProtocol) ProtocolSet {
	return ProtocolSet{
		protocol:          protocol,
		Routers:           make(map[string]*Resource),
		Services:          make(map[string]*Resource),
		ServersTransports: make(map[string]*Resource),
	}
}

func (s *Set) protocolSet(protocol labelProtocol) *ProtocolSet {
	switch protocol {
	case labelProtocolTCP:
		return &s.TCP
	case labelProtocolUDP:
		return &s.UDP
	default:
		return &s.HTTP
	}
}

func (s *ProtocolSet) markExplicit(collection string) {
	switch collection {
	case collectionRouters:
		s.explicitRouters = true
	case collectionServices:
		s.explicitServices = true
	case collectionServersTransports:
		s.explicitServersTransports = true
	}
}

func (s ProtocolSet) RouterNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.Routers), s.explicitRouters
}

func (s ProtocolSet) ServiceNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.Services), s.explicitServices
}

func (s ProtocolSet) ServersTransportNames() ([]string, bool) {
	return sortedExplicitLabelResourceNames(s.ServersTransports), s.explicitServersTransports
}

func (s *ProtocolSet) router(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.Routers, collectionRouters, name, origin)
}

func (s *ProtocolSet) service(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.Services, collectionServices, name, origin)
}

func (s *ProtocolSet) serversTransport(name string, origin labelAssignmentOrigin) *Resource {
	return s.namedResource(s.ServersTransports, collectionServersTransports, name, origin)
}

func (s *ProtocolSet) namedResource(objects map[string]*Resource, collection string, name string, origin labelAssignmentOrigin) *Resource {
	resource := objects[name]
	if resource == nil {
		resource = newResource(s.protocol, collection, name)
		objects[name] = resource
	}
	if origin == labelAssignmentOriginExplicit {
		resource.explicit = true
	}
	return resource
}

func sortedExplicitLabelResourceNames(objects map[string]*Resource) []string {
	names := make([]string, 0, len(objects))
	for name, resource := range objects {
		if resource.explicit {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
