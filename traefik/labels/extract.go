package labels

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const DefaultPrefix = "traefik."
const codeFenceMarker = "```"

type ExtractMode string

const (
	ExtractModeFenced ExtractMode = "fenced"
	ExtractModeLoose  ExtractMode = "loose"
	ExtractModeAuto   ExtractMode = "auto"
)

var (
	ErrInvalidMode        = errors.New("invalid extraction mode")
	defaultLabelStartExpr = regexp.MustCompile(`(?i)(^|\s)(traefik\.)`)
	labelKeySuffixExpr    = regexp.MustCompile(`(?i)^[a-z0-9_.\-\[\]]+$`)
)

type ExtractDiagnostic struct {
	Message  string
	Fragment string
}

type ExtractResult struct {
	Labels      map[string]string
	Diagnostics []ExtractDiagnostic
}

type Extractor struct {
	Prefix string
	Mode   ExtractMode
}

func Extract(input string) ExtractResult {
	extractor := Extractor{Prefix: DefaultPrefix, Mode: ExtractModeFenced}
	return extractor.Extract(input)
}

func ParseExtractMode(raw string) (ExtractMode, error) {
	if raw == "" {
		return ExtractModeFenced, nil
	}

	switch mode := ExtractMode(strings.ToLower(strings.TrimSpace(raw))); mode {
	case ExtractModeFenced, ExtractModeLoose, ExtractModeAuto:
		return mode, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidMode, raw)
	}
}

func (e Extractor) Extract(input string) ExtractResult {
	prefix := e.effectivePrefix()
	mode := e.effectiveMode()
	result := newExtractResult()

	switch mode {
	case ExtractModeFenced:
		e.extractFenced(input, prefix, &result)
	case ExtractModeLoose:
		e.extractLoose(input, prefix, &result)
	case ExtractModeAuto:
		if e.extractFenced(input, prefix, &result) {
			return result
		}
		e.extractLoose(input, prefix, &result)
	default:
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  fmt.Sprintf("invalid extraction mode %q", mode),
			Fragment: string(mode),
		})
	}

	return result
}

func (e Extractor) extractFenced(input, prefix string, result *ExtractResult) bool {
	lines := strings.Split(normalizeLineEndings(input), "\n")

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
					e.extractFencedBlock(block, prefix, result)
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
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  "unterminated traefik code fence",
			Fragment: codeFenceMarker + "traefik",
		})
		e.extractFencedBlock(block, prefix, result)
	}

	return seenTraefikFence
}

func (e Extractor) extractFencedBlock(lines []string, prefix string, result *ExtractResult) {
	for _, line := range lines {
		e.extractFencedLine(line, prefix, result)
	}
}

func (e Extractor) extractFencedLine(line, prefix string, result *ExtractResult) {
	fragment := strings.TrimSpace(line)
	if fragment == "" || strings.HasPrefix(fragment, "#") {
		return
	}

	key, value, ok := strings.Cut(fragment, "=")
	if !ok {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  "missing '=' in label",
			Fragment: fragment,
		})
		return
	}

	key = normalizeFencedKey(key, prefix)
	value = normalizeValue(value)
	e.storeLabel(key, value, fragment, prefix, result)
}

func (e Extractor) extractLoose(input, prefix string, result *ExtractResult) {
	for line := range strings.SplitSeq(normalizeLineEndings(input), "\n") {
		e.extractLooseLine(line, prefix, result)
	}
}

func (e Extractor) extractLooseLine(line, prefix string, result *ExtractResult) {
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
			result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
				Message:  "missing '=' in label",
				Fragment: fragment,
			})
			continue
		}

		key = normalizeKey(key)
		value = normalizeValue(value)

		if !strings.HasPrefix(key, prefix) {
			continue
		}
		e.storeLabel(key, value, fragment, prefix, result)
	}
}

func (e Extractor) storeLabel(key, value, fragment, prefix string, result *ExtractResult) {
	if !validLabelKey(key, prefix) {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  fmt.Sprintf("invalid label key %q", key),
			Fragment: fragment,
		})
		return
	}

	if _, exists := result.Labels[key]; exists {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  fmt.Sprintf("duplicate label %q overwritten", key),
			Fragment: fragment,
		})
	}
	result.Labels[key] = value
}

func (e Extractor) effectivePrefix() string {
	prefix := normalizePrefix(e.Prefix)
	if prefix == "" {
		return DefaultPrefix
	}
	return prefix
}

func (e Extractor) effectiveMode() ExtractMode {
	if e.Mode == "" {
		return ExtractModeFenced
	}
	return e.Mode
}

func newExtractResult() ExtractResult {
	return ExtractResult{Labels: make(map[string]string)}
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

func labelStarts(input, prefix string) []int {
	if !strings.EqualFold(prefix, DefaultPrefix) {
		return customLabelStarts(input, prefix)
	}

	matches := defaultLabelStartExpr.FindAllStringSubmatchIndex(input, -1)
	starts := make([]int, 0, len(matches))
	for _, match := range matches {
		starts = append(starts, match[4])
	}
	return starts
}

func customLabelStarts(input, prefix string) []int {
	lowerInput := strings.ToLower(input)
	lowerPrefix := strings.ToLower(prefix)

	var starts []int
	searchFrom := 0
	for {
		idx := strings.Index(lowerInput[searchFrom:], lowerPrefix)
		if idx == -1 {
			break
		}

		start := searchFrom + idx
		if start == 0 || isWhitespace(input[start-1]) {
			starts = append(starts, start)
		}
		searchFrom = start + len(prefix)
	}
	return starts
}

func validLabelKey(key, prefix string) bool {
	if !strings.HasPrefix(key, prefix) {
		return false
	}

	suffix := strings.TrimPrefix(key, prefix)
	return suffix != "" && labelKeySuffixExpr.MatchString(suffix)
}

func isWhitespace(b byte) bool {
	switch b {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
}
