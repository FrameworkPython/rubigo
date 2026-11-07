package account

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/transport"
)

// TestRequestRoundTrip exercises the full Request path against an
// in-process server that returns a canned OK response. It is a smoke
// test for the encrypted-request pipeline (without encryption enabled).
func TestRequestRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{"status":"OK","status_det":"OK","data":{"user":{"name":"test"}}}`
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()

	a, err := New(WithAuth("abcdefghijklmnopqrstuvwxyz012345"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer a.Close()
	setTestAPIURL(a, srv.URL)

	// Disable encryption so we don't need a real symmetric key for
	// this round-trip.
	resp, err := a.Request(context.Background(), RequestOptions{
		Method:  "getUserInfo",
		Encrypt: false,
	})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	// events.Find recurses, so Str("name") finds "test" inside user.
	if name := resp.Str("name"); name != "test" {
		t.Errorf("expected name=test, got %q", name)
	}
}

// TestRequestBody verifies that the body sent to transport.Post is a
// valid JSON object the server can decode, and that the method and
// input fields survive the pooled-buffer serialisation path.
func TestRequestBody(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"status":"OK","status_det":"OK","data":{}}`)
	}))
	defer srv.Close()

	a, err := New(WithAuth("abcdefghijklmnopqrstuvwxyz012345"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer a.Close()
	setTestAPIURL(a, srv.URL)

	_, err = a.Request(context.Background(), RequestOptions{
		Method:  "sendMessage",
		Encrypt: false,
		Input:   map[string]any{"text": "hello", "to": "u0test"},
	})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if !strings.Contains(string(capturedBody), `"sendMessage"`) {
		t.Errorf("body missing method: %s", capturedBody)
	}
	if !strings.Contains(string(capturedBody), `"hello"`) {
		t.Errorf("body missing input text: %s", capturedBody)
	}
	var v any
	if err := json.Unmarshal(capturedBody, &v); err != nil {
		t.Fatalf("captured body is not valid JSON: %v\n%s", err, capturedBody)
	}
}

// BenchmarkRequestBuild measures the cost of building the request
// envelope (without going to the network). The transport's HTTP client
// is replaced with a no-op round-tripper that returns a canned OK
// response.
func BenchmarkRequestBuild(b *testing.B) {
	a, _ := New(WithAuth("abcdefghijklmnopqrstuvwxyz012345"))
	defer a.Close()
	setTestAPIURL(a, "http://127.0.0.1:0")

	a.transport.SetHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"OK","status_det":"OK","data":{}}`))),
				Header:     make(http.Header),
			}, nil
		}),
	})

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := a.Request(ctx, RequestOptions{
			Method:  "getUserInfo",
			Encrypt: false,
			Input:   map[string]any{"user_guid": "u0test"},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// roundTripperFunc adapts a function into an http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// setTestAPIURL injects a test server URL into the account's dcConfig
// so Request skips DC discovery and posts directly to the test server.
func setTestAPIURL(a *Account, url string) {
	a.initMu.Lock()
	defer a.initMu.Unlock()
	a.dcConfig = &transport.DCConfig{APIURL: url}
}
