package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core"
	coreerrors "github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core/errors"
)

// DCDiscoveryURLs is the list of endpoints tried, in order, to discover
// the active Rubika data-centers. If one fails (DNS, TLS, timeout,
// HTTP 5xx) the next is tried. The primary endpoint has historically
// been the most reliable; the fallbacks cover regional outages.
var DCDiscoveryURLs = []string{
	"https://getdcmess.iranlms.ir/",
	"https://messengerget.iranlms.ir/",
}

// DCConfig holds the discovered API map + WebSocket URL.
type DCConfig struct {
	APIURL   string
	WSSURL   string
	APIMap   map[string]string
	Priority []string
}

// DiscoverDCs fetches the API/WS endpoints from the data-center
// discovery service. It tries every URL in `DCDiscoveryURLs` until one
// returns a complete configuration. When all endpoints fail, the
// returned error contains a short diagnostic for each endpoint so the
// caller can tell whether the failure was DNS, TLS, timeout, or a
// server-side 5xx — instead of the opaque "all DC discovery endpoints
// failed" message.
func DiscoverDCs(ctx context.Context, t *Transport) (*DCConfig, error) {
	var diagnostics []string
	for _, url := range DCDiscoveryURLs {
		body, status, err := t.Get(ctx, url)
		if err != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		if status >= 400 {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: HTTP %d", url, status))
			continue
		}
		cfg, parseErr := parseDCs(body)
		if parseErr != nil {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: parse: %v", url, parseErr))
			continue
		}
		return cfg, nil
	}
	return nil, coreerrors.New(coreerrors.KindNetwork,
		"DC discovery failed; tried "+strings.Join(diagnostics, "; "))
}

// parseDCs extracts the API map + WebSocket URL from the JSON payload.
func parseDCs(body []byte) (*DCConfig, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("transport: invalid dc json: %w", err)
	}
	data, _ := raw["data"].(map[string]any)
	if data == nil {
		return nil, coreerrors.New(coreerrors.KindNetwork, "missing data field in DC response")
	}
	apiList, _ := data["API"].(map[string]any)
	socketList, _ := data["socket"].(map[string]any)

	cfg := &DCConfig{APIMap: map[string]string{}}
	for code, endpoint := range apiList {
		s, ok := endpoint.(string)
		if !ok || s == "" {
			continue
		}
		normalized := ensureTrailingSlash(strings.TrimRight(s, "/"))
		cfg.APIMap[strcode(code)] = normalized
	}

	var preferred []string
	if defAPI, ok := data["default_api"]; ok {
		code := strcode(defAPI)
		if _, exists := cfg.APIMap[code]; exists {
			preferred = append(preferred, code)
		}
	}
	if defs, ok := data["default_apis"].([]any); ok {
		for _, c := range defs {
			cs := strcode(c)
			if _, exists := cfg.APIMap[cs]; !exists {
				continue
			}
			if !containsStr(preferred, cs) {
				preferred = append(preferred, cs)
			}
		}
	}
	if len(preferred) == 0 {
		for c := range cfg.APIMap {
			preferred = append(preferred, c)
		}
	}
	cfg.Priority = preferred
	if len(cfg.Priority) > 0 {
		cfg.APIURL = cfg.APIMap[cfg.Priority[0]]
	}
	if defSocket, ok := data["default_socket"]; ok {
		if s, ok := socketList[strcode(defSocket)].(string); ok {
			cfg.WSSURL = s
		}
	}
	if cfg.APIURL == "" || cfg.WSSURL == "" {
		return nil, coreerrors.New(coreerrors.KindNetwork, "incomplete DC payload")
	}
	return cfg, nil
}

// PostFormFile uploads a single file via multipart/form-data. It is
// used by the file-upload endpoint returned by `requestSendFile`.
//
// Errors from the pipe-writer goroutine (e.g. the file reader fails
// mid-stream) are propagated via CloseWithError so the HTTP request
// fails with a real error instead of completing with truncated data.
func PostFormFile(ctx context.Context, t *Transport, url, fieldName, fileName string, body io.Reader) ([]byte, int, error) {
	// Build multipart body manually to avoid importing mime/multipart
	// boundary in this file.
	boundary := "parsrubika-boundary-" + randomBoundary()
	var sb strings.Builder
	fmt.Fprintf(&sb, "--%s\r\n", boundary)
	fmt.Fprintf(&sb, "Content-Disposition: form-data; name=%q; filename=%q\r\n", fieldName, fileName)
	sb.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	header := sb.String()

	// We can't stream a multipart body composed of multiple readers
	// without a pipe; use a simple pipe. Propagate the FIRST error
	// (write or copy) via CloseWithError so the HTTP request fails
	// instead of completing with truncated data.
	pr, pw := io.Pipe()
	go func() {
		var firstErr error
		if _, err := pw.Write([]byte(header)); err != nil {
			firstErr = err
		} else if _, err := io.Copy(pw, body); err != nil {
			firstErr = err
		} else if _, err := pw.Write([]byte("\r\n--" + boundary + "--\r\n")); err != nil {
			firstErr = err
		}
		if firstErr != nil {
			_ = pw.CloseWithError(firstErr)
		} else {
			_ = pw.Close()
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("transport: upload http %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, nil
}

// randomBoundary returns a small pseudo-unique boundary string. It is
// not cryptographically secure — its only purpose is to be unique within
// a single request body.
func randomBoundary() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ===== small helpers ====================================================

func strcode(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%d", int(x))
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func containsStr(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func ensureTrailingSlash(u string) string {
	if u == "" || strings.HasSuffix(u, "/") {
		return u
	}
	return u + "/"
}

// Constants exported for callers that need to tune the discovery loop.
const (
	DCMaxRetries   = 3
	DCBackoffDelay = time.Second
)

// Re-export from core so callers don't have to import both packages.
var _ = core.Version
