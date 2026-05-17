package target

import (
	"regexp"
	"strconv"
	"strings"
)

type partKind int

const (
	partLiteral partKind = iota
	partCapture
	partDomainIndex
)

type part struct {
	kind    partKind
	literal string
	capture string
}

type targetPart struct {
	kind    targetPartKind
	literal string
	capture string
	key     []part
}

type targetPartKind int

const (
	targetPartLiteral targetPartKind = iota
	targetPartCapture
	targetPartDomainIndex
	targetPartKey
)

func parsePattern(pattern string) []targetPart {
	segments := splitPatternSegments(pattern)
	parts := make([]targetPart, 0, len(segments))
	for _, segment := range segments {
		part := parsePatternSegment(segment)
		switch part.kind {
		case partLiteral:
			parts = append(parts, targetPart{kind: targetPartLiteral, literal: part.literal})
		case partCapture:
			parts = append(parts, targetPart{kind: targetPartCapture, capture: part.capture})
		case partDomainIndex:
			parts = append(parts, targetPart{kind: targetPartDomainIndex, capture: part.capture})
		}
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

func parsePatternSegment(segment string) part {
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		capture := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		if capture == "" {
			panic("empty capture in label schema pattern segment " + segment)
		}
		return part{kind: partCapture, capture: capture}
	}
	if strings.HasPrefix(segment, "domains[{") && strings.HasSuffix(segment, "}]") {
		capture := strings.TrimSuffix(strings.TrimPrefix(segment, "domains[{"), "}]")
		if capture == "" {
			panic("empty domain capture in label schema pattern segment " + segment)
		}
		return part{kind: partDomainIndex, capture: capture}
	}
	return part{kind: partLiteral, literal: segment}
}

var domainIndexPattern = regexp.MustCompile(`^domains\[(\d+)\]$`)

func ParseDomainIndex(segment string) (int, bool) {
	matches := domainIndexPattern.FindStringSubmatch(segment)
	if matches == nil {
		return 0, false
	}
	index, err := strconv.Atoi(matches[1])
	return index, err == nil
}
