package schema

import (
	"testing"

	schematarget "github.com/F0903/traefik-pve-provider/traefik/labels/schema/target"
)

func TestSchemaRowRegistersPlainRowsAsExplicit(t *testing.T) {
	spec := label("http.routers.{router}.rule", ValueString)
	if spec.Origin() != OriginExplicit {
		t.Fatalf("origin = %v, want explicit", spec.Origin())
	}

	match, ok := spec.Match([]string{"http", "routers", "app", "rule"})
	if !ok {
		t.Fatal("schema did not match")
	}

	target := spec.Target(match, Context{DefaultName: "default"})
	if target.Name != "app" {
		t.Fatalf("resource name = %q, want app", target.Name)
	}
	if target.Key != "rule" {
		t.Fatalf("target key = %q, want rule", target.Key)
	}
}

func TestSchemaRowRegistersArrowRowsAsShorthand(t *testing.T) {
	spec := label("tcp.port -> tcp.services.{default}.loadbalancer.server.port", ValueInt)
	if spec.Origin() != OriginShorthand {
		t.Fatalf("origin = %v, want shorthand", spec.Origin())
	}

	match, ok := spec.Match([]string{"tcp", "port"})
	if !ok {
		t.Fatal("schema did not match")
	}

	target := spec.Target(match, Context{DefaultName: "pg"})
	if target.Protocol != schematarget.ProtocolTCP {
		t.Fatalf("protocol = %v, want TCP", target.Protocol)
	}
	if target.Collection != schematarget.CollectionServices {
		t.Fatalf("collection = %q, want services", target.Collection)
	}
	if target.Name != "pg" {
		t.Fatalf("resource name = %q, want pg", target.Name)
	}
	if target.Key != "loadbalancer.server.port" {
		t.Fatalf("target key = %q, want loadbalancer.server.port", target.Key)
	}
}

func TestSchemaCapturesAreGeneric(t *testing.T) {
	spec := label("http.services.{serviceName}.loadbalancer.healthcheck.headers.{headerName}", ValueString)
	match, ok := spec.Match([]string{"http", "services", "app", "loadbalancer", "healthcheck", "headers", "x-forwarded"})
	if !ok {
		t.Fatal("schema did not match")
	}

	target := spec.Target(match, Context{DefaultName: "default"})
	if target.Name != "app" {
		t.Fatalf("resource name = %q, want app", target.Name)
	}
	if target.Key != "loadbalancer.healthcheck.headers" {
		t.Fatalf("target key = %q", target.Key)
	}
	if target.Entry != "x-forwarded" {
		t.Fatalf("entry = %q, want x-forwarded", target.Entry)
	}
}

func TestSchemaDomainCapturesAreGeneric(t *testing.T) {
	spec := label("http.routers.{routerName}.tls.domains[{domainIndex}].main", ValueString)
	match, ok := spec.Match([]string{"http", "routers", "app", "tls", "domains[2]", "main"})
	if !ok {
		t.Fatal("schema did not match")
	}

	target := spec.Target(match, Context{DefaultName: "default"})
	if target.Name != "app" {
		t.Fatalf("resource name = %q, want app", target.Name)
	}
	if target.Domain == nil {
		t.Fatal("missing domain target")
	}
	if target.Domain.Prefix != "tls" || target.Domain.Index != 2 || target.Domain.Field != schematarget.TLSDomainMain {
		t.Fatalf("domain target = %#v", target.Domain)
	}
}

func TestSchemaRowRejectsMultipleMappings(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	label("port -> one -> two", ValueInt)
}
