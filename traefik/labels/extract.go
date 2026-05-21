package labels

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const DefaultPrefix = "traefik."
const ProviderPrefix = "pve."
const codeFenceMarker = "```"

type ExtractMode string

const (
	ExtractModeFenced ExtractMode = "fenced"
	ExtractModeLoose  ExtractMode = "loose"
	ExtractModeAuto   ExtractMode = "auto"
)

var (
	ErrInvalidMode     = errors.New("invalid extraction mode")
	labelKeySuffixExpr = regexp.MustCompile(`(?i)^[a-z0-9_.\-\[\]]+$`)
)

type ExtractDiagnostic struct {
	Message  string
	Line     int
	Fragment string
}

type LabelSource struct {
	Line     int
	Fragment string
}

type ExtractResult struct {
	Labels      map[string]string
	Sources     map[string]LabelSource
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
	fenceStartLine := 0
	block := make([]sourceLine, 0)

	for index, line := range lines {
		lineNumber := index + 1
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, codeFenceMarker); ok {
			info := strings.TrimSpace(after)
			if inFence {
				if inTraefikFence {
					e.extractFencedBlock(block, prefix, result)
					return true
				}
				inFence = false
				inTraefikFence = false
				continue
			}

			inFence = true
			inTraefikFence = strings.EqualFold(info, "traefik")
			if inTraefikFence {
				seenTraefikFence = true
				fenceStartLine = lineNumber
			}
			continue
		}

		if inTraefikFence {
			block = append(block, sourceLine{
				Number: lineNumber,
				Text:   line,
			})
		}
	}

	if inTraefikFence {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  "unterminated traefik code fence",
			Line:     fenceStartLine,
			Fragment: codeFenceMarker + "traefik",
		})
		e.extractFencedBlock(block, prefix, result)
	}

	return seenTraefikFence
}

type sourceLine struct {
	Number int
	Text   string
}

func (e Extractor) extractFencedBlock(lines []sourceLine, prefix string, result *ExtractResult) {
	for _, line := range lines {
		e.extractFencedLine(line, prefix, result)
	}
}

func (e Extractor) extractFencedLine(line sourceLine, prefix string, result *ExtractResult) {
	fragment := strings.TrimSpace(line.Text)
	if fragment == "" || strings.HasPrefix(fragment, "#") {
		return
	}

	key, value, ok := strings.Cut(fragment, "=")
	if !ok {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  "missing '=' in label",
			Line:     line.Number,
			Fragment: fragment,
		})
		return
	}

	key = normalizeFencedKey(key, prefix)
	value = normalizeValue(value)
	e.storeLabel(key, value, fragment, line.Number, prefix, result)
}

func (e Extractor) extractLoose(input, prefix string, result *ExtractResult) {
	for index, line := range strings.Split(normalizeLineEndings(input), "\n") {
		e.extractLooseLine(line, index+1, prefix, result)
	}
}

func (e Extractor) extractLooseLine(line string, lineNumber int, prefix string, result *ExtractResult) {
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
				Line:     lineNumber,
				Fragment: fragment,
			})
			continue
		}

		key = normalizeKey(key)
		value = normalizeValue(value)

		if !strings.HasPrefix(key, prefix) && !strings.HasPrefix(key, ProviderPrefix) {
			continue
		}
		e.storeLabel(key, value, fragment, lineNumber, prefix, result)
	}
}

func (e Extractor) storeLabel(key, value, fragment string, lineNumber int, prefix string, result *ExtractResult) {
	if !validLabelKey(key, prefix) {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  fmt.Sprintf("invalid label key %q", key),
			Line:     lineNumber,
			Fragment: fragment,
		})
		return
	}

	if _, exists := result.Labels[key]; exists {
		result.Diagnostics = append(result.Diagnostics, ExtractDiagnostic{
			Message:  fmt.Sprintf("duplicate label %q overwritten", key),
			Line:     lineNumber,
			Fragment: fragment,
		})
	}
	result.Labels[key] = value
	result.Sources[key] = LabelSource{
		Line:     lineNumber,
		Fragment: fragment,
	}
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
	return ExtractResult{
		Labels:  make(map[string]string),
		Sources: make(map[string]LabelSource),
	}
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
	if strings.HasPrefix(key, prefix) || strings.HasPrefix(key, ProviderPrefix) {
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
	starts := customLabelStarts(input, prefix)
	if !strings.EqualFold(normalizePrefix(prefix), ProviderPrefix) {
		starts = append(starts, customLabelStarts(input, ProviderPrefix)...)
	}
	if len(starts) < 2 {
		return starts
	}

	sort.Ints(starts)
	unique := starts[:0]
	last := -1
	for _, start := range starts {
		if start == last {
			continue
		}
		unique = append(unique, start)
		last = start
	}
	return unique
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
	if strings.HasPrefix(key, ProviderPrefix) {
		suffix := strings.TrimPrefix(key, ProviderPrefix)
		return suffix != "" && labelKeySuffixExpr.MatchString(suffix)
	}
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
