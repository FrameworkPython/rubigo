package account

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Missing chat methods ============================================

// GetChatsByID returns info for multiple chat GUIDs at once.
func (a *Account) GetChatsByID(ctx context.Context, chatIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getChatsByID", Input: map[string]any{
		"object_guids": chatIDs,
	}, Encrypt: true})
}

// ClearDraft clears the draft of a single chat.
func (a *Account) ClearDraft(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "clearDrafts", Input: map[string]any{
		"action":      "Selected",
		"object_guid": objectGUID,
	}, Encrypt: true})
}

// ClearAllDrafts clears all chat drafts.
func (a *Account) ClearAllDrafts(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "clearDrafts", Input: map[string]any{
		"action": "All",
	}, Encrypt: true})
}

// GetAllDrafts returns all chat drafts.
func (a *Account) GetAllDrafts(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getAllDrafts", Input: map[string]any{}, Encrypt: true})
}

// SetChatUseTime sets the chat use-time hint (anti-rate-limit).
func (a *Account) SetChatUseTime(ctx context.Context, objectGUID string, useTime int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setChatUseTime", Input: map[string]any{
		"object_guid": objectGUID,
		"time":        useTime,
	}, Encrypt: true})
}

// SetAutoDelete sets the auto-delete timer for chats. `autoDelete` is in
// seconds (0 to disable).
func (a *Account) SetAutoDelete(ctx context.Context, objectGUIDs []string, autoDelete int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setAutoDelete", Input: map[string]any{
		"object_guids": objectGUIDs,
		"auto_delete":  autoDelete,
	}, Encrypt: true})
}

// ClickMessageURL clicks a link inside a message.
func (a *Account) ClickMessageURL(ctx context.Context, objectGUID, messageID, linkURL string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "clickMessageUrl", Input: map[string]any{
		"object_guid": objectGUID,
		"message_id":  messageID,
		"link_url":    linkURL,
	}, Encrypt: true})
}

// ClickKeyPadButton clicks a chat keypad button by its ID.
func (a *Account) ClickKeyPadButton(ctx context.Context, objectGUID, messageID, buttonID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "sendMessage", Input: map[string]any{
		"object_guid": objectGUID,
		"message_id":  messageID,
		"aux_data":    map[string]any{"button_id": buttonID},
		"rnd":         randInt63(),
	}, Encrypt: true})
}

// GetChatAds returns chat advertisements.
func (a *Account) GetChatAds(ctx context.Context, state int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getChatAds", Input: map[string]any{
		"state": state,
	}, Encrypt: true})
}

// ActionOnChatAds performs an action on a chat ad (View/Click/DontShow/Report).
func (a *Account) ActionOnChatAds(ctx context.Context, action, chatAdsID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "actionOnChatAds", Input: map[string]any{
		"action":      action,
		"chat_ads_id": chatAdsID,
	}, Encrypt: true})
}

// GetUnreadMentions returns unread mentions for a chat.
func (a *Account) GetUnreadMentions(ctx context.Context, objectGUID, maxID, minID string) (*events.Event, error) {
	in := map[string]any{"object_guid": objectGUID}
	if maxID != "" {
		in["max_id"] = maxID
	}
	if minID != "" {
		in["min_id"] = minID
	}
	return a.Request(ctx, RequestOptions{Method: "getUnreadMentions", Input: in, Encrypt: true})
}

// SetAskSpamAction sets the ask-spam action for a chat.
func (a *Account) SetAskSpamAction(ctx context.Context, objectGUID, action string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setAskSpamAction", Input: map[string]any{
		"object_guid": objectGUID,
		"action":      action, // "AddToContact", "BlockUser", "Cancel" for user; "ReportAndLeave", "Cancel" for group/channel
	}, Encrypt: true})
}

// ===== Folder pin/unpin ===============================================

// PinChatInFolder pins a chat inside a folder.
func (a *Account) PinChatInFolder(ctx context.Context, objectGUID, folderID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setPinChatInFolder", Input: map[string]any{
		"action":      "Pin",
		"folder_id":   folderID,
		"object_guid": objectGUID,
	}, Encrypt: true})
}

// UnpinChatInFolder unpins a chat from a folder.
func (a *Account) UnpinChatInFolder(ctx context.Context, objectGUID, folderID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setPinChatInFolder", Input: map[string]any{
		"action":      "Unpin",
		"folder_id":   folderID,
		"object_guid": objectGUID,
	}, Encrypt: true})
}

// ===== Owner transfer =================================================

// RequestChangeOwner requests ownership transfer to a new user.
func (a *Account) RequestChangeOwner(ctx context.Context, objectGUID, newOwnerGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "requestChangeObjectOwner", Input: map[string]any{
		"object_guid":         objectGUID,
		"new_owner_user_guid": newOwnerGUID,
	}, Encrypt: true})
}

// AcceptOwnershipRequest accepts a pending ownership transfer.
func (a *Account) AcceptOwnershipRequest(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "replyRequestObjectOwner", Input: map[string]any{
		"object_guid": objectGUID,
		"action":      "Accept",
	}, Encrypt: true})
}

// RejectOwnershipRequest rejects a pending ownership transfer.
func (a *Account) RejectOwnershipRequest(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "replyRequestObjectOwner", Input: map[string]any{
		"object_guid": objectGUID,
		"action":      "Reject",
	}, Encrypt: true})
}

// CancelChangeOwner cancels a pending ownership transfer.
func (a *Account) CancelChangeOwner(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "cancelChangeObjectOwner", Input: map[string]any{
		"object_guid": objectGUID,
	}, Encrypt: true})
}

// GetPendingOwner returns the pending owner of a chat.
func (a *Account) GetPendingOwner(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getPendingObjectOwner", Input: map[string]any{
		"object_guid": objectGUID,
	}, Encrypt: true})
}

// ===== Join links (edit / revoke) =====================================

// EditJoinLink edits an existing join link.
func (a *Account) EditJoinLink(ctx context.Context, objectGUID, hashLink, title string, expirationSec, usageLimit int, requestNeeded *bool) (*events.Event, error) {
	in := map[string]any{
		"object_guid": objectGUID,
		"hash_link":   trailingSlug(hashLink),
	}
	updated := []string{}
	if title != "" {
		in["title"] = title
		updated = append(updated, "title")
	}
	if expirationSec > 0 {
		in["expire_time"] = expirationSec
		updated = append(updated, "expire_time")
	}
	if usageLimit > 0 {
		in["usage_limit"] = usageLimit
		updated = append(updated, "usage_limit")
	}
	if requestNeeded != nil {
		in["request_needed"] = *requestNeeded
		updated = append(updated, "request_needed")
	}
	in["updated_parameters"] = updated
	return a.Request(ctx, RequestOptions{Method: "editJoinLink", Input: in, Encrypt: true})
}

// RevokeJoinLink revokes a join link.
func (a *Account) RevokeJoinLink(ctx context.Context, objectGUID, hashLink string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "revokeJoinLink", Input: map[string]any{
		"object_guid": objectGUID,
		"hash_link":   trailingSlug(hashLink),
	}, Encrypt: true})
}

// ===== Search global messages =========================================

// SearchGlobalMessages searches messages globally.
func (a *Account) SearchGlobalMessages(ctx context.Context, searchText, startID string) (*events.Event, error) {
	in := map[string]any{"search_text": searchText, "type": "Text"}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "searchGlobalMessages", Input: in, Encrypt: true})
}

// ===== Missing message methods ========================================

// SendMessageAPICall clicks an inline button on the user-side by
// sending a message with the button's aux_data.
func (a *Account) SendMessageAPICall(ctx context.Context, objectGUID, messageID, buttonID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "sendMessageAPICall", Input: map[string]any{
		"object_guid": objectGUID,
		"message_id":  messageID,
		"aux_data":    map[string]any{"button_id": buttonID},
	}, Encrypt: true})
}

// FeedbackTranscription sends feedback on a voice transcription.
func (a *Account) FeedbackTranscription(ctx context.Context, messageID, transcriptionID, feedbackType, feedbackText string) (*events.Event, error) {
	in := map[string]any{
		"message_id":       messageID,
		"transcription_id": transcriptionID,
		"feedback_type":    feedbackType, // "OK" or "Incorrect"
	}
	if feedbackText != "" {
		in["feedback_text"] = feedbackText
	}
	return a.Request(ctx, RequestOptions{Method: "feedbackTranscription", Input: in, Encrypt: true})
}

// SetPollAction stops or retracts a poll.
func (a *Account) SetPollAction(ctx context.Context, pollID, action string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setPollAction", Input: map[string]any{
		"poll_id": pollID,
		"action":  action, // "Stop" or "Retract"
	}, Encrypt: true})
}

// StopPoll stops a poll (action = "Stop").
func (a *Account) StopPoll(ctx context.Context, pollID string) (*events.Event, error) {
	return a.SetPollAction(ctx, pollID, "Stop")
}

// RetractPoll retracts a poll (action = "Retract").
func (a *Account) RetractPoll(ctx context.Context, pollID string) (*events.Event, error) {
	return a.SetPollAction(ctx, pollID, "Retract")
}

// GetPinnedMessages returns the pinned messages of a chat. It works
// for groups, channels and user (private) chats.
func (a *Account) GetPinnedMessages(ctx context.Context, objectGUID string) (*events.Event, error) {
	// Try the right info method based on the GUID prefix so user
	// (u0) chats are supported too, not just groups and channels.
	var chatInfo *events.Event
	var err error
	switch {
	case strings.HasPrefix(objectGUID, "g0"):
		chatInfo, err = a.GetGroupInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "c0"):
		chatInfo, err = a.GetChannelInfo(ctx, objectGUID)
	default:
		chatInfo, err = a.GetUserInfo(ctx, objectGUID)
	}
	if err != nil {
		return nil, err
	}
	pinnedIDs := chatInfo.Slice("pinned_message_ids")
	if len(pinnedIDs) == 0 {
		return events.New(map[string]any{"messages": []any{}}), nil
	}
	// The server may return pinned_message_ids as strings or as JSON
	// numbers; coerce both to string to avoid a type-assertion panic.
	var ids []string
	for _, id := range pinnedIDs {
		ids = append(ids, anyToStr(id))
	}
	return a.GetMessagesByID(ctx, objectGUID, ids)
}

// anyToStr coerces a scalar value (string, float64, int, json.Number)
// to its string form. Returns "" for unsupported types.
func anyToStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case json.Number:
		return x.String()
	default:
		return ""
	}
}

// GetScheduledHistory returns scheduled messages.
func (a *Account) GetScheduledHistory(ctx context.Context, objectGUID, startID string) (*events.Event, error) {
	in := map[string]any{"object_guid": objectGUID}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getScheduledHistory", Input: in, Encrypt: true})
}

// SendNowScheduledMessage sends a scheduled message immediately.
func (a *Account) SendNowScheduledMessage(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "sendNowScheduledMessage", Input: map[string]any{
		"object_guid": objectGUID,
		"message_id":  messageID,
	}, Encrypt: true})
}

// SendRubinoPost shares a Rubino post to a chat.
func (a *Account) SendRubinoPost(ctx context.Context, objectGUID, postID, postProfileID string, isMute bool) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "sendRubinoPost", Input: map[string]any{
		"object_guid":     objectGUID,
		"post_id":         postID,
		"post_profile_id": postProfileID,
		"rnd":             randInt63(),
		"is_mute":         isMute,
	}, Encrypt: true})
}

// SendRubinoProduct shares a Rubino product to a chat.
func (a *Account) SendRubinoProduct(ctx context.Context, objectGUID, storeID, productID string, productVarieties []map[string]any) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "sendRubinoProduct", Input: map[string]any{
		"object_guid":       objectGUID,
		"store_id":          storeID,
		"product_id":        productID,
		"product_varieties": productVarieties,
		"rnd":               randInt63(),
	}, Encrypt: true})
}

// SendRubinoStory replies to or shares a Rubino story.
func (a *Account) SendRubinoStory(ctx context.Context, objectGUID, storyID, storyProfileID, replyText, storyType string, isMute bool) (*events.Event, error) {
	if storyType == "" {
		storyType = "Reply"
	}
	return a.Request(ctx, RequestOptions{Method: "sendRubinoStory", Input: map[string]any{
		"object_guid":      objectGUID,
		"story_id":         storyID,
		"story_profile_id": storyProfileID,
		"reply_text":       replyText,
		"type":             storyType, // "Reply" or "Direct"
		"rnd":              randInt63(),
		"is_mute":          isMute,
	}, Encrypt: true})
}

// Import guard for strings import.
