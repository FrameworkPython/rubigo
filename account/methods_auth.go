package account

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Auth methods (sendCode / signIn / registerDevice) =================

// SendCode triggers the SMS / Internal verification code for `phone`.
func (a *Account) SendCode(ctx context.Context, phone, passKey, sendType string) (*events.Event, error) {
	if sendType == "" {
		sendType = "SMS"
	}
	if sendType != "SMS" && sendType != "Internal" {
		return nil, fmt.Errorf("account: send_type must be 'SMS' or 'Internal'")
	}
	in := map[string]any{
		"phone_number": phone,
		"send_type":    sendType,
	}
	// pass_key must NOT be included when empty (server returns INVALID_INPUT).
	if passKey != "" {
		in["pass_key"] = passKey
	}
	return a.Request(ctx, RequestOptions{
		Method:     "sendCode",
		Input:      in,
		TmpSession: true,
		Encrypt:    true,
	})
}

// SignIn completes the registration handshake.
func (a *Account) SignIn(ctx context.Context, phoneCode, phone, phoneCodeHash, publicKey string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "signIn",
		Input: map[string]any{
			"phone_code":      phoneCode,
			"phone_number":    phone,
			"phone_code_hash": phoneCodeHash,
			"public_key":      publicKey,
		},
		TmpSession: true,
		Encrypt:    true,
	})
}

// RegisterDevice registers the device model with the server.
func (a *Account) RegisterDevice(ctx context.Context, deviceModel string) (*events.Event, error) {
	var input map[string]any
	switch a.platform {
	case "Android":
		input = androidDevice(a.userAgent, a.langCode, a.identity.AppVersion, deviceModel)
	default:
		input = browserDevice(a.userAgent, a.langCode, a.identity.AppVersion, deviceModel)
	}
	return a.Request(ctx, RequestOptions{Method: "registerDevice", Input: input, Encrypt: true})
}

// UnregisterDevice removes the current device registration from the
// server. Use this for a clean logout (without it the device lingers
// in the server's device list and may keep receiving push tokens).
// For a full logout, follow this with Logout (in methods_auth2fa.go).
func (a *Account) UnregisterDevice(ctx context.Context) (*events.Event, error) {
	input := map[string]any{
		"lang_code":   a.langCode,
		"app_version": a.identity.AppVersion,
	}
	return a.Request(ctx, RequestOptions{Method: "unregisterDevice", Input: input, Encrypt: true})
}

var browserRE = regexp.MustCompile(`(?i)(opera|chrome|safari|firefox|msie|trident)/(\d+)`)
var digitsRE = regexp.MustCompile(`\d+`)

func browserDevice(userAgent, langCode, appVersion, deviceModel string) map[string]any {
	dm := parseBrowserModel(userAgent)
	if deviceModel != "" {
		dm = deviceModel
	}
	return map[string]any{
		"token":          "",
		"lang_code":      langCode,
		"token_type":     "Firebase",
		"app_version":    "PW_" + appVersion,
		"system_version": parseSystemVersion(userAgent),
		"device_model":   dm,
		"device_hash":    "2" + strings.Join(digitsRE.FindAllString(userAgent, -1), ""),
	}
}

func androidDevice(userAgent, langCode, appVersion, _ string) map[string]any {
	return map[string]any{
		"token":          "",
		"lang_code":      langCode,
		"token_type":     "Firebase",
		"app_version":    "MA_" + appVersion,
		"system_version": "SDK 22",
		"device_model":   "samsungSM-G925F",
		"device_hash":    "2" + strings.Join(digitsRE.FindAllString(userAgent, -1), ""),
	}
}

func parseBrowserModel(ua string) string {
	m := browserRE.FindStringSubmatch(ua)
	if m == nil {
		return "Unknown"
	}
	return m[1] + " " + m[2]
}

func parseSystemVersion(ua string) string {
	mapping := map[string]string{
		"Windows NT 10.0": "Windows 10",
		"Windows NT 6.2":  "Windows 8",
		"Windows NT 6.1":  "Windows 7",
		"Windows NT 6.0":  "Windows Vista",
		"Windows NT 5.1":  "Windows XP",
		"Windows NT 5.0":  "Windows 2000",
		"Mac":             "Mac/iOS",
		"X11":             "UNIX",
		"Linux":           "Linux",
	}
	for k, v := range mapping {
		if strings.Contains(ua, k) {
			return v
		}
	}
	return "Unknown"
}
