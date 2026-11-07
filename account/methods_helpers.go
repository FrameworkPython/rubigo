package account

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/enums"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Bulk member operations ==========================
//
// Helpers such as BanMembers and ClearBlackList loop over the underlying
// single-target API and abort on the first error.

// BanMembers bans multiple members from a group or channel. The GUID
// prefix (g0/c0) selects the correct underlying API.
func (a *Account) BanMembers(ctx context.Context, objectGUID string, memberGUIDs []string) error {
	for _, mg := range memberGUIDs {
		if _, err := a.BanMember(ctx, objectGUID, mg); err != nil {
			return fmt.Errorf("ban %s: %w", mg, err)
		}
	}
	return nil
}

// UnbanMembers unbans multiple members from a group or channel.
func (a *Account) UnbanMembers(ctx context.Context, objectGUID string, memberGUIDs []string) error {
	for _, mg := range memberGUIDs {
		if _, err := a.UnbanMember(ctx, objectGUID, mg); err != nil {
			return fmt.Errorf("unban %s: %w", mg, err)
		}
	}
	return nil
}

// UnbanAllMembers fetches the banned-members list and unbans everyone.
// excludeGUIDs (if non-nil) lists members that should remain banned.
// The banned-members API returns members under "in_chat_members" with
// each member's GUID in the "user_guid" field (not "object_guid").
func (a *Account) UnbanAllMembers(ctx context.Context, objectGUID string, excludeGUIDs []string) error {
	exclude := make(map[string]struct{}, len(excludeGUIDs))
	for _, g := range excludeGUIDs {
		exclude[g] = struct{}{}
	}
	startID := ""
	for {
		page, err := a.GetBannedGroupMembers(ctx, objectGUID, startID)
		if err != nil {
			return err
		}
		members := page.Slice("in_chat_members")
		for _, m := range members {
			if mm, ok := m.(map[string]any); ok {
				// Banned-member objects use "user_guid" for the member's GUID.
				guid, _ := mm["user_guid"].(string)
				if guid == "" {
					continue
				}
				if _, skip := exclude[guid]; skip {
					continue
				}
				if _, err := a.UnbanMember(ctx, objectGUID, guid); err != nil {
					return err
				}
			}
		}
		ns := page.AnyStr("next_start_id")
		if ns == "" || ns == startID {
			break
		}
		startID = ns
	}
	return nil
}

// ClearBlackList is an alias for UnbanAllMembers.
func (a *Account) ClearBlackList(ctx context.Context, objectGUID string, excludeGUIDs []string) error {
	return a.UnbanAllMembers(ctx, objectGUID, excludeGUIDs)
}

// AcceptAllJoinRequests accepts every pending join request for a group
// or channel. excludeGUIDs (if non-nil) lists users to skip.
func (a *Account) AcceptAllJoinRequests(ctx context.Context, objectGUID string, excludeGUIDs []string) error {
	exclude := make(map[string]struct{}, len(excludeGUIDs))
	for _, g := range excludeGUIDs {
		exclude[g] = struct{}{}
	}
	cursor := ""
	for {
		page, err := a.GetJoinRequests(ctx, objectGUID, cursor)
		if err != nil {
			return err
		}
		reqs := page.Slice("join_requests")
		for _, r := range reqs {
			if mm, ok := r.(map[string]any); ok {
				guid, _ := mm["user_guid"].(string)
				if guid == "" {
					continue
				}
				if _, skip := exclude[guid]; skip {
					continue
				}
				if _, err := a.AcceptJoinRequest(ctx, objectGUID, guid); err != nil {
					return err
				}
			}
		}
		ns := page.AnyStr("next_start_id")
		if ns == "" || ns == cursor {
			break
		}
		cursor = ns
	}
	return nil
}

// DeleteAllAvatars deletes every avatar of an object by looping over
// getAvatars and calling deleteAvatar for each.
func (a *Account) DeleteAllAvatars(ctx context.Context, objectGUID string) error {
	resp, err := a.GetAvatars(ctx, objectGUID)
	if err != nil {
		return err
	}
	avatars := resp.Slice("avatars")
	for _, av := range avatars {
		if avm, ok := av.(map[string]any); ok {
			avatarID, _ := avm["avatar_id"].(string)
			if avatarID == "" {
				continue
			}
			if _, err := a.DeleteAvatar(ctx, objectGUID, avatarID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ===== Message helpers ================================

// GetMessageInfo fetches a single message by ID (delegates to
// GetMessagesByID with a single-element slice).
func (a *Account) GetMessageInfo(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.GetMessagesByID(ctx, objectGUID, []string{messageID})
}

// GetLastMessage returns the last message in a chat. Rubika chat-info
// responses nest the last message under the "last_message" key (an
// object containing "message_id"), not as a top-level "last_message_id"
// string. We extract the message_id from that nested object.
func (a *Account) GetLastMessage(ctx context.Context, objectGUID string) (*events.Event, error) {
	info, err := a.GetInfo(ctx, objectGUID)
	if err != nil {
		return nil, err
	}
	// Rubika nests the last message as an object under "last_message".
	lastMsg := info.EventOf("last_message")
	if lastMsg != nil {
		lastID := lastMsg.AnyStr("message_id")
		if lastID != "" {
			return a.GetMessageInfo(ctx, objectGUID, lastID)
		}
	}
	// Fallback: some chat types return last_message_id as a top-level field.
	lastID := info.AnyStr("last_message_id")
	if lastID == "" {
		return nil, fmt.Errorf("no last_message in chat info")
	}
	return a.GetMessageInfo(ctx, objectGUID, lastID)
}

// GetFirstMessage returns the first message in a chat by fetching the
// messages interval and picking the lowest message_id. Message IDs are
// compared numerically (not lexicographically) to handle IDs with
// different digit counts correctly.
func (a *Account) GetFirstMessage(ctx context.Context, objectGUID string) (*events.Event, error) {
	interval, err := a.GetMessagesInterval(ctx, objectGUID, "")
	if err != nil {
		return nil, err
	}
	msgs := interval.Slice("messages")
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages in interval")
	}
	var firstID int64 = -1
	for _, m := range msgs {
		if mm, ok := m.(map[string]any); ok {
			midStr := fmt.Sprintf("%v", mm["message_id"])
			midVal, _ := strconv.ParseInt(midStr, 10, 64)
			if firstID < 0 || (midVal > 0 && midVal < firstID) {
				firstID = midVal
			}
		}
	}
	if firstID < 0 {
		return nil, fmt.Errorf("could not determine first message id")
	}
	return a.GetMessageInfo(ctx, objectGUID, strconv.FormatInt(firstID, 10))
}

// MarkAsRead marks a chat as read by fetching the last message id and
// calling seenChats with it. The seenChats API expects a map of
// chat_guid → last_seen_message_id; passing 0 marks nothing.
func (a *Account) MarkAsRead(ctx context.Context, objectGUID string) error {
	info, err := a.GetInfo(ctx, objectGUID)
	if err != nil {
		return err
	}
	lastID := int64(0)
	if lastMsg := info.EventOf("last_message"); lastMsg != nil {
		idStr := lastMsg.AnyStr("message_id")
		if v, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			lastID = v
		}
	}
	seen := map[string]any{objectGUID: lastID}
	_, err = a.Request(ctx, RequestOptions{
		Method:  "seenChats",
		Input:   map[string]any{"seen_list": seen},
		Encrypt: true,
	})
	return err
}

// MarkAsSeen is an alias for MarkAsRead.
func (a *Account) MarkAsSeen(ctx context.Context, objectGUID string) error {
	return a.MarkAsRead(ctx, objectGUID)
}

// SearchMessages searches messages in a chat by text (client-side filter
// over getMessages). It fetches up to maxPages pages of 50 messages each and returns those
// whose text contains the query (case-insensitive). The cursor advances
// to the OLDEST message in each page (msgs[len-1]) so each page fetches
// a distinct window of messages.
func (a *Account) SearchMessages(ctx context.Context, objectGUID, query string, maxPages int) ([]map[string]any, error) {
	if maxPages <= 0 {
		maxPages = 5
	}
	query = strings.ToLower(query)
	var out []map[string]any
	maxID := ""
	for i := 0; i < maxPages; i++ {
		page, err := a.GetMessages(ctx, objectGUID, maxID, 50, "FromMax")
		if err != nil {
			return out, err
		}
		msgs := page.Slice("messages")
		if len(msgs) == 0 {
			break
		}
		for _, m := range msgs {
			mm, ok := m.(map[string]any)
			if !ok {
				continue
			}
			text, _ := mm["text"].(string)
			if strings.Contains(strings.ToLower(text), query) {
				out = append(out, mm)
			}
		}
		// Advance cursor to the OLDEST message in this page so the next
		// call fetches an earlier window. With sort=FromMax, msgs[0] is
		// the newest and msgs[len-1] is the oldest.
		if last, ok := msgs[len(msgs)-1].(map[string]any); ok {
			maxID = fmt.Sprintf("%v", last["message_id"])
		}
	}
	return out, nil
}

// ClickInlineButton clicks an inline button by sending a sendMessageAPICall
// with the button's ID.
func (a *Account) ClickInlineButton(ctx context.Context, objectGUID, messageID, buttonID string) (*events.Event, error) {
	return a.SendMessageAPICall(ctx, objectGUID, messageID, buttonID)
}

// DeleteMyMessages deletes all of the current user's messages in a chat
// by fetching and deleting them in batches. The author is identified by
// the "author_object_guid" field (not "author_guid"). The cursor advances
// to the OLDEST message in each page so the next call fetches an earlier
// window — using the newest would re-fetch nearly the same page and
// loop ~forever.
func (a *Account) DeleteMyMessages(ctx context.Context, objectGUID string) error {
	maxID := ""
	for {
		page, err := a.GetMessages(ctx, objectGUID, maxID, 50, "FromMax")
		if err != nil {
			return err
		}
		msgs := page.Slice("messages")
		if len(msgs) == 0 {
			break
		}
		var ids []string
		for _, m := range msgs {
			if mm, ok := m.(map[string]any); ok {
				// Rubika uses "author_object_guid" for the message author.
				if author, _ := mm["author_object_guid"].(string); author == a.guid {
					ids = append(ids, fmt.Sprintf("%v", mm["message_id"]))
				}
			}
		}
		if len(ids) > 0 {
			if _, err := a.DeleteMessages(ctx, objectGUID, ids); err != nil {
				return err
			}
		}
		// Advance cursor to the OLDEST message in this page.
		if last, ok := msgs[len(msgs)-1].(map[string]any); ok {
			maxID = fmt.Sprintf("%v", last["message_id"])
		}
	}
	return nil
}

// DeleteAllMessages deletes every message in a chat (regardless of
// author). Use with caution. The cursor advances to the OLDEST message
// in each page (see DeleteMyMessages for rationale).
func (a *Account) DeleteAllMessages(ctx context.Context, objectGUID string) error {
	maxID := ""
	for {
		page, err := a.GetMessages(ctx, objectGUID, maxID, 50, "FromMax")
		if err != nil {
			return err
		}
		msgs := page.Slice("messages")
		if len(msgs) == 0 {
			break
		}
		var ids []string
		for _, m := range msgs {
			if mm, ok := m.(map[string]any); ok {
				ids = append(ids, fmt.Sprintf("%v", mm["message_id"]))
			}
		}
		if len(ids) > 0 {
			if _, err := a.DeleteMessages(ctx, objectGUID, ids); err != nil {
				return err
			}
		}
		// Advance cursor to the OLDEST message in this page.
		if last, ok := msgs[len(msgs)-1].(map[string]any); ok {
			maxID = fmt.Sprintf("%v", last["message_id"])
		}
	}
	return nil
}

// ===== Profile / privacy helpers =====================

// UpdateMyName updates the current user's display name.
func (a *Account) UpdateMyName(ctx context.Context, firstName, lastName string) (*events.Event, error) {
	return a.UpdateProfile(ctx, firstName, lastName, "", "")
}

// UpdateMyBio updates the current user's bio.
func (a *Account) UpdateMyBio(ctx context.Context, bio string) (*events.Event, error) {
	return a.UpdateProfile(ctx, "", "", bio, "")
}

// UpdateMyBirthday updates the current user's birthday (format: YYYY-MM-DD).
func (a *Account) UpdateMyBirthday(ctx context.Context, birthday string) (*events.Event, error) {
	return a.UpdateProfile(ctx, "", "", "", birthday)
}

// EditShowLastOnline sets the privacy for "last online" visibility.
func (a *Account) EditShowLastOnline(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "show_my_last_online", value, exceptions)
}

// EditShowPhoneNumber sets the privacy for phone number visibility.
func (a *Account) EditShowPhoneNumber(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "show_my_phone_number", value, exceptions)
}

// EditShowProfilePhoto sets the privacy for profile photo visibility.
func (a *Account) EditShowProfilePhoto(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "show_my_profile_photo", value, exceptions)
}

// EditShowBirthday sets the privacy for birthday visibility.
func (a *Account) EditShowBirthday(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "show_my_birth_date", value, exceptions)
}

// EditCanCalledBy sets the privacy for who can call the user.
func (a *Account) EditCanCalledBy(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "can_called_by", value, exceptions)
}

// EditCanJoinChatBy sets the privacy for who can add the user to chats.
func (a *Account) EditCanJoinChatBy(ctx context.Context, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	return a.SetPrivacySetting(ctx, "can_join_chat_by", value, exceptions)
}

// UnblockAllUsers unblocks every blocked user. The blocked-users API
// returns members under "in_chat_members" with each user's GUID in the
// "user_guid" field (not "object_guid").
func (a *Account) UnblockAllUsers(ctx context.Context) error {
	resp, err := a.GetBlockedUsers(ctx)
	if err != nil {
		return err
	}
	members := resp.Slice("in_chat_members")
	for _, m := range members {
		if mm, ok := m.(map[string]any); ok {
			// Blocked-user objects use "user_guid" for the user's GUID.
			guid, _ := mm["user_guid"].(string)
			if guid == "" {
				continue
			}
			if _, err := a.UnblockUser(ctx, guid); err != nil {
				return err
			}
		}
	}
	return nil
}

// ===== Chat info dispatchers =========================

// GetGUID resolves a username, join-link, or GUID to a GUID. If the
// input is already a GUID (starts with u0/g0/c0/b0/s0) it is returned
// as-is. Join-links (rubika.ir/joing/... or /joinc/...) are resolved
// via groupPreviewByJoinLink / channelPreviewByJoinLink.
func (a *Account) GetGUID(ctx context.Context, input string) (string, error) {
	if isGUID(input) {
		return input, nil
	}
	if strings.HasPrefix(input, "@") {
		obj, err := a.GetObjectByUsername(ctx, strings.TrimPrefix(input, "@"))
		if err != nil {
			return "", err
		}
		for _, key := range []string{"user", "channel", "group", "bot"} {
			if objMap := obj.Map(key); objMap != nil {
				if guid, _ := objMap["object_guid"].(string); guid != "" {
					return guid, nil
				}
			}
		}
		return "", fmt.Errorf("could not resolve username %q", input)
	}
	// Join-link detection: /joing/ → group, /joinc/ → channel.
	if strings.Contains(input, "/joing/") {
		preview, err := a.GroupPreviewByJoinLink(ctx, input)
		if err != nil {
			return "", err
		}
		guid := preview.AnyStr("group_guid", "object_guid")
		if guid == "" {
			return "", fmt.Errorf("could not resolve group link %q", input)
		}
		return guid, nil
	}
	if strings.Contains(input, "/joinc/") {
		preview, err := a.ChannelPreviewByJoinLink(ctx, input)
		if err != nil {
			return "", err
		}
		guid := preview.AnyStr("channel_guid", "object_guid")
		if guid == "" {
			return "", fmt.Errorf("could not resolve channel link %q", input)
		}
		return guid, nil
	}
	// Fallback: try treating it as a GUID-less chat info.
	preview, err := a.GetInfo(ctx, input)
	if err != nil {
		return "", err
	}
	guid := preview.AnyStr("object_guid")
	if guid == "" {
		return "", fmt.Errorf("could not resolve link %q", input)
	}
	return guid, nil
}

// GetMyGUID returns the current user's GUID.
func (a *Account) GetMyGUID() string { return a.guid }

// ===== Voice chat helpers ============================

// StartVoiceChat creates a voice chat in a group (alias for
// CreateGroupVoiceChat with a more intuitive name).
func (a *Account) StartVoiceChat(ctx context.Context, groupGUID, title string) (*events.Event, error) {
	return a.CreateGroupVoiceChat(ctx, groupGUID, title)
}

// RequestSpeakVoiceChat requests to speak in a voice chat.
func (a *Account) RequestSpeakVoiceChat(ctx context.Context, objectGUID, voiceChatID, participantGUID string) (*events.Event, error) {
	return a.SetVoiceChatState(ctx, objectGUID, voiceChatID, string(enums.VoiceChatStateRequestSpeak), participantGUID)
}

// CancelRequestSpeakVoiceChat cancels a pending speak request.
func (a *Account) CancelRequestSpeakVoiceChat(ctx context.Context, objectGUID, voiceChatID, participantGUID string) (*events.Event, error) {
	return a.SetVoiceChatState(ctx, objectGUID, voiceChatID, string(enums.VoiceChatStateCancelRequestSpeak), participantGUID)
}

// ===== CheckJoin =====================================

// CheckJoin verifies that a user has joined a group/channel by fetching
// the member list and scanning for the user's GUID.
func (a *Account) CheckJoin(ctx context.Context, objectGUID, userGUID string) (bool, error) {
	startID := ""
	for i := 0; i < 50; i++ {
		var page *events.Event
		var err error
		if strings.HasPrefix(objectGUID, "c0") {
			page, err = a.GetChannelAllMembers(ctx, objectGUID, startID, "", 200)
		} else {
			page, err = a.GetGroupAllMembers(ctx, objectGUID, startID, "")
		}
		if err != nil {
			return false, err
		}
		// Regular member-list APIs return results under "in_chat"
		// (not "in_chat_members", which is for banned-members).
		members := page.Slice("in_chat")
		for _, m := range members {
			if mm, ok := m.(map[string]any); ok {
				// Member objects use "user_guid" for the member's GUID.
				if guid, _ := mm["user_guid"].(string); guid == userGUID {
					return true, nil
				}
			}
		}
		ns := page.AnyStr("next_start_id")
		if ns == "" || ns == startID {
			break
		}
		startID = ns
	}
	return false, nil
}

// ===== DownloadProfilePicture ==================

// DownloadProfilePicture downloads the avatar of an object. If saveAs is
// empty the raw bytes are returned; otherwise the file path is returned.
func (a *Account) DownloadProfilePicture(ctx context.Context, objectGUID string, saveAs string) (any, error) {
	resp, err := a.GetAvatars(ctx, objectGUID)
	if err != nil {
		return nil, err
	}
	avatars := resp.Slice("avatars")
	if len(avatars) == 0 {
		return nil, fmt.Errorf("no avatars for %s", objectGUID)
	}
	av, _ := avatars[0].(map[string]any)
	if av == nil {
		return nil, fmt.Errorf("invalid avatar response")
	}
	fileInline, _ := av["file_inline"].(map[string]any)
	if fileInline == nil {
		return nil, fmt.Errorf("avatar has no file_inline")
	}
	return a.Download(ctx, events.New(fileInline), saveAs, 0, nil)
}

// ===== Bulk broadcast (account-side) =================================

// BulkSendResult is the outcome of a single bulk send target.
type BulkSendResult struct {
	ObjectGUID string
	Err        error
	Success    bool
}

// SendBulkMessage sends the same text to multiple chats concurrently.
// concurrency limits in-flight sends.
func (a *Account) SendBulkMessage(ctx context.Context, objectGUIDs []string, text string, concurrency int, delayBetween time.Duration, opts ...SendOptions) []BulkSendResult {
	if concurrency <= 0 {
		concurrency = 5
	}
	results := make([]BulkSendResult, len(objectGUIDs))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, oid := range objectGUIDs {
		select {
		case <-ctx.Done():
			results[i] = BulkSendResult{ObjectGUID: oid, Err: ctx.Err()}
			continue
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(idx int, guid string) {
			defer wg.Done()
			defer func() { <-sem }()
			_, err := a.SendMessage(ctx, guid, text, opts...)
			results[idx] = BulkSendResult{ObjectGUID: guid, Err: err, Success: err == nil}
		}(i, oid)
		if delayBetween > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(delayBetween):
			}
		}
	}
	wg.Wait()
	return results
}

// ===== Utility =======================================================

// isGUID reports whether s looks like a Rubika GUID (starts with one of
// the known prefixes and is at least 34 chars long).
func isGUID(s string) bool {
	for _, p := range []string{"u0", "g0", "c0", "b0", "s0"} {
		if strings.HasPrefix(s, p) && len(s) >= 34 {
			return true
		}
	}
	return false
}
