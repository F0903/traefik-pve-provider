package labels

type labelAssignmentOrigin int

const (
	labelAssignmentOriginShorthand labelAssignmentOrigin = iota
	labelAssignmentOriginExplicit
)

type labelAssignment struct {
	target labelTarget
	value  any
	origin labelAssignmentOrigin
}

type labelTarget struct {
	key        labelPathKey
	protocol   labelProtocol
	collection string
	name       string
	entry      string
	domain     *labelDomainTarget
	resource   bool
}

type labelDomainTarget struct {
	prefix labelPathKey
	index  int
	field  string
}
