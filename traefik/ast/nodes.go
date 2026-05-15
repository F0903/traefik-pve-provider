package ast

import "github.com/F0903/traefik-pve-provider/traefik/ast/lexer"

type Node interface {
	node()
}

type Assignment struct {
	Target *Target
	Value  Value
}

type Target struct {
	Segment Segment
	Next    *Target
}

type Segment struct {
	Type   lexer.TokenType
	Lexeme string
	Value  any
}

type Value interface {
	valueNode()
}

type StringValue struct {
	Value string
}

type BoolValue struct {
	Value bool
}

type NumberValue struct {
	Value int
}

type ListValue struct {
	Values []string
}

func NewTarget(segments ...Segment) *Target {
	if len(segments) == 0 {
		return nil
	}
	return &Target{
		Segment: segments[0],
		Next:    NewTarget(segments[1:]...),
	}
}

func (t *Target) Segments() []Segment {
	if t == nil {
		return nil
	}
	segments := make([]Segment, 0)
	for current := t; current != nil; current = current.Next {
		segments = append(segments, current.Segment)
	}
	return segments
}

func TokenSegment(token lexer.Token) Segment {
	return Segment{
		Type:   token.Type,
		Lexeme: token.Lexeme,
		Value:  token.Value,
	}
}

func SegmentFor(tokenType lexer.TokenType) Segment {
	return Segment{Type: tokenType}
}

func Identifier(name string) Segment {
	return Segment{
		Type:   lexer.TokenIdentifier,
		Lexeme: name,
	}
}

func (Assignment) node() {}

func (StringValue) valueNode() {}
func (BoolValue) valueNode()   {}
func (NumberValue) valueNode() {}
func (ListValue) valueNode()   {}
