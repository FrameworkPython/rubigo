package account

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core"
	coreerrors "github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core/errors"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/security"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/storage"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/transport"
)

// Start performs the connection + (optional) sign-in handshake. If a
// session has been saved previously it is reused; otherwise the
// `phoneNumber` (or the one passed in) is used to send a code and
// sign in interactively.
func (a *Account) Start(ctx context.Context, phone string) error {
	if a.transport == nil {
		return coreerrors.New(coreerrors.KindConfig, "transport is nil")
	}
	// 1) Discover DCs.
	cfg, err := transport.DiscoverDCs(ctx, a.transport)
	if err != nil {
		return err
	}
	a.dcConfig = cfg

	// 2) Load any saved credentials.
	rec, err := a.store.Load()
	if err != nil {
		return err
	}
	if rec != nil {
		if rec.Auth != "" {
			a.auth = rec.Auth
		}
		if rec.GUID != "" {
			a.guid = rec.GUID
		}
		if rec.UserAgent != "" {
			a.userAgent = rec.UserAgent
		}
		if rec.PrivateKey != "" {
			a.privateKey = rec.PrivateKey
		}
	}

	// 3) Decode auth + parse the RSA key.
	if a.auth != "" {
		a.decodedAuth = security.DecodeAuth(a.auth)
	}
	if a.privateKey != "" {
		pk, err := security.ParsePrivateKey(a.privateKey)
		if err != nil {
			return coreerrors.Wrap(coreerrors.KindAuth, "parse private key", err)
		}
		a.rsaKey = pk
	}

	// 4) Track whether we had a real auth before this point.
	hadAuth := a.auth != ""

	// Generate a tmp_session secret only if we don't have a real auth.
	if a.auth == "" {
		secret, err := security.RandomSecret(core.AuthLength)
		if err != nil {
			return err
		}
		a.auth = secret
	}
	if a.symmetricKey == "" {
		key, err := security.DeriveKey(a.auth)
		if err != nil {
			return err
		}
		a.symmetricKey = key
	}

	// 5) If we had no real auth, go straight to sign-in (skip
	// the getUserInfo probe — it will always fail with a
	// random tmp_session).
	if !hadAuth {
		if phone == "" {
			phone = a.phoneNumber
		}
		if phone == "" {
			return coreerrors.New(coreerrors.KindAuth, "not registered and no phone number provided")
		}
		if err := a.signInInteractive(ctx, phone); err != nil {
			return err
		}
	} else {
		// We have a real auth — verify it works.
		if _, err := a.GetUserInfo(ctx, ""); err != nil {
			if !coreerrors.IsStatus(err, "NotRegistered") {
				a.warnf("get_me failed during start: %v", err)
			} else {
				if phone == "" {
					phone = a.phoneNumber
				}
				if phone == "" {
					return coreerrors.New(coreerrors.KindAuth, "not registered and no phone number provided")
				}
				if err := a.signInInteractive(ctx, phone); err != nil {
					return err
				}
			}
		}
	}

	// 6) Persist the session.
	if err := a.store.Save(storage.Record{
		Phone:      a.phoneNumber,
		Auth:       a.auth,
		GUID:       a.guid,
		UserAgent:  a.userAgent,
		PrivateKey: a.privateKey,
	}); err != nil {
		a.infof("persist session: %v", err)
	}
	return nil
}

// signInInteractive performs the sendCode → signIn handshake.
func (a *Account) signInInteractive(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if phone == "" {
		return coreerrors.New(coreerrors.KindValidation, "invalid phone number")
	}

	res, err := a.SendCode(ctx, phone, "", "SMS")
	if err != nil {
		return err
	}
	phoneCodeHash := res.Str("phone_code_hash")
	hint := res.Str("hint_pass_key")
	status := res.Str("status")
	// 2FA pass-key loop. Bounded to 5 attempts to prevent an
	// infinite loop on persistent wrong input.
	for attempts := 0; status == "SendPassKey" && attempts < 5; attempts++ {
		passKey, err := a.readCode(ctx, "Password ["+hint+"] > ")
		if err != nil {
			return err
		}
		res, err = a.SendCode(ctx, phone, passKey, "SMS")
		if err != nil {
			return err
		}
		status = res.Str("status")
		// Re-read phone_code_hash in case the server rotated it.
		if h := res.Str("phone_code_hash"); h != "" {
			phoneCodeHash = h
		}
	}
	if status == "SendPassKey" {
		return coreerrors.New(coreerrors.KindAuth, "too many wrong 2FA attempts")
	}

	publicKey, privateKeyPEM, err := security.GenerateKeyPair(core.RSAKeyBits)
	if err != nil {
		return err
	}
	a.privateKey = privateKeyPEM
	// Parse the newly generated private key so we can decrypt the
	// auth returned by signIn.
	newKey, err := security.ParsePrivateKey(a.privateKey)
	if err != nil {
		return coreerrors.Wrap(coreerrors.KindAuth, "parse new private key", err)
	}
	a.rsaKey = newKey
	// Code loop. Bounded to 5 attempts.
	for attempts := 0; attempts < 5; attempts++ {
		code, err := a.readCode(ctx, "Code: ")
		if err != nil {
			return err
		}
		res, err := a.SignIn(ctx, code, phone, phoneCodeHash, publicKey)
		if err != nil {
			// Allow retry on code-already-used / wrong-code by
			// re-prompting; other errors abort.
			if isRetryableCodeErr(err) && attempts < 4 {
				continue
			}
			return err
		}
		// Request() returns nil error only when status=="OK" &&
		// status_det=="OK", so sign-in succeeded. The response
		// data does not carry a nested "status" key — extract
		// the auth directly.
		encAuth := res.Str("auth")
		auth, err := security.DecryptOAEP(a.rsaKey, encAuth)
		if err != nil {
			return err
		}
		a.auth = auth
		a.symmetricKey, err = security.DeriveKey(auth)
		if err != nil {
			return err
		}
		a.decodedAuth = security.DecodeAuth(a.auth)
		if a.privateKey != "" {
			pk, err := security.ParsePrivateKey(a.privateKey)
			if err != nil {
				return err
			}
			a.rsaKey = pk
		}
		if user := res.EventOf("user"); user != nil {
			a.guid = user.Str("user_guid")
		}
		if _, err := a.RegisterDevice(ctx, a.name); err != nil {
			return err
		}
		return nil
	}
	return coreerrors.New(coreerrors.KindAuth, "too many wrong code attempts")
}

// isRetryableCodeErr reports whether a signIn error is worth retrying
// (wrong code / code already used) versus a hard failure.
func isRetryableCodeErr(err error) bool {
	msg := err.Error()
	return containsFold(msg, "CodeIsUsed") ||
		containsFold(msg, "CodeIsExpired") ||
		containsFold(msg, "InvalidInput")
}

// containsFold reports whether s contains substr case-insensitively.
func containsFold(s, substr string) bool {
	ls := strings.ToLower(s)
	return strings.Contains(ls, strings.ToLower(substr))
}

// Stop tears down the WebSocket and the transport.
func (a *Account) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	if a.ws != nil {
		_ = a.ws.Close()
	}
}

// Run is the convenience method that calls Start + Listen. It blocks
// until `ctx` is cancelled.
func (a *Account) Run(ctx context.Context, phone string) error {
	if err := a.Start(ctx, phone); err != nil {
		return err
	}
	a.ctx, a.cancel = context.WithCancel(ctx)
	return a.Listen(a.ctx)
}

// Listen starts the WebSocket update loop. It blocks until `ctx` is
// cancelled or a fatal error occurs.
func (a *Account) Listen(ctx context.Context) error {
	if a.dcConfig == nil || a.dcConfig.WSSURL == "" {
		cfg, err := transport.DiscoverDCs(ctx, a.transport)
		if err != nil {
			return err
		}
		a.dcConfig = cfg
	}
	a.debugf("WebSocket URL = %s", a.dcConfig.WSSURL)
	a.ws = transport.NewWebSocket(a.dcConfig.WSSURL, nil)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := a.ws.Dial(ctx); err != nil {
			a.warnf("ws dial failed: %v", err)
			if sleepCtx(ctx, 3) {
				return ctx.Err()
			}
			continue
		}
		a.infof("ws connected, sending handshake...")
		if err := a.ws.Handshake(a.auth); err != nil {
			a.warnf("ws handshake failed: %v", err)
			_ = a.ws.Close()
			continue
		}
		a.infof("ws handshake OK, listening for updates...")
		// Per-connection keep-alive context. Each Dial gets its own
		// cancellable context so the keep-alive goroutine exits as
		// soon as this specific WS connection drops, rather than
		// accumulating across redials (which would leak goroutines
		// and produce duplicate pings on the new connection).
		keepAliveCtx, keepAliveCancel := context.WithCancel(ctx)
		go func() {
			if err := a.ws.KeepAlive(keepAliveCtx, pingInterval(), func() error {
				_, err := a.GetChatsUpdates(keepAliveCtx)
				return err
			}); err != nil {
				a.debugf("ws keepalive: %v", err)
			}
		}()
		a.readWSLoop(ctx)
		keepAliveCancel() // stop the keep-alive immediately
		_ = a.ws.Close()
		if sleepCtx(ctx, 3) {
			return ctx.Err()
		}
	}
}

// readWSLoop reads messages from the WebSocket until it is closed.
func (a *Account) readWSLoop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		raw, err := a.ws.Read()
		if err != nil {
			a.warnf("ws read error: %v", err)
			return
		}
		if a.debug {
			a.infof("ws received %d bytes", len(raw))
		}
		msg, err := transport.DecodeMessage(raw)
		if err != nil {
			a.warnf("ws decode error: %v", err)
			continue
		}
		if a.debug {
			a.infof("ws message keys: %v", mapKeys(msg))
		}
		go a.handleWSMessage(ctx, msg)
	}
}

// handleWSMessage is in handlews.go

// ===== small helpers ====================================================

// normalizePhone normalises a Persian/Iranian phone number to the
// international format (98...). It handles all common input forms:
//
// 09981530922 → 989981530922 (local, leading 0)
// 9981530922 → 989981530922 (bare mobile, no prefix)
// +989981530922 → 989981530922 (international with +)
// 00989981530922 → 989981530922 (international with 00)
// 989981530922 → 989981530922 (already normalized)
//
// Persian digits and separators (space, dash, parens) are also handled.
func normalizePhone(p string) string {
	// Convert Persian digits to ASCII.
	p = strings.NewReplacer("۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9").Replace(p)
	// Strip separators.
	p = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(p)
	// Strip leading "+".
	if strings.HasPrefix(p, "+") {
		p = p[1:]
	}
	// Strip leading "00" (international prefix).
	if strings.HasPrefix(p, "00") {
		p = p[2:]
	}
	// Local format: leading "0" → replace with "98".
	if strings.HasPrefix(p, "0") {
		p = "98" + p[1:]
	}
	// Bare Iranian mobile: 10 digits starting with "9" (e.g. 9981530922)
	// → prepend "98". Iranian mobiles without country code are exactly
	// 10 digits and always start with 9.
	if len(p) == 10 && strings.HasPrefix(p, "9") {
		p = "98" + p
	}
	// Validate length and digits.
	if len(p) < 7 || len(p) > 15 {
		return ""
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return p
}

// readCode obtains a code/password from the configured CodeProvider,
// or falls back to the interactive stdin prompt when no provider is
// set. The `hint` argument is a short human-readable prompt shown to
// the user (e.g. "Code:" or "Password [hint] > ").
func (a *Account) readCode(ctx context.Context, hint string) (string, error) {
	if a.codeProvider != nil {
		return a.codeProvider(ctx, hint)
	}
	return prompt(hint), nil
}

// prompt reads a single line from stdin (interactive mode). It uses
// bufio.ReadString so codes/passwords containing spaces are preserved
// (fmt.Fscanln would stop at the first whitespace and truncate them).
func prompt(msg string) string {
	fmt.Print(msg)
	s, _ := stdin().ReadString('\n')
	return strings.TrimRight(s, "\r\n")
}

func pingInterval() time.Duration {
	return durationOfSeconds(core.DefaultWebSocketPingInterval)
}

func sleepCtx(ctx context.Context, seconds int) bool {
	select {
	case <-ctx.Done():
		return true
	case <-timerAfterSeconds(seconds):
		return false
	}
}

// mapKeys returns the keys of a map for debug logging.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
