package schema

import (
	"strings"

	schematarget "github.com/F0903/traefik-pve-provider/traefik/labels/schema/target"
)

type ValueKind int

const (
	ValueString ValueKind = iota
	ValueBool
	ValueInt
	ValueCSV
)

type Origin int

const (
	OriginShorthand Origin = iota
	OriginExplicit
)

type Match = schematarget.Match
type Context = schematarget.Context
type Target = schematarget.Target
type DomainTarget = schematarget.Domain

type Spec struct {
	pattern []pathPart
	value   ValueKind
	origin  Origin
	target  func(Match, Context) Target
}

func label(row string, kind ValueKind) Spec {
	pattern, targetPattern, origin := parseSchemaRow(row)
	return schemaSpec(pattern, targetPattern, kind, origin)
}

func schemaSpec(pattern, targetPattern string, kind ValueKind, origin Origin) Spec {
	patternParts := parseMatchPattern(pattern)

	return Spec{
		pattern: patternParts,
		value:   kind,
		origin:  origin,
		target:  schematarget.Compile(targetPattern, captureSets(patternParts)),
	}
}

func parseSchemaRow(row string) (string, string, Origin) {
	if strings.Count(row, "->") > 1 {
		panic("label schema row has multiple mappings: " + row)
	}

	pattern, target, ok := strings.Cut(row, "->")
	if !ok {
		pattern = strings.TrimSpace(row)
		if pattern == "" {
			panic("empty label schema row")
		}
		return pattern, pattern, OriginExplicit
	}

	pattern = strings.TrimSpace(pattern)
	target = strings.TrimSpace(target)
	if pattern == "" || target == "" {
		panic("label shorthand has empty source or target: " + row)
	}
	return pattern, target, OriginShorthand
}

func (s Spec) Match(segments []string) (Match, bool) {
	if len(segments) != len(s.pattern) {
		return Match{}, false
	}

	match := schematarget.NewMatch()
	for index, part := range s.pattern {
		segment := segments[index]
		if segment == "" {
			return Match{}, false
		}

		switch part.kind {
		case pathPartLiteral:
			if segment != part.literal {
				return Match{}, false
			}
		case pathPartCapture:
			match.SetString(part.capture, segment)
		case pathPartDomainIndex:
			domainIndex, ok := schematarget.ParseDomainIndex(segment)
			if !ok {
				return Match{}, false
			}
			match.SetInt(part.capture, domainIndex)
		}
	}
	return match, true
}

func (s Spec) Value() ValueKind {
	return s.value
}

func (s Spec) Origin() Origin {
	return s.origin
}

func (s Spec) Target(match Match, context Context) Target {
	return s.target(match, context)
}
