package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core"
	coreerrors "github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core/errors"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/security"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/transport"
)

// RequestOptions carries the optional flags for the encrypted request.
type RequestOptions struct {
	APIVersion string
	Auth       string
	Input      map[string]any
	Method     string
	Encrypt    bool
	TmpSession bool
}

// Request is the central encrypted request dispatcher. It encrypts the
// input, sends it through the transport layer, decrypts the response,
// validates `status_det == OK` and returns a typed *events.Event.
//
// The top-level payload map and the inner encInput map are acquired
// from the transport map pool and returned once the request has been
// serialised, so a `getUserInfo` round-trip does not allocate two
// fresh maps per call. When `opts.Input` is nil a pooled empty map is
// used (the server treats absent keys and empty maps identically).
// The payload is JSON-encoded into a pooled *bytes.Buffer and handed
// to transport.Post as raw bytes, skipping the reflection-based
// marshal in the transport layer.
func (a *Account) Request(ctx context.Context, opts RequestOptions) (*events.Event, error) {
	// Lazy-init the DC config, auth secret and symmetric key under a
	// lock so concurrent Request calls cannot race on those fields or
	// double-discover DCs.
	a.initMu.Lock()
	if a.dcConfig == nil || a.dcConfig.APIURL == "" {
		cfg, err := transport.DiscoverDCs(ctx, a.transport)
		if err != nil {
			a.initMu.Unlock()
			return nil, err
		}
		a.dcConfig = cfg
	}
	if a.auth == "" {
		secret, err := security.RandomSecret(core.AuthLength)
		if err != nil {
			a.initMu.Unlock()
			return nil, err
		}
		a.auth = secret
	}
	if a.symmetricKey == "" {
		key, err := security.DeriveKey(a.auth)
		if err != nil {
			a.initMu.Unlock()
			return nil, err
		}
		a.symmetricKey = key
	}
	a.initMu.Unlock()

	apiVersion := opts.APIVersion
	if apiVersion == "" {
		apiVersion = core.APIVersion
	}
	auth := opts.Auth
	if auth == "" {
		auth = a.auth
	}
	input := opts.Input
	if input == nil {
		// Use the pool only for the call duration; we don't return
		// this map to the caller, so we must release it.
		input = transport.AcquireMap()
		defer transport.ReleaseMap(input)
	}
	method := opts.Method
	if method == "" {
		method = "getUserInfo"
	}
	encrypt := opts.Encrypt
	tmpSession := opts.TmpSession

	// Build the top-level payload using a pooled map. We hold onto it
	// only for the duration of marshal, then release.
	payload := transport.AcquireMap()
	defer transport.ReleaseMap(payload)
	payload["api_version"] = apiVersion
	if tmpSession {
		payload["tmp_session"] = auth
	} else {
		payload["auth"] = a.decodedAuth
	}

	var encInput map[string]any
	if apiVersion == "6" {
		// encInput is also pooled — it's only needed until the
		// payload has been serialised. We defer the release so the
		// map survives the Encode(payload) call below (when
		// encrypt=false, payload["data_enc"] holds a reference to
		// encInput, so we cannot release it earlier).
		encInput = transport.AcquireMap()
		defer transport.ReleaseMap(encInput)
		encInput["client"] = a.identity
		encInput["method"] = method
		encInput["input"] = input
		// Verbose debug logging — only when the account was
		// explicitly constructed with debug enabled, so plaintext
		// payloads are never logged in production.
		if a.debug && method == "sendMessage" {
			if jb, err := json.Marshal(encInput); err == nil {
				a.infof("sendMessage plaintext input: %s", string(jb))
			}
		}
		if encrypt {
			enc, err := security.EncryptJSON(encInput, []byte(a.symmetricKey))
			if err != nil {
				return nil, err
			}
			payload["data_enc"] = enc
			if !tmpSession && a.rsaKey != nil {
				sig, err := security.SignRSA(a.rsaKey, enc)
				if err != nil {
					return nil, err
				}
				payload["sign"] = sig
			}
		} else {
			payload["data_enc"] = encInput
		}
	} else {
		// API versions 0 and 4 (rarely used; parity for completeness).
		switch apiVersion {
		case "0":
			payload["auth"] = auth
			payload["client"] = a.identity
			payload["data"] = input
			payload["method"] = method
		case "4":
			payload["client"] = a.identity
			payload["method"] = method
		}
	}

	// Pre-marshal the payload into a pooled buffer and hand the
	// transport raw bytes — this lets transport.Post skip its own
	// json.Marshal reflection pass.
	bodyBuf := transport.AcquireBuffer()
	defer transport.ReleaseBuffer(bodyBuf)
	if err := json.NewEncoder(bodyBuf).Encode(payload); err != nil {
		return nil, fmt.Errorf("account: encode payload: %w", err)
	}
	// json.Encoder.Encode appends a trailing newline; strip it so the
	// server sees a tight JSON object.
	body := bodyBuf.Bytes()
	if n := len(body); n > 0 && body[n-1] == '\n' {
		body = body[:n-1]
	}

	respBody, status, err := a.transport.Post(ctx, a.dcConfig.APIURL, body)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, coreerrors.New(coreerrors.KindAPI, fmt.Sprintf("http %d: %s", status, string(respBody)))
	}
	var resp map[string]any
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, coreerrors.Wrap(coreerrors.KindAPI, "decode response", err)
	}
	if enc, ok := resp["data_enc"].(string); ok && enc != "" {
		decrypted, err := security.Decrypt(enc, []byte(a.symmetricKey))
		if err != nil {
			return nil, coreerrors.Wrap(coreerrors.KindAuth, "decrypt", err)
		}
		if err := json.Unmarshal(decrypted, &resp); err != nil {
			return nil, coreerrors.Wrap(coreerrors.KindAPI, "decode decrypted", err)
		}
	}
	st, _ := resp["status"].(string)
	stDet, _ := resp["status_det"].(string)
	if st == "OK" && stDet == "OK" {
		data, _ := resp["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}
		return events.New(data), nil
	}
	// Verbose debug logging of the decrypted server response — only
	// when debug is explicitly enabled.
	if a.debug {
		if jb, err := json.Marshal(resp); err == nil {
			a.infof("%s server response: %s", method, string(jb))
		}
	}
	return nil, coreerrors.FromStatus(stDet, fmt.Sprintf("%s failed", method))
}
