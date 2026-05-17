package labels

import (
	"slices"
	"strconv"
	"strings"

	labelschema "github.com/F0903/traefik-pve-provider/traefik/labels/schema"
)

type ParseErrorKind int

const (
	ErrUnsupported ParseErrorKind = iota
	ErrInvalidBoolean
	ErrInvalidInteger
)

type ParseError struct {
	Kind ParseErrorKind
}

func (e *ParseError) Error() string {
	switch e.Kind {
	case ErrInvalidBoolean:
		return "invalid boolean"
	case ErrInvalidInteger:
		return "invalid integer"
	default:
		return "unsupported label"
	}
}

type Diagnostic struct {
	Key    string
	Value  string
	Err    *ParseError
	Source LabelSource
}

func Enabled(labels map[string]string) bool {
	keys := sortedKeys(labels)
	for _, key := range keys {
		segments, ok := labelSegments(key)
		if !ok || len(segments) != 1 || segments[0] != rootEnable {
			continue
		}
		enabled, ok := parseBool(labels[key])
		return ok && enabled
	}
	return false
}

func Parse(labels map[string]string, defaultName string) (*Set, []Diagnostic) {
	return ParseWithSources(labels, defaultName, nil)
}

func ParseWithSources(labels map[string]string, defaultName string, sources map[string]LabelSource) (*Set, []Diagnostic) {
	set := newLabelSet()
	context := labelschema.Context{DefaultName: defaultName}
	specs := labelschema.Rows()
	diagnostics := make([]Diagnostic, 0)

	for _, key := range sortedKeys(labels) {
		value := labels[key]
		segments, ok := labelSegments(key)
		if !ok {
			continue
		}
		set.observeSegments(segments)

		assignment, err := parseAssignment(specs, segments, value, context)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Key:    key,
				Value:  value,
				Err:    err,
				Source: sources[key],
			})
			continue
		}
		set.applyAssignment(assignment)
	}

	set.applyNameOverride(defaultName)
	return set, diagnostics
}

func sortedKeys(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func labelSegments(key string) ([]string, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	const prefix = "traefik."
	if !strings.HasPrefix(key, prefix) {
		return nil, false
	}

	rest := strings.TrimPrefix(key, prefix)
	if rest == "" {
		return nil, true
	}
	return strings.Split(rest, "."), true
}

func parseAssignment(
	specs []labelschema.Spec,
	segments []string,
	rawValue string,
	context labelschema.Context,
) (labelAssignment, *ParseError) {
	for _, spec := range specs {
		match, ok := spec.Match(segments)
		if !ok {
			continue
		}

		value, err := parseValue(rawValue, spec.Value())
		if err != nil {
			return labelAssignment{}, err
		}
		return labelAssignment{
			target: labelTargetFromSchema(spec.Target(match, context)),
			value:  value,
			origin: assignmentOriginFromSchema(spec.Origin()),
		}, nil
	}
	return labelAssignment{}, unsupportedLabel()
}

func parseValue(raw string, kind labelschema.ValueKind) (any, *ParseError) {
	switch kind {
	case labelschema.ValueBool:
		value, ok := parseBool(raw)
		if !ok {
			return nil, &ParseError{Kind: ErrInvalidBoolean}
		}
		return value, nil
	case labelschema.ValueInt:
		value, ok := parseInt(raw)
		if !ok {
			return nil, &ParseError{Kind: ErrInvalidInteger}
		}
		return value, nil
	case labelschema.ValueCSV:
		return splitCSV(raw), nil
	default:
		return strings.TrimSpace(raw), nil
	}
}

func assignmentOriginFromSchema(origin labelschema.Origin) labelAssignmentOrigin {
	if origin == labelschema.OriginExplicit {
		return labelAssignmentOriginExplicit
	}
	return labelAssignmentOriginShorthand
}

func labelTargetFromSchema(target labelschema.Target) labelTarget {
	converted := labelTarget{
		key:        labelKey(target.Key),
		collection: target.Collection,
		name:       target.Name,
		entry:      target.Entry,
		resource:   target.Resource,
	}
	if target.Resource {
		converted.protocol, _ = protocolForPath(target.Protocol)
	}
	if target.Domain != nil {
		converted.domain = &labelDomainTarget{
			prefix: labelKey(target.Domain.Prefix),
			index:  target.Domain.Index,
			field:  target.Domain.Field,
		}
	}
	return converted
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func parseBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func parseInt(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil
}

func unsupportedLabel() *ParseError {
	return &ParseError{Kind: ErrUnsupported}
}
