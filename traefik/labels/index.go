package labels

type labelPathKey string

type indexedValue struct {
	value  any
	origin labelAssignmentOrigin
}

type indexedString struct {
	value  string
	origin labelAssignmentOrigin
}

type indexedTLSDomain struct {
	main       string
	mainOrigin labelAssignmentOrigin
	sans       []string
	sansOrigin labelAssignmentOrigin
}

func putIndexedValue(values map[labelPathKey]indexedValue, key labelPathKey, value any, origin labelAssignmentOrigin) {
	if existing, exists := values[key]; exists && existing.origin > origin {
		return
	}
	values[key] = indexedValue{value: value, origin: origin}
}

func (s *Resource) indexEntry(key labelPathKey, name string, value any, origin labelAssignmentOrigin) {
	if key == "" || name == "" {
		return
	}
	headerValue, ok := value.(string)
	if !ok {
		return
	}

	if s.headers[key] == nil {
		s.headers[key] = make(map[string]indexedString)
	}
	if existing, exists := s.headers[key][name]; exists && existing.origin > origin {
		return
	}
	s.headers[key][name] = indexedString{value: headerValue, origin: origin}
}

func (s *Resource) indexTLSDomain(target labelDomainTarget, value any, origin labelAssignmentOrigin) {
	if target.prefix == "" {
		return
	}

	if s.tlsDomains[target.prefix] == nil {
		s.tlsDomains[target.prefix] = make(map[int]*indexedTLSDomain)
	}
	domain := s.tlsDomains[target.prefix][target.index]
	if domain == nil {
		domain = &indexedTLSDomain{}
		s.tlsDomains[target.prefix][target.index] = domain
	}

	switch target.field {
	case tlsDomainMain:
		main, ok := value.(string)
		if !ok || domain.mainOrigin > origin {
			return
		}
		domain.main = main
		domain.mainOrigin = origin
	case tlsDomainSANs:
		sans, ok := value.([]string)
		if !ok || domain.sansOrigin > origin {
			return
		}
		domain.sans = sans
		domain.sansOrigin = origin
	}
}
