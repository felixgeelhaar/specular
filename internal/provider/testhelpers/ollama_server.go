package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// StartFakeOllamaServer spins up an HTTP server that mimics the Ollama HTTP API.
func StartFakeOllamaServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": "0.1.0",
		})
	})
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":    req["model"],
			"response": "4",
			"done":     true,
		})
	})

	return httptest.NewServer(mux)
}
