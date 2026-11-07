package account

import (
	"context"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/enums"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Chats methods (corrected payloads) ===============================

// GetChats returns the user's chats.
func (a *Account) GetChats(ctx context.Context, startID string) (*events.Event, error) {
	in := map[string]any{}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getChats", Input: in, Encrypt: true})
}

// GetChatsUpdates fetches the latest chat updates.
//
// The server requires a `state` parameter (Unix-seconds timestamp,
// typically `now - 150`). Without it the server replies
// `INVALID_INPUT: getChatsUpdates failed`, so the call always sends
// `state = now - 150`.
func (a *Account) GetChatsUpdates(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getChatsUpdates",
		Input:   map[string]any{"state": time.Now().Unix() - 150},
		Encrypt: true,
	})
}

// SeenChats marks the given chat GUIDs as seen.
func (a *Account) SeenChats(ctx context.Context, chatIDs []string) (*events.Event, error) {
	if len(chatIDs) == 0 {
		return nil, nil
	}
	seen := map[string]any{}
	for _, id := range chatIDs {
		seen[id] = int64(0)
	}
	return a.Request(ctx, RequestOptions{Method: "seenChats", Input: map[string]any{"seen_list": seen}, Encrypt: true})
}

// GetAbsObjects returns the absolute object info for the given GUIDs.
func (a *Account) GetAbsObjects(ctx context.Context, guids []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getAbsObjects", Input: map[string]any{"object_guids": guids}, Encrypt: true})
}

// SendChatActivity reports a chat activity. The API expects
// `{activity: "Typing"}` (a string), not `{activity: {type: "Typing"}}`.
func (a *Account) SendChatActivity(ctx context.Context, objectGUID string, action enums.ChatAction) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "sendChatActivity",
		Input:   map[string]any{"object_guid": objectGUID, "activity": string(action)},
		Encrypt: true,
	})
}

// SetActionChat mutes / unmutes / archives / unarchives / pins / unpins
// a chat. `action` is one of "Mute", "Unmute", "Archive", "UnArchive",
// "Pin", "Unpin".
func (a *Account) SetActionChat(ctx context.Context, objectGUID, action string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "setActionChat",
		Input:   map[string]any{"object_guid": objectGUID, "action": action},
		Encrypt: true,
	})
}

// MuteChat mutes the supplied chat.
func (a *Account) MuteChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "Mute")
}

// UnmuteChat unmutes the supplied chat.
func (a *Account) UnmuteChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "Unmute")
}

// ArchiveChat archives the supplied chat.
func (a *Account) ArchiveChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "Archive")
}

// UnarchiveChat unarchives the supplied chat.
func (a *Account) UnarchiveChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "UnArchive")
}

// PinChat pins the supplied chat to the top of the chat list.
func (a *Account) PinChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "Pin")
}

// UnpinChat unpins the supplied chat.
func (a *Account) UnpinChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.SetActionChat(ctx, objectGUID, "Unpin")
}

// DeleteChatHistory clears the chat history for the supplied GUID.
func (a *Account) DeleteChatHistory(ctx context.Context, objectGUID, lastMessageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "deleteChatHistory",
		Input:   map[string]any{"object_guid": objectGUID, "last_message_id": lastMessageID},
		Encrypt: true,
	})
}

// GetLinkFromAppURL converts an `app.rubika.ir/...` URL into a deep link.
func (a *Account) GetLinkFromAppURL(ctx context.Context, appURL string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "getLinkFromAppUrl",
		Input:      map[string]any{"app_url": appURL},
		TmpSession: true,
		Encrypt:    true,
	})
}

// SearchChatMessages searches messages inside a single chat. The API
// uses `{type: "Text"|"Hashtag"}` (no `from_user_id`).
func (a *Account) SearchChatMessages(ctx context.Context, objectGUID, searchText, searchType string) (*events.Event, error) {
	if searchType == "" {
		searchType = "Text"
	}
	return a.Request(ctx, RequestOptions{
		Method:  "searchChatMessages",
		Input:   map[string]any{"object_guid": objectGUID, "search_text": searchText, "type": searchType},
		Encrypt: true,
	})
}

// GetAvatars returns the avatar list of an object.
func (a *Account) GetAvatars(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getAvatars", Input: map[string]any{"object_guid": objectGUID}, Encrypt: true})
}

// UploadAvatar uploads a new avatar for the supplied object.
func (a *Account) UploadAvatar(ctx context.Context, objectGUID, fileID string, dcID int, accessHashRec string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "uploadAvatar",
		Input: map[string]any{
			"object_guid":     objectGUID,
			"file_id":         fileID,
			"dc_id":           dcID,
			"access_hash_rec": accessHashRec,
		},
		Encrypt: true,
	})
}

// DeleteAvatar deletes an avatar.
func (a *Account) DeleteAvatar(ctx context.Context, objectGUID, avatarID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "deleteAvatar",
		Input:   map[string]any{"object_guid": objectGUID, "avatar_id": avatarID},
		Encrypt: true,
	})
}
