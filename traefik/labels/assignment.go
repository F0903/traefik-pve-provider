package labels

import "github.com/F0903/traefik-pve-provider/traefik/ast"

type labelAssignmentOrigin int

const (
	labelAssignmentOriginShorthand labelAssignmentOrigin = iota
	labelAssignmentOriginExplicit
)

type labelAssignment struct {
	assignment ast.Assignment
	origin     labelAssignmentOrigin
}

func assignmentValue(value ast.Value) (any, bool) {
	switch typed := value.(type) {
	case ast.StringValue:
		return typed.Value, true
	case ast.BoolValue:
		return typed.Value, true
	case ast.NumberValue:
		return typed.Value, true
	case ast.ListValue:
		return typed.Values, true
	default:
		return nil, false
	}
}
