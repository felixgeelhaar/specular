package provider

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer starts an HTTP server bound to IPv4-only loopback so tests work
// inside restricted sandboxes that forbid IPv6 listeners.
func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("unable to start test server: %v", err)
	}

	server := &httptest.Server{
		Listener: listener,
		Config:   &http.Server{Handler: handler},
	}
	server.Start()
	t.Cleanup(server.Close)
	return server
}
