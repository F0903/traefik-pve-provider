package metadata

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const DefaultPrefix = "traefik."
const codeFenceMarker = "```"

type Mode string

const (
	ModeFenced Mode = "fenced"
	ModeLoose  Mode = "loose"
	ModeAuto   Mode = "auto"
)

var (
	ErrInvalidMode   = errors.New("invalid metadata mode")
	labelStartRegexp = regexp.MustCompile(`(?i)(^|\s)(traefik\.)`)
	labelKeyRegexp   = regexp.MustCompile(`(?i)^traefik\.[a-z0-9_.\-\[\]]+$`)
)

type Diagnostic struct {
	Message  string
	Fragment string
}

type ParseResult struct {
	Labels      map[string]string
	Diagnostics []Diagnostic
}

type Parser struct {
	Prefix string
	Mode   Mode
}

func ParseNotes(notes string) ParseResult {
	parser := Parser{Prefix: DefaultPrefix, Mode: ModeFenced}
	return parser.Parse(notes)
}

func ParseMode(raw string) (Mode, error) {
	if raw == "" {
		return ModeFenced, nil
	}

	switch mode := Mode(strings.ToLower(strings.TrimSpace(raw))); mode {
	case ModeFenced, ModeLoose, ModeAuto:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidMode, raw)
	}
}

func (p Parser) Parse(notes string) ParseResult {
	prefix := p.effectivePrefix()
	mode := p.effectiveMode()
	result := newParseResult()

	switch mode {
	case ModeFenced:
		p.parseFenced(notes, prefix, &result)
	case ModeLoose:
		p.parseLoose(notes, prefix, &result)
	case ModeAuto:
		if p.parseFenced(notes, prefix, &result) {
			return result
		}
		p.parseLoose(notes, prefix, &result)
	default:
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Message:  fmt.Sprintf("invalid metadata mode %q", mode),
			Fragment: string(mode),
		})
	}

	return result
}

func (p Parser) parseFenced(notes, prefix string, result *ParseResult) bool {
	lines := strings.Split(normalizeLineEndings(notes), "\n")

	inFence := false
	inTraefikFence := false
	seenTraefikFence := false
	block := make([]string, 0)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, codeFenceMarker); ok {
			info := strings.TrimSpace(after)
			if inFence {
				if inTraefikFence {
					p.parseFencedBlock(block, prefix, result)
					block = block[:0]
				}
				inFence = false
				inTraefikFence = false
				continue
			}

			inFence = true
			inTraefikFence = strings.EqualFold(info, "traefik")
			if inTraefikFence {
				seenTraefikFence = true
			}
			continue
		}

		if inTraefikFence {
			block = append(block, line)
		}
	}

	if inTraefikFence {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Message:  "unterminated traefik code fence",
			Fragment: codeFenceMarker + "traefik",
		})
		p.parseFencedBlock(block, prefix, result)
	}

	return seenTraefikFence
}

func (p Parser) parseFencedBlock(lines []string, prefix string, result *ParseResult) {
	for _, line := range lines {
		p.parseFencedLine(line, prefix, result)
	}
}

func (p Parser) parseFencedLine(line, prefix string, result *ParseResult) {
	fragment := strings.TrimSpace(line)
	if fragment == "" || strings.HasPrefix(fragment, "#") {
		return
	}

	key, value, ok := strings.Cut(fragment, "=")
	if !ok {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Message:  "missing '=' in label",
			Fragment: fragment,
		})
		return
	}

	key = normalizeFencedKey(key, prefix)
	value = normalizeValue(value)
	p.storeLabel(key, value, fragment, result)
}

func (p Parser) parseLoose(notes, prefix string, result *ParseResult) {
	for line := range strings.SplitSeq(normalizeLineEndings(notes), "\n") {
		p.parseLooseLine(line, prefix, result)
	}
}

func (p Parser) parseLooseLine(line, prefix string, result *ParseResult) {
	starts := labelStarts(line, prefix)
	for i, start := range starts {
		end := len(line)
		if i+1 < len(starts) {
			end = starts[i+1]
		}

		fragment := strings.TrimSpace(line[start:end])
		if fragment == "" {
			continue
		}

		key, value, ok := strings.Cut(fragment, "=")
		if !ok {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Message:  "missing '=' in label",
				Fragment: fragment,
			})
			continue
		}

		key = normalizeKey(key)
		value = normalizeValue(value)

		if !strings.HasPrefix(key, strings.ToLower(prefix)) {
			continue
		}
		p.storeLabel(key, value, fragment, result)
	}
}

func (p Parser) storeLabel(key, value, fragment string, result *ParseResult) {
	if !labelKeyRegexp.MatchString(key) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Message:  fmt.Sprintf("invalid label key %q", key),
			Fragment: fragment,
		})
		return
	}

	if _, exists := result.Labels[key]; exists {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Message:  fmt.Sprintf("duplicate label %q overwritten", key),
			Fragment: fragment,
		})
	}
	result.Labels[key] = value
}

func (p Parser) effectivePrefix() string {
	if p.Prefix == "" {
		return DefaultPrefix
	}
	return p.Prefix
}

func (p Parser) effectiveMode() Mode {
	if p.Mode == "" {
		return ModeFenced
	}
	return p.Mode
}

func newParseResult() ParseResult {
	return ParseResult{Labels: make(map[string]string)}
}

func normalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, `"'`)
	return strings.ToLower(key)
}

func normalizeFencedKey(key, prefix string) string {
	key = normalizeKey(key)
	prefix = normalizePrefix(prefix)
	if strings.HasPrefix(key, prefix) {
		return key
	}
	return prefix + key
}

func normalizePrefix(prefix string) string {
	return strings.ToLower(strings.TrimSpace(prefix))
}

func normalizeValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}

	first := value[0]
	last := value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func labelStarts(notes, prefix string) []int {
	if !strings.EqualFold(prefix, DefaultPrefix) {
		return customLabelStarts(notes, prefix)
	}

	matches := labelStartRegexp.FindAllStringSubmatchIndex(notes, -1)
	starts := make([]int, 0, len(matches))
	for _, match := range matches {
		starts = append(starts, match[4])
	}
	return starts
}

func customLabelStarts(notes, prefix string) []int {
	lowerNotes := strings.ToLower(notes)
	lowerPrefix := strings.ToLower(prefix)

	var starts []int
	searchFrom := 0
	for {
		idx := strings.Index(lowerNotes[searchFrom:], lowerPrefix)
		if idx == -1 {
			break
		}

		start := searchFrom + idx
		if start == 0 || isWhitespace(notes[start-1]) {
			starts = append(starts, start)
		}
		searchFrom = start + len(prefix)
	}
	return starts
}

func isWhitespace(b byte) bool {
	switch b {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}
