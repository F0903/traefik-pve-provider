package schema

import (
	"strings"

	schematarget "github.com/F0903/traefik-pve-provider/traefik/labels/schema/target"
)

type pathPartKind int

const (
	pathPartLiteral pathPartKind = iota
	pathPartCapture
	pathPartDomainIndex
)

type pathPart struct {
	kind    pathPartKind
	literal string
	capture string
}

func parseMatchPattern(pattern string) []pathPart {
	segments := splitPatternSegments(pattern)
	parts := make([]pathPart, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, parsePatternSegment(segment))
	}
	return parts
}

func splitPatternSegments(pattern string) []string {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" {
		panic("empty label schema pattern")
	}

	segments := make([]string, 0)
	start := 0
	depth := 0
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				panic("unbalanced ']' in label schema pattern " + pattern)
			}
		case '.':
			if depth == 0 {
				segments = appendPatternSegment(segments, pattern[start:index], pattern)
				start = index + 1
			}
		}
	}
	if depth != 0 {
		panic("unbalanced '[' in label schema pattern " + pattern)
	}
	segments = appendPatternSegment(segments, pattern[start:], pattern)
	return segments
}

func appendPatternSegment(segments []string, segment, pattern string) []string {
	if segment == "" {
		panic("empty segment in label schema pattern " + pattern)
	}
	return append(segments, segment)
}

func parsePatternSegment(segment string) pathPart {
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		capture := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		if capture == "" {
			panic("empty capture in label schema pattern segment " + segment)
		}
		return capturePart(capture)
	}
	if strings.HasPrefix(segment, "domains[{") && strings.HasSuffix(segment, "}]") {
		capture := strings.TrimSuffix(strings.TrimPrefix(segment, "domains[{"), "}]")
		if capture == "" {
			panic("empty domain capture in label schema pattern segment " + segment)
		}
		return domainIndexPart(capture)
	}
	return pathPart{kind: pathPartLiteral, literal: segment}
}

func capturePart(capture string) pathPart {
	return pathPart{kind: pathPartCapture, capture: capture}
}

func domainIndexPart(capture string) pathPart {
	return pathPart{kind: pathPartDomainIndex, capture: capture}
}

func captureSets(parts []pathPart) schematarget.CaptureNames {
	captures := schematarget.NewCaptureNames()
	for _, part := range parts {
		switch part.kind {
		case pathPartCapture:
			captures.AddString(part.capture)
		case pathPartDomainIndex:
			captures.AddInt(part.capture)
		}
	}
	return captures
}
