package labels

type Set struct {
	assignments []labelAssignment
	values      map[labelPathKey]indexedValue

	explicitHTTP bool
	explicitTCP  bool
	explicitUDP  bool

	HTTP ProtocolSet
	TCP  ProtocolSet
	UDP  ProtocolSet
}

func newLabelSet() *Set {
	set := &Set{
		values: make(map[labelPathKey]indexedValue),
	}
	set.HTTP = newLabelProtocolSet(set, labelProtocolHTTP)
	set.TCP = newLabelProtocolSet(set, labelProtocolTCP)
	set.UDP = newLabelProtocolSet(set, labelProtocolUDP)
	return set
}

func (s *Set) observeSegments(segments []string) {
	if len(segments) == 0 {
		return
	}

	switch segments[0] {
	case protocolHTTP:
		s.explicitHTTP = true
		s.observeProtocolObject(&s.HTTP, segments)
	case protocolTCP:
		s.explicitTCP = true
		s.observeProtocolObject(&s.TCP, segments)
	case protocolUDP:
		s.explicitUDP = true
		s.observeProtocolObject(&s.UDP, segments)
	}
}

func (s *Set) observeProtocolObject(protocol *ProtocolSet, segments []string) {
	if len(segments) < 3 {
		return
	}
	protocol.markExplicit(segments[1])
}

func (s *Set) Enabled() bool {
	value, ok := s.BoolValue(rootEnable)
	return ok && value
}

func (s *Set) NameOverride() (string, bool) {
	return s.StringValue(rootName)
}

func (s *Set) HasExplicitHTTP() bool {
	return s.explicitHTTP
}

func (s *Set) HasExplicitTCP() bool {
	return s.explicitTCP
}

func (s *Set) HasExplicitUDP() bool {
	return s.explicitUDP
}

func (s *Set) applyAssignment(assignment labelAssignment) {
	s.assignments = append(s.assignments, assignment)
	s.indexAssignment(assignment)
}

func (s *Set) indexAssignment(assignment labelAssignment) {
	target := assignment.target
	if !target.resource {
		putIndexedValue(s.values, target.key, assignment.value, assignment.origin)
		return
	}

	resource := s.resourceForTarget(target, assignment.origin)
	if resource == nil {
		return
	}

	putIndexedValue(resource.values, target.key, assignment.value, assignment.origin)
	resource.indexEntry(target.key, target.entry, assignment.value, assignment.origin)
	if target.domain != nil {
		resource.indexTLSDomain(*target.domain, assignment.value, assignment.origin)
	}
}

func (s *Set) applyNameOverride(defaultName string) {
	name, ok := s.NameOverride()
	if !ok || name == defaultName {
		return
	}

	for index := range s.assignments {
		if s.assignments[index].origin != labelAssignmentOriginShorthand {
			continue
		}
		s.assignments[index].target = renamedDefaultTarget(s.assignments[index].target, defaultName, name)
	}
	s.rebuildResourceIndexes()
}

func (s *Set) rebuildResourceIndexes() {
	s.values = make(map[labelPathKey]indexedValue)

	httpRouters := s.HTTP.explicitRouters
	httpServices := s.HTTP.explicitServices
	httpTransports := s.HTTP.explicitServersTransports
	tcpRouters := s.TCP.explicitRouters
	tcpServices := s.TCP.explicitServices
	tcpTransports := s.TCP.explicitServersTransports
	udpRouters := s.UDP.explicitRouters
	udpServices := s.UDP.explicitServices
	udpTransports := s.UDP.explicitServersTransports

	s.HTTP = newLabelProtocolSet(s, labelProtocolHTTP)
	s.HTTP.explicitRouters = httpRouters
	s.HTTP.explicitServices = httpServices
	s.HTTP.explicitServersTransports = httpTransports
	s.TCP = newLabelProtocolSet(s, labelProtocolTCP)
	s.TCP.explicitRouters = tcpRouters
	s.TCP.explicitServices = tcpServices
	s.TCP.explicitServersTransports = tcpTransports
	s.UDP = newLabelProtocolSet(s, labelProtocolUDP)
	s.UDP.explicitRouters = udpRouters
	s.UDP.explicitServices = udpServices
	s.UDP.explicitServersTransports = udpTransports

	for _, assignment := range s.assignments {
		s.indexAssignment(assignment)
	}
}

func (s *Set) resourceForTarget(target labelTarget, origin labelAssignmentOrigin) *Resource {
	if !target.resource || target.name == "" {
		return nil
	}

	protocolSet := s.protocolSet(target.protocol)
	switch target.collection {
	case collectionRouters:
		return protocolSet.router(target.name, origin)
	case collectionServices:
		return protocolSet.service(target.name, origin)
	case collectionServersTransports:
		return protocolSet.serversTransport(target.name, origin)
	default:
		return nil
	}
}

func protocolForPath(path string) (labelProtocol, bool) {
	switch path {
	case protocolTCP:
		return labelProtocolTCP, true
	case protocolUDP:
		return labelProtocolUDP, true
	case protocolHTTP:
		return labelProtocolHTTP, true
	default:
		return 0, false
	}
}

func renamedDefaultTarget(target labelTarget, from, to string) labelTarget {
	if !target.resource || target.name != from {
		return target
	}
	target.name = to
	return target
}
