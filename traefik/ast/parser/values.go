package parser

import (
	"strconv"
	"strings"

	"github.com/F0903/traefik-pve-provider/traefik/ast"
)

type labelValueType int

const (
	valueString labelValueType = iota
	valueBool
	valueInt
	valueCSV
)

func parseString(raw string) string {
	return strings.TrimSpace(raw)
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

func parseLabelValue(raw string, valueType labelValueType) (ast.Value, *ParseError) {
	switch valueType {
	case valueBool:
		value, ok := parseBool(raw)
		if !ok {
			return nil, &ParseError{Kind: ErrInvalidBoolean}
		}
		return ast.BoolValue{Value: value}, nil
	case valueInt:
		value, ok := parseInt(raw)
		if !ok {
			return nil, &ParseError{Kind: ErrInvalidInteger}
		}
		return ast.NumberValue{Value: value}, nil
	case valueCSV:
		return ast.ListValue{Values: splitCSV(raw)}, nil
	default:
		return ast.StringValue{Value: parseString(raw)}, nil
	}
}
