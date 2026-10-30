package proxmox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// JSON also exercises backward compatibility: old Endpoint remains usable.
func failoverClient(t *testing.T, endpoints ...string) *Client {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"Endpoint": endpoints[0], "Endpoints": endpoints, "TokenID": "discovery@pve!automation", "Token": "test"})
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Timeout = 100 * time.Millisecond
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestFailoverSticksToWorkingEndpoint(t *testing.T) {
	var first, second atomic.Int32
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { first.Add(1); http.Error(w, "unavailable", 503) }))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		second.Add(1)
		w.Write([]byte(`{"data":[{"node":"pve-good"}]}`))
	}))
	defer good.Close()
	c := failoverClient(t, bad.URL, good.URL)
	for i := 0; i < 2; i++ {
		nodes, err := c.Nodes(context.Background())
		if err != nil || len(nodes) != 1 || nodes[0].Name != "pve-good" {
			t.Fatalf("nodes=%v error=%v", nodes, err)
		}
	}
	if first.Load() != 1 || second.Load() != 2 {
		t.Fatalf("requests first=%d second=%d", first.Load(), second.Load())
	}
}

func TestFailoverDoesNotDuplicateSuccessfulRequests(t *testing.T) {
	var first, second atomic.Int32
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { first.Add(1); w.Write([]byte(`{"data":[]}`)) }))
	defer good.Close()
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { second.Add(1); w.Write([]byte(`{"data":[]}`)) }))
	defer other.Close()
	c := failoverClient(t, good.URL, other.URL)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Nodes(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if first.Load() != 20 || second.Load() != 0 {
		t.Fatalf("requests first=%d second=%d", first.Load(), second.Load())
	}
}

func TestFailoverDoesNotRetryClientErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 429} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var otherCalls atomic.Int32
			bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "denied", status) }))
			defer bad.Close()
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { otherCalls.Add(1); w.Write([]byte(`{"data":[]}`)) }))
			defer other.Close()
			_, err := failoverClient(t, bad.URL, other.URL).Nodes(context.Background())
			if err == nil || otherCalls.Load() != 0 {
				t.Fatalf("err=%v fallback calls=%d", err, otherCalls.Load())
			}
		})
	}
}

func TestFailoverConnectionFailureAndDuplicateEndpoints(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dead.Close()
	var calls atomic.Int32
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(`{"data":[]}`)) }))
	defer good.Close()
	if _, err := failoverClient(t, dead.URL, dead.URL, good.URL).Nodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("successful calls=%d", calls.Load())
	}
}

func TestFailoverBoundsAllFailedAttempts(t *testing.T) {
	var calls atomic.Int32
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Error(w, "unavailable", 503) }))
	defer bad.Close()
	c := failoverClient(t, bad.URL, bad.URL+"/api2/json")
	if _, err := c.Nodes(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("same endpoint retried %d times", calls.Load())
	}
}

func TestFailoverHonorsCancellation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.Write([]byte(`{"data":[]}`)) }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := failoverClient(t, server.URL).Nodes(ctx); err == nil {
		t.Fatal("expected cancellation")
	}
	if calls.Load() != 0 {
		t.Fatalf("requests after cancellation=%d", calls.Load())
	}
}

func TestFailoverRetriesRequestTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"data":[]}`)) }))
	defer good.Close()
	if _, err := failoverClient(t, slow.URL, good.URL).Nodes(context.Background()); err != nil {
		t.Fatalf("failover after timeout: %v", err)
	}
}

func TestFailoverDoesNotRetryWritesOrDecodeErrors(t *testing.T) {
	for _, test := range []struct {
		name, method, body string
		status             int
	}{
		{"write", http.MethodPost, "unavailable", http.StatusServiceUnavailable},
		{"decode", http.MethodGet, "not-json", http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			var otherCalls atomic.Int32
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(test.status); w.Write([]byte(test.body)) }))
			defer first.Close()
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { otherCalls.Add(1); w.Write([]byte(`{"data":[]}`)) }))
			defer other.Close()
			var data []Node
			err := failoverClient(t, first.URL, other.URL).do(context.Background(), test.method, "/nodes", nil, &data)
			if err == nil || otherCalls.Load() != 0 {
				t.Fatalf("err=%v fallback calls=%d", err, otherCalls.Load())
			}
		})
	}
}

func TestEndpointListTakesPrecedenceAndValidatesEveryURL(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"data":[]}`)) }))
	defer good.Close()
	client, err := NewClient(Config{Endpoint: "invalid-unused-legacy-endpoint", Endpoints: []string{good.URL}, TokenID: "test", Token: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Nodes(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewClient(Config{Endpoints: []string{good.URL, "invalid"}, TokenID: "test", Token: "test"}); err == nil {
		t.Fatal("expected invalid endpoint error")
	}
}

func TestConcurrentSuccessDoesNotUndoEndpointSwitch(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var firstCalls, secondCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstCalls.Add(1) == 1 {
			close(started)
			<-release
			w.Write([]byte(`{"data":[]}`))
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer second.Close()
	client := failoverClient(t, first.URL, second.URL)
	client.httpClient.Timeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	older := make(chan error, 1)
	go func() { _, err := client.Nodes(ctx); older <- err }()
	<-started
	_, err := client.Nodes(ctx)
	close(release)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-older; err != nil {
		t.Fatal(err)
	}
	if _, err := client.Nodes(ctx); err != nil {
		t.Fatal(err)
	}
	if firstCalls.Load() != 2 || secondCalls.Load() != 2 {
		t.Fatalf("stale success changed endpoint: first=%d second=%d", firstCalls.Load(), secondCalls.Load())
	}
}

func TestFailoverHandlesResponseBodyFailures(t *testing.T) {
	for _, test := range []struct {
		name         string
		status       int
		stall, retry bool
	}{
		{"body-timeout", http.StatusOK, true, true},
		{"truncated-success-body", http.StatusOK, false, true},
		{"truncated-server-error", http.StatusServiceUnavailable, false, true},
		{"truncated-permission-error", http.StatusForbidden, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "1000")
				w.WriteHeader(test.status)
				w.(http.Flusher).Flush()
				if test.stall {
					<-r.Context().Done()
				} else {
					w.Write([]byte("truncated"))
				}
			}))
			defer first.Close()
			var otherCalls atomic.Int32
			other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				otherCalls.Add(1)
				w.Write([]byte(`{"data":[]}`))
			}))
			defer other.Close()
			_, err := failoverClient(t, first.URL, other.URL).Nodes(context.Background())
			if test.retry {
				if err != nil || otherCalls.Load() != 1 {
					t.Fatalf("err=%v fallback calls=%d", err, otherCalls.Load())
				}
			} else if err == nil || otherCalls.Load() != 0 {
				t.Fatalf("err=%v fallback calls=%d", err, otherCalls.Load())
			}
		})
	}
}
