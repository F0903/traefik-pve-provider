package labels

type Resource struct {
	labels     *Set
	protocol   labelProtocol
	collection string
	name       string
	explicit   bool
	values     map[labelPathKey]indexedValue
	headers    map[labelPathKey]map[string]indexedString
	tlsDomains map[labelPathKey]map[int]*indexedTLSDomain
}

func newResource(labels *Set, protocol labelProtocol, collection string, name string) *Resource {
	return &Resource{
		labels:     labels,
		protocol:   protocol,
		collection: collection,
		name:       name,
		values:     make(map[labelPathKey]indexedValue),
		headers:    make(map[labelPathKey]map[string]indexedString),
		tlsDomains: make(map[labelPathKey]map[int]*indexedTLSDomain),
	}
}
