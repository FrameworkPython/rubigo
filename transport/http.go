package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Post sends a POST request with the supplied JSON payload and returns
// the raw response body. It retries on transient failures according to
// the configured RetryPolicy.
//
// When `payload` is already a `[]byte` it is sent verbatim (no
// re-marshal). When it is a `*bytes.Buffer` it is also sent as-is.
// Otherwise it is JSON-marshalled. This lets the account client hand
// us a pre-encoded body that skips a marshal on the request path.
func (t *Transport) Post(ctx context.Context, url string, payload any) ([]byte, int, error) {
	switch v := payload.(type) {
	case []byte:
		return t.postRaw(ctx, url, v)
	case *bytes.Buffer:
		return t.postRaw(ctx, url, v.Bytes())
	case *bytes.Reader:
		buf := AcquireBuffer()
		_, _ = io.Copy(buf, v)
		body := append([]byte(nil), buf.Bytes()...)
		ReleaseBuffer(buf)
		return t.postRaw(ctx, url, body)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("transport: marshal: %w", err)
	}
	return t.postRaw(ctx, url, body)
}

// postRaw is the inner loop that performs the actual POST with retries.
//
// The request body is wrapped in a `bytes.NewReader` so it is read
// zero-copy on every retry attempt. The response body is read into a
// pooled `*bytes.Buffer` and the final bytes copied out, so a typical
// <4 KB Rubika response no longer triggers a fresh growable buffer
// allocation per call.
func (t *Transport) postRaw(ctx context.Context, url string, body []byte) ([]byte, int, error) {
	policy := DefaultRetryPolicy()
	if t.maxRetries > 0 {
		policy.MaxRetries = t.maxRetries
	}

	var lastErr error
	var lastStatus int
	for attempt := 1; attempt <= policy.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		// bytes.NewReader is zero-copy over body — no string()
		// conversion, no second allocation per retry.
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, 0, err
		}
		// Pre-set Content-Length so net/http doesn't have to wrap the
		// body in a chunked reader. For a small JSON envelope this
		// skips one alloc per request.
		req.ContentLength = int64(len(body))
		for k, v := range t.headers {
			req.Header.Set(k, v)
		}
		resp, err := t.httpClient.Do(req)
		if err != nil {
			lastErr = err
			lastStatus = 0
			if !policy.ShouldRetry(0, err) {
				break
			}
			_ = policy.Sleep(ctx, attempt)
			continue
		}
		// Read the response body through a pooled buffer.
		respBody, readErr := readBodyPooled(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			lastStatus = resp.StatusCode
			if !policy.ShouldRetry(resp.StatusCode, readErr) {
				break
			}
			_ = policy.Sleep(ctx, attempt)
			continue
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("transport: http %d: %s", resp.StatusCode, string(respBody))
			lastStatus = resp.StatusCode
			if !policy.ShouldRetry(resp.StatusCode, nil) {
				break
			}
			_ = policy.Sleep(ctx, attempt)
			continue
		}
		return respBody, resp.StatusCode, nil
	}
	if lastErr == nil {
		lastErr = ErrMaxRetries
	}
	return nil, lastStatus, lastErr
}

// maxResponseBody caps the response body read. Rubika API responses
// are small JSON envelopes (a few KB at most); 8 MiB is a generous
// safety net that still protects against a runaway server.
const maxResponseBody = 8 << 20

// readBodyPooled reads the entire response body into a freshly
// allocated []byte, but uses a pooled *bytes.Buffer as the growable
// intermediate so the read path doesn't allocate one buffer for
// io.ReadAll + another for the final result. The returned slice is
// always a fresh allocation (so the caller can keep it without
// holding the pool's capacity).
func readBodyPooled(r io.Reader) ([]byte, error) {
	buf := AcquireBuffer()
	defer ReleaseBuffer(buf)
	// Read in 32 KB chunks; copy the final size out in one alloc.
	if _, err := io.CopyBuffer(buf, io.LimitReader(r, maxResponseBody+1), make([]byte, 32*1024)); err != nil {
		return nil, err
	}
	if buf.Len() > maxResponseBody {
		return nil, fmt.Errorf("transport: response body too large (>%d bytes)", maxResponseBody)
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

// Get sends a GET request and returns the raw response body.
func (t *Transport) Get(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return body, resp.StatusCode, fmt.Errorf("transport: http %d: %s", resp.StatusCode, string(body))
	}
	return body, resp.StatusCode, nil
}

// PostJSON is a convenience helper that JSON-decodes the response body
// into `out`.
func (t *Transport) PostJSON(ctx context.Context, url string, payload any, out any) (int, error) {
	body, status, err := t.Post(ctx, url, payload)
	if err != nil {
		return status, err
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return status, fmt.Errorf("transport: decode response: %w", err)
		}
	}
	return status, nil
}

// New constructs a Transport with the supplied options. It uses the
// default HTTP/2-enabled client and the standard header set.
func New(opts ...Option) *Transport {
	t := &Transport{
		headers:    DefaultHeaders(""),
		timeout:    30 * time.Second,
		maxRetries: 5,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(t)
		}
	}
	if t.userAgent != "" {
		t.headers["user-agent"] = t.userAgent
	}
	t.httpClient = NewHTTPClient(t.timeout, t.proxy, false)
	return t
}

// SetHeader updates a single header. Useful when a client needs to
// override a single header (e.g. switch the User-Agent at runtime).
func (t *Transport) SetHeader(key, value string) {
	t.headers[key] = value
}

// SetHTTPClient replaces the internal *http.Client. Callers can pass
// in a custom HTTP/3 client here.
func (t *Transport) SetHTTPClient(c *http.Client) {
	if c != nil {
		t.httpClient = c
	}
}

// HTTPClient returns the internal *http.Client (for advanced use).
func (t *Transport) HTTPClient() *http.Client { return t.httpClient }
