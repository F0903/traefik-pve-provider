package proxmox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRequestsNodes(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"node":"pve-1","status":"online"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint: server.URL,
		TokenID:  "root@pam!traefik",
		Token:    "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	nodes, err := client.Nodes(context.Background())
	if err != nil {
		t.Fatalf("Nodes() error = %v", err)
	}

	if gotPath != "/api2/json/nodes" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "PVEAPIToken=root@pam!traefik=secret" {
		t.Fatalf("auth header = %q", gotAuth)
	}
	if len(nodes) != 1 || nodes[0].Name != "pve-1" {
		t.Fatalf("nodes = %#v", nodes)
	}
}

func TestClientDoesNotDoubleAppendAPIPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint: server.URL + "/api2/json",
		TokenID:  "root@pam!traefik",
		Token:    "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.Nodes(context.Background()); err != nil {
		t.Fatalf("Nodes() error = %v", err)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "permission denied", http.StatusForbidden)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		Endpoint: server.URL,
		TokenID:  "root@pam!traefik",
		Token:    "secret",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.Nodes(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Body, "permission denied") {
		t.Fatalf("body = %q", apiErr.Body)
	}
}
