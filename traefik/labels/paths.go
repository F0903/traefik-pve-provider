package labels

import (
	"sort"

	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func (s *Resource) assignmentsWithPrefix(path ...lexer.TokenType) []labelAssignment {
	if s == nil || s.labels == nil {
		return nil
	}

	matches := make([]labelAssignment, 0)
	for _, assignment := range s.labels.assignments {
		rest, ok := assignment.objectPath(s)
		if !ok || !segmentTypesHavePrefix(rest, path) {
			continue
		}
		matches = append(matches, assignment)
	}
	return matches
}

func (a labelAssignment) objectPath(resource *Resource) ([]ast.Segment, bool) {
	segments := a.assignment.Target.Segments()
	if resource == nil ||
		len(segments) < 3 ||
		segments[0].Type != resource.protocol ||
		segments[1].Type != resource.collection ||
		segments[2].Type != lexer.TokenIdentifier ||
		segments[2].Lexeme != resource.name {
		return nil, false
	}
	return segments[3:], true
}

func segmentTypesEqual(segments []ast.Segment, path []lexer.TokenType) bool {
	if len(segments) != len(path) {
		return false
	}
	return segmentTypesHavePrefix(segments, path)
}

func segmentTypesHavePrefix(segments []ast.Segment, path []lexer.TokenType) bool {
	if len(segments) < len(path) {
		return false
	}
	for index, tokenType := range path {
		if segments[index].Type != tokenType {
			return false
		}
	}
	return true
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
