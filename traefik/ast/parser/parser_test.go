package parser

import (
	"testing"

	"github.com/F0903/traefik-pve-provider/traefik/ast"
	"github.com/F0903/traefik-pve-provider/traefik/ast/lexer"
)

func TestParseLabelASTBuildsHTTPServiceAssignment(t *testing.T) {
	node, err := Parse("traefik.http.services.app.loadbalancer.server.port", "8080", Context{
		DefaultName: "default",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assignment := node.(ast.Assignment)
	assertTargetPath(t, assignment.Target,
		wantSegment(lexer.TokenHTTP, ""),
		wantSegment(lexer.TokenServices, ""),
		wantSegment(lexer.TokenIdentifier, "app"),
		wantSegment(lexer.TokenLoadBalancer, ""),
		wantSegment(lexer.TokenServer, ""),
		wantSegment(lexer.TokenPort, ""),
	)
	value := assignment.Value.(ast.NumberValue)
	if value.Value != 8080 {
		t.Fatalf("value = %#v", value.Value)
	}
}

func TestParseLabelASTExpandsHTTPShorthandToDefaultService(t *testing.T) {
	node, err := Parse("traefik.port", "8080", Context{
		DefaultName: "app",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assignment := node.(ast.Assignment)
	assertTargetPath(t, assignment.Target,
		wantSegment(lexer.TokenHTTP, ""),
		wantSegment(lexer.TokenServices, ""),
		wantSegment(lexer.TokenIdentifier, "app"),
		wantSegment(lexer.TokenLoadBalancer, ""),
		wantSegment(lexer.TokenServer, ""),
		wantSegment(lexer.TokenPort, ""),
	)
	value := assignment.Value.(ast.NumberValue)
	if value.Value != 8080 {
		t.Fatalf("value = %#v", value.Value)
	}
}

func TestParseLabelASTExpandsTCPShorthandToDefaultService(t *testing.T) {
	node, err := Parse("traefik.tcp.port", "5432", Context{
		DefaultName: "pg",
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assignment := node.(ast.Assignment)
	assertTargetPath(t, assignment.Target,
		wantSegment(lexer.TokenTCP, ""),
		wantSegment(lexer.TokenServices, ""),
		wantSegment(lexer.TokenIdentifier, "pg"),
		wantSegment(lexer.TokenLoadBalancer, ""),
		wantSegment(lexer.TokenServer, ""),
		wantSegment(lexer.TokenPort, ""),
	)
	value := assignment.Value.(ast.NumberValue)
	if value.Value != 5432 {
		t.Fatalf("value = %#v", value.Value)
	}
}

func TestParseLabelASTRejectsInvalidValues(t *testing.T) {
	_, err := Parse("traefik.port", "eight", Context{
		DefaultName: "app",
	})
	if err == nil {
		t.Fatal("Parse() error = nil")
	}
	if err.Kind != ErrInvalidInteger {
		t.Fatalf("error kind = %v", err.Kind)
	}
}

func TestParseLabelASTRejectsUnsupportedDottedNames(t *testing.T) {
	_, err := Parse("traefik.http.routers.my.app.rule", "Host(`app.example.com`)", Context{
		DefaultName: "app",
	})
	if err == nil {
		t.Fatal("Parse() error = nil")
	}
	if err.Kind != ErrUnsupported {
		t.Fatalf("error kind = %v", err.Kind)
	}
}

type expectedSegment struct {
	tokenType lexer.TokenType
	lexeme    string
}

func wantSegment(tokenType lexer.TokenType, lexeme string) expectedSegment {
	return expectedSegment{tokenType: tokenType, lexeme: lexeme}
}

func assertTargetPath(t *testing.T, target *ast.Target, expected ...expectedSegment) {
	t.Helper()

	segments := target.Segments()
	if len(segments) != len(expected) {
		t.Fatalf("target path length = %d, want %d: %#v", len(segments), len(expected), segments)
	}
	for index, segment := range segments {
		if segment.Type != expected[index].tokenType || segment.Lexeme != expected[index].lexeme {
			t.Fatalf("segment[%d] = %#v, want %#v", index, segment, expected[index])
		}
	}
}
