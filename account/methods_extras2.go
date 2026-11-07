package account

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Channel post link resolution ==================
//
// Public channel post links (https://rubika.ir/post/<hash>) are resolved
// by calling getLinkFromAppUrl and decoding the resulting deep link.

// channelPostLinkRegex matches rubika.ir/post/<hash> URLs.
var channelPostLinkRegex = regexp.MustCompile(`https?://rubika\.ir/post/([A-Za-z0-9]+)`)

// GetChannelPostByLink resolves a channel post from a web URL. It calls
// getLinkFromAppUrl to convert the web link into a deep link, then parses
// the result to extract channel GUID and message ID.
func (a *Account) GetChannelPostByLink(ctx context.Context, postURL string) (*events.Event, error) {
	resp, err := a.GetLinkFromAppURL(ctx, postURL)
	if err != nil {
		return nil, err
	}
	// The response contains a deep link like:
	// rubika://c.rubika.ir/<channel_guid>/<message_id>
	link := resp.AnyStr("link")
	if link == "" {
		// Try alternate field names
		link = resp.AnyStr("app_url", "url", "deep_link")
	}
	if link == "" {
		return resp, nil
	}
	// Parse the deep link
	parts := strings.Split(strings.TrimPrefix(link, "rubika://c.rubika.ir/"), "/")
	if len(parts) >= 2 {
		channelGUID := parts[0]
		messageID := parts[1]
		// Fetch the actual message
		return a.GetMessagesByID(ctx, channelGUID, []string{messageID})
	}
	return resp, nil
}

// ===== Channel post scraping =========================
//
// getPostByAskLink resolves an "ask link" (a special share link used for
// channel posts that require user confirmation).

// GetPostByAskLink resolves a post from an ask link. It calls
// getLinkFromAppUrl and then fetches the message.
func (a *Account) GetPostByAskLink(ctx context.Context, askLink string) (*events.Event, error) {
	return a.GetChannelPostByLink(ctx, askLink)
}

// ===== HTTP helpers for scraping =====================
//
// Some channel posts can only be resolved through the public web page.
// This lightweight helper fetches the rubika.ir page and extracts the
// deep link from the HTML.

// ScrapeChannelPostLink fetches the rubika.ir web page for a post and
// extracts the deep link. This is a fallback when getLinkFromAppUrl
// does not resolve.
func (a *Account) ScrapeChannelPostLink(ctx context.Context, postURL string) (string, error) {
	matches := channelPostLinkRegex.FindStringSubmatch(postURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid channel post URL: %s", postURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, postURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := a.transport.HTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Look for a deep link or meta refresh in the HTML.
	// The page typically contains: <meta property="og:url" content="rubika://...">
	html := string(body)
	idx := strings.Index(html, "rubika://")
	if idx >= 0 {
		end := strings.IndexAny(html[idx:], "\"'<>\n ")
		if end > 0 {
			return html[idx : idx+end], nil
		}
	}
	return "", fmt.Errorf("could not extract deep link from %s", postURL)
}

// ===== JSON convenience =================================

// FetchJSON is a low-level helper that performs an HTTP GET and decodes
// the response as JSON. It is useful for calling third-party APIs
// (e.g. translation services, Rubino download helpers).
func (a *Account) FetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.transport.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// ===== GetStories ===================================

// GetContactsStories returns the stories of the user's contacts.
//
// (GetContactsLastOnline is already defined in methods_users.go.)
func (a *Account) GetContactsStories(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getStories", Input: map[string]any{}, Encrypt: true})
}

// ===== SetChatProtectedContent ======================

// SetChatProtectedContent toggles the "protected content" (restricted
// content / no-forward) flag on a group or channel.
func (a *Account) SetChatProtectedContent(ctx context.Context, objectGUID string, enabled bool) (*events.Event, error) {
	if strings.HasPrefix(objectGUID, "c0") {
		// Channel
		return a.Request(ctx, RequestOptions{Method: "editChannelInfo", Input: map[string]any{
			"channel_guid":          objectGUID,
			"is_restricted_content": enabled,
			"updated_parameters":    []string{"is_restricted_content"},
		}, Encrypt: true})
	}
	// Group
	return a.Request(ctx, RequestOptions{Method: "editGroupInfo", Input: map[string]any{
		"group_guid":            objectGUID,
		"is_restricted_content": enabled,
		"updated_parameters":    []string{"is_restricted_content"},
	}, Encrypt: true})
}

// ===== SetAskSpamAction =============================
//
// SetAskSpamAction is already defined in methods_missing.go.

// ===== MuteChat with duration =======================

// MuteChatFor mutes a chat for the given duration (in seconds). The
// Rubika API requires a minimum of 1800 seconds (30 minutes).
func (a *Account) MuteChatFor(ctx context.Context, objectGUID string, durationSeconds int) (*events.Event, error) {
	if durationSeconds < 1800 {
		durationSeconds = 1800
	}
	return a.Request(ctx, RequestOptions{Method: "setActionChat", Input: map[string]any{
		"object_guid": objectGUID,
		"action":      "Mute",
		"mute":        durationSeconds,
	}, Encrypt: true})
}
