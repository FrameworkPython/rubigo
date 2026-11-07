package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// DefaultHeaders returns the default HTTP header set used by ParsRubika.
func DefaultHeaders(userAgent string) map[string]string {
	return map[string]string{
		"origin":       "https://m.rubika.ir",
		"referer":      "https://m.rubika.ir/",
		"content-type": "application/json",
		"connection":   "keep-alive",
		"user-agent":   userAgent,
	}
}

// AndroidHeaders returns the header set used when the platform is
// Android. The User-Agent is overridden to `okhttp/3.12.1` and the
// Origin/Referer headers are dropped.
func AndroidHeaders(userAgent string) map[string]string {
	return map[string]string{
		"content-type": "application/json",
		"connection":   "keep-alive",
		"user-agent":   "okhttp/3.12.1",
	}
}

// NewHTTPClient builds an *http.Client with:
//
// - HTTP/2 enabled automatically (Go's net/http enables HTTP/2 over
// TLS by default; we just need to keep InsecureSkipVerify off in
// production and tune the connection pool).
// - Aggressive connection pooling: 256 idle conns total, 32 per host
// (the Rubika API is a single host, so the per-host value dominates),
// 120 s idle timeout, 30 s dial timeout, 30 s TLS handshake timeout.
// The defaults Go ships with (2 conns per host!) make every bursty
// request load stall on TLS handshakes — these tuned values let the
// client reuse warm TLS sessions instead.
// - Optional proxy support — accepts HTTP, HTTPS, and SOCKS5 proxy
// URLs (e.g. "http://127.0.0.1:8080", "socks5://127.0.0.1:1080",
// "socks5://user:pass@127.0.0.1:1080").
//
// The returned client is safe for concurrent use.
func NewHTTPClient(timeout time.Duration, proxyURL string, insecure bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   32,
		MaxConnsPerHost:       0, // 0 = unlimited; the rate limiter caps QPS
		IdleConnTimeout:       120 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 15 * time.Second,
		ForceAttemptHTTP2:   true,
		DisableCompression:  false,
	}
	if proxyURL != "" {
		applyProxy(tr, proxyURL)
	}
	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

// applyProxy configures the supplied http.Transport to use the given
// proxy URL. It supports HTTP, HTTPS, and SOCKS5 proxies (with or
// without authentication).
//
// Supported formats:
//
//	http://[user:pass@]host:port
//	https://[user:pass@]host:port
//	socks5://[user:pass@]host:port
//	socks5h://[user:pass@]host:port (DNS via proxy)
//
// For SOCKS5 we install a custom DialContext so the rest of the
// transport (TLS, HTTP/2, connection pooling) works unchanged.
func applyProxy(tr *http.Transport, proxyURL string) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "socks5", "socks5h":
		// Build a SOCKS5 dialer with optional auth.
		var auth *proxy.Auth
		if u.User != nil {
			pw, _ := u.User.Password()
			auth = &proxy.Auth{
				User:     u.User.Username(),
				Password: pw,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, auth, &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		})
		if err != nil {
			return
		}
		// Wrap as DialContext so http.Transport can use it.
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// SOCKS5 dialers don't honour context natively; we just
			// call Dial and rely on the dialer's own Timeout.
			return dialer.Dial(network, addr)
		}
		// When scheme is socks5h, DNS is resolved by the proxy
		// (already the default for proxy.SOCKS5). For plain socks5,
		// Go still resolves locally first — both work in practice.
	default:
		// HTTP / HTTPS proxy.
		tr.Proxy = http.ProxyURL(u)
	}
}
