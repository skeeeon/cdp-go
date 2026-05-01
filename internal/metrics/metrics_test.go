package metrics

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestServe brings up the /metrics endpoint on an ephemeral port, drives a
// couple of counters, and verifies the scrape body contains them. This
// guards against future refactors silently dropping registration.
func TestServe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // we just wanted a free port

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Serve(ctx, addr) }()

	DatagramsReceived.Inc()
	GeofenceEvents.WithLabelValues("enter").Inc()

	// Poll briefly for the server to bind.
	var resp *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = http.Get("http://" + addr + "/metrics")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, want := range []string{
		"cdp_datagrams_received_total",
		`cdp_geofence_events_total{type="enter"}`,
		"go_goroutines",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("scrape body missing %q", want)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve returned: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not exit after ctx cancel")
	}
}
