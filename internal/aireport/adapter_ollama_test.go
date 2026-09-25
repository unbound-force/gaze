package aireport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newOllamaTestServer creates an httptest.Server that simulates the ollama
// /api/generate endpoint. The handler func is called for each request.
func newOllamaTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *http.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, srv.Client()
}

// ollamaSuccessHandler returns a valid ollama response with the given text.
func ollamaSuccessHandler(response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ollamaResponse{Response: response})
	}
}

func TestOllamaAdapter_SuccessfulPost(t *testing.T) {
	report := "# Fake Ollama Report\n\n🔍 CRAP Analysis\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"
	srv, client := newOllamaTestServer(t, ollamaSuccessHandler(report))

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2", OllamaHost: srv.URL},
		httpClient: client,
	}
	got, err := adapter.Format(context.Background(), "system prompt", strings.NewReader(`{"crap":{}}`))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if got != report {
		t.Errorf("expected %q, got %q", report, got)
	}
}

func TestOllamaAdapter_MissingModel_ReturnsImmediateError(t *testing.T) {
	called := false
	srv, client := newOllamaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "", OllamaHost: srv.URL},
		httpClient: client,
	}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if !strings.Contains(err.Error(), "FR-003") {
		t.Errorf("expected FR-003 in error, got: %v", err)
	}
	if called {
		t.Error("expected no HTTP call when model is missing")
	}
}

func TestOllamaAdapter_HTTP500_ReturnsError(t *testing.T) {
	srv, client := newOllamaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	})

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2", OllamaHost: srv.URL},
		httpClient: client,
	}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected HTTP 500 in error, got: %v", err)
	}
}

func TestOllamaAdapter_EmptyResponseField_ReturnsError(t *testing.T) {
	srv, client := newOllamaTestServer(t, ollamaSuccessHandler(""))

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2", OllamaHost: srv.URL},
		httpClient: client,
	}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected FR-016 error for empty response")
	}
	if !strings.Contains(err.Error(), "FR-016") {
		t.Errorf("expected FR-016 in error, got: %v", err)
	}
}

func TestOllamaAdapter_OLLAMAHOSTEnvVar(t *testing.T) {
	report := "# Report"
	srv, client := newOllamaTestServer(t, ollamaSuccessHandler(report))

	t.Setenv("OLLAMA_HOST", srv.URL)

	// OllamaHost field is empty; adapter should read OLLAMA_HOST from env.
	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2"},
		httpClient: client,
	}
	got, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if got != report {
		t.Errorf("expected %q, got %q", report, got)
	}
}

func TestOllamaAdapter_ContextCancellation(t *testing.T) {
	// started signals that the HTTP request has arrived at the server,
	// ensuring cancellation doesn't race ahead of the round-trip.
	started := make(chan struct{})
	// handlerDone unblocks the server handler. We use an explicit channel
	// instead of r.Context().Done() because Go 1.24's httptest.Server
	// does not reliably cancel the request context when the client
	// disconnects — the handler blocks forever and srv.Close() deadlocks
	// waiting for it.
	handlerDone := make(chan struct{})
	defer close(handlerDone) // Unblock handler before t.Cleanup calls srv.Close.

	srv, client := newOllamaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-handlerDone
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2", OllamaHost: srv.URL},
		httpClient: client,
	}

	done := make(chan error, 1)
	go func() {
		_, err := adapter.Format(ctx, "prompt", strings.NewReader(`{}`))
		done <- err
	}()

	// Wait for the HTTP request to reach the server before cancelling.
	<-started
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error on cancelled context")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Format did not return after context cancellation")
	}
}

func TestOllamaAdapter_RequestBodyContainsRequiredFields(t *testing.T) {
	var capturedBody []byte
	srv, client := newOllamaTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(ollamaResponse{Response: "ok"})
	})

	adapter := &OllamaAdapter{
		config:     AdapterConfig{Name: "ollama", Model: "llama3.2", OllamaHost: srv.URL},
		httpClient: client,
	}
	_, _ = adapter.Format(context.Background(), "my system prompt", strings.NewReader(`{"crap":"data"}`))

	var body ollamaRequest
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("parsing captured body: %v", err)
	}
	if body.Model != "llama3.2" {
		t.Errorf("expected model=llama3.2, got %q", body.Model)
	}
	if body.System != "my system prompt" {
		t.Errorf("expected system=my system prompt, got %q", body.System)
	}
	if !strings.Contains(body.Prompt, "crap") {
		t.Errorf("expected crap in prompt, got %q", body.Prompt)
	}
	if body.Stream != false {
		t.Error("expected stream=false")
	}
}
