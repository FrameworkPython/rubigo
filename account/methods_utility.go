package account

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Utility helpers ===================
//
// Convenience wrappers and link/text utilities.

// rubikaLinkRegex matches any rubika.ir URL.
var rubikaLinkRegex = regexp.MustCompile(`https?://rubika\.ir/\S+`)

// groupLinkRegex matches rubika.ir/joing/<hash> URLs.
var groupLinkRegex = regexp.MustCompile(`https?://rubika\.ir/joing/[A-Za-z0-9]+`)

// channelLinkRegex matches rubika.ir/joinc/<hash> URLs.
var channelLinkRegex = regexp.MustCompile(`https?://rubika\.ir/joinc/[A-Za-z0-9]+`)

// postLinkRegex matches rubika.ir/post/<hash> URLs.
var postLinkRegex = regexp.MustCompile(`https?://rubika\.ir/post/[A-Za-z0-9]+`)

// usernameRegex matches @username patterns (4-32 chars).
var usernameRegex = regexp.MustCompile(`@([A-Za-z0-9_]{4,32})`)

// IsRubikaLink reports whether s contains a rubika.ir URL.
func IsRubikaLink(s string) bool { return rubikaLinkRegex.MatchString(s) }

// IsGroupLink reports whether s is a rubika group join link.
func IsGroupLink(s string) bool { return groupLinkRegex.MatchString(s) }

// IsChannelLink reports whether s is a rubika channel join link.
func IsChannelLink(s string) bool { return channelLinkRegex.MatchString(s) }

// IsPostLink reports whether s is a rubika post link.
func IsPostLink(s string) bool { return postLinkRegex.MatchString(s) }

// IsUsername reports whether s is a @username (4-32 alphanumeric/underscore).
func IsUsername(s string) bool { return usernameRegex.MatchString(s) }

// GetRubikaLinks extracts all rubika.ir URLs from s.
func GetRubikaLinks(s string) []string {
	return rubikaLinkRegex.FindAllString(s, -1)
}

// GetGroupLinks extracts all group join links from s.
func GetGroupLinks(s string) []string {
	return groupLinkRegex.FindAllString(s, -1)
}

// GetChannelLinks extracts all channel join links from s.
func GetChannelLinks(s string) []string {
	return channelLinkRegex.FindAllString(s, -1)
}

// GetPostLinks extracts all post links from s.
func GetPostLinks(s string) []string {
	return postLinkRegex.FindAllString(s, -1)
}

// GetUsernames extracts all @usernames from s.
func GetUsernames(s string) []string {
	return usernameRegex.FindAllString(s, -1)
}

// ===== Text formatting helpers ==========================
//
// Small builders that wrap text in Rubika-flavoured Markdown.

// Bold wraps text in **bold** markers.
func Bold(text string) string { return "**" + text + "**" }

// Italic wraps text in __italic__ markers.
func Italic(text string) string { return "__" + text + "__" }

// Underline wraps text in --underline-- markers.
func Underline(text string) string { return "--" + text + "--" }

// Strike wraps text in ~~strike~~ markers.
func Strike(text string) string { return "~~" + text + "~~" }

// Spoiler wraps text in ||spoiler|| markers.
func Spoiler(text string) string { return "||" + text + "||" }

// Code wraps text in `code` markers (inline monospace).
func Code(text string) string { return "`" + text + "`" }

// Pre wraps text in ```pre``` markers (block monospace).
func Pre(text string) string { return "```" + text + "```" }

// Hyperlink creates a [text](url) markdown link.
func Hyperlink(text, url string) string { return "[" + text + "](" + url + ")" }

// Mention creates a mention link for a GUID. The URL prefix is inferred
// from the GUID: u0→User, g0→Group, c0→Channel, b0→Bot.
func Mention(text, guid string) string {
	prefix := "u"
	switch {
	case strings.HasPrefix(guid, "g0"):
		prefix = "g"
	case strings.HasPrefix(guid, "c0"):
		prefix = "c"
	case strings.HasPrefix(guid, "b0"):
		prefix = "b"
	}
	return Hyperlink(text, prefix+guid)
}

// Quote creates a >quote block.
func Quote(text string) string { return "> " + text }

// ===== Time helpers ===========================

// GetServerTime fetches the current server time. The server answers with a
// timestamp (as a string or number); it is parsed robustly.
func (a *Account) GetServerTime(ctx context.Context) (time.Time, error) {
	resp, err := a.GetTime(ctx)
	if err != nil {
		return time.Time{}, err
	}
	ts := resp.AnyStr("time")
	if ts == "" {
		return time.Time{}, fmt.Errorf("no time field in response")
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q: %w", ts, err)
	}
	return time.Unix(t, 0), nil
}

// ===== GetMessageURL helper =============================

// GetMessageURL returns the share URL of a message. It is an alias for
// GetMessageShareURL with a clearer name.
func (a *Account) GetMessageURL(ctx context.Context, objectGUID, messageID string) (string, error) {
	resp, err := a.GetMessageShareURL(ctx, objectGUID, messageID)
	if err != nil {
		return "", err
	}
	return resp.AnyStr("share_url", "link", "url"), nil
}

// ===== GetChatMessages alias ========================

// GetChatMessages fetches messages from a chat. It is an alias for
// GetMessages.
func (a *Account) GetChatMessages(ctx context.Context, objectGUID, maxID string, limit int, sort string) (*events.Event, error) {
	return a.GetMessages(ctx, objectGUID, maxID, limit, sort)
}

// ===== GetMessagesBatch =============================
//
// GetMessagesBatch fetches multiple messages by ID, automatically
// chunking the request into batches of 20 (the Rubika API limit).

// GetMessagesBatch fetches messages by IDs in batches of 20.
func (a *Account) GetMessagesBatch(ctx context.Context, objectGUID string, messageIDs []string) ([]map[string]any, error) {
	var all []map[string]any
	const batchSize = 20
	for i := 0; i < len(messageIDs); i += batchSize {
		end := i + batchSize
		if end > len(messageIDs) {
			end = len(messageIDs)
		}
		batch := messageIDs[i:end]
		resp, err := a.GetMessagesByID(ctx, objectGUID, batch)
		if err != nil {
			return all, err
		}
		msgs := resp.Slice("messages")
		for _, m := range msgs {
			if mm, ok := m.(map[string]any); ok {
				all = append(all, mm)
			}
		}
	}
	return all, nil
}

// ===== GetAllMessages ===============================
//
// GetAllMessages fetches ALL messages in a chat by paginating through
// getMessagesInterval until no more messages are returned. This can be
// slow for large chats; use maxPages to bound the number of requests.

// GetAllMessages fetches all messages in a chat (up to maxPages pages).
// Set maxPages to 0 for unbounded (use with caution).
func (a *Account) GetAllMessages(ctx context.Context, objectGUID string, maxPages int) ([]map[string]any, error) {
	if maxPages <= 0 {
		maxPages = 10000 // safety bound
	}
	var all []map[string]any
	middleID := ""
	for i := 0; i < maxPages; i++ {
		resp, err := a.GetMessagesInterval(ctx, objectGUID, middleID)
		if err != nil {
			return all, err
		}
		msgs := resp.Slice("messages")
		if len(msgs) == 0 {
			break
		}
		for _, m := range msgs {
			if mm, ok := m.(map[string]any); ok {
				all = append(all, mm)
			}
		}
		// Use the last message ID as the next cursor
		if last, ok := msgs[len(msgs)-1].(map[string]any); ok {
			middleID = fmt.Sprintf("%v", last["message_id"])
		}
		if middleID == "" {
			break
		}
	}
	return all, nil
}

// ===== GetFirstMessages =============================

// GetFirstMessages fetches the first batch of messages in a chat by
// calling getMessagesInterval with an empty middleMessageID.
func (a *Account) GetFirstMessages(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.GetMessagesInterval(ctx, objectGUID, "")
}

// GetLastMessages fetches the most recent messages of a chat.
func (a *Account) GetLastMessages(ctx context.Context, objectGUID string, limit int) (*events.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.GetMessages(ctx, objectGUID, "", limit, "FromMax")
}

// GetScheduledMessages returns the scheduled messages of a chat.
func (a *Account) GetScheduledMessages(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.GetScheduledHistory(ctx, objectGUID, "")
}
