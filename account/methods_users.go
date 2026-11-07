package account

import (
	"context"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Users methods ====================================================

// GetUserInfo returns information about a user. When `userGUID` is
// empty the server returns information about the current user.
func (a *Account) GetUserInfo(ctx context.Context, userGUID string) (*events.Event, error) {
	in := map[string]any{}
	if userGUID != "" {
		in["user_guid"] = userGUID
	}
	return a.Request(ctx, RequestOptions{
		Method:     "getUserInfo",
		Input:      in,
		TmpSession: a.auth == "",
		Encrypt:    true,
	})
}

// GetMe is a convenience wrapper for GetUserInfo that fetches the
// currently authenticated user.
func (a *Account) GetMe(ctx context.Context) (*events.Event, error) {
	return a.GetUserInfo(ctx, "")
}

// SetBlockUser toggles the blocked state of `userGUID`. `action` is
// "Block" or "Unblock".
func (a *Account) SetBlockUser(ctx context.Context, userGUID, action string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "setBlockUser",
		Input:   map[string]any{"user_guid": userGUID, "action": action},
		Encrypt: true,
	})
}

// BlockUser blocks the supplied user.
func (a *Account) BlockUser(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.SetBlockUser(ctx, userGUID, "Block")
}

// UnblockUser unblocks the supplied user.
func (a *Account) UnblockUser(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.SetBlockUser(ctx, userGUID, "Unblock")
}

// DeleteUserChat deletes the chat history with the supplied user GUID.
func (a *Account) DeleteUserChat(ctx context.Context, userGUID, lastMessageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "deleteUserChat",
		Input:   map[string]any{"user_guid": userGUID, "last_message_id": lastMessageID},
		Encrypt: true,
	})
}

// CheckUserUsername reports whether the supplied username exists.
func (a *Account) CheckUserUsername(ctx context.Context, username string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "checkUserUsername",
		Input:      map[string]any{"username": stripAt(username)},
		TmpSession: true,
		Encrypt:    true,
	})
}

// GetCommonGroups returns the chats shared with the supplied user.
func (a *Account) GetCommonGroups(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getCommonGroups", Input: map[string]any{"user_guid": userGUID}, Encrypt: true})
}

// GetLastOnline returns the online status of the supplied user. This
// is a composite helper that fetches the user info and reads the
// `online_time` field. It guards against nil intermediate events so a
// missing "user" key does not panic.
func (a *Account) GetLastOnline(ctx context.Context, userGUID string) (*events.Event, error) {
	res, err := a.GetUserInfo(ctx, userGUID)
	if err != nil {
		return nil, err
	}
	userEv := res.EventOf("user")
	if userEv == nil {
		return events.New(map[string]any{"type": "Unknown"}), nil
	}
	online := userEv.EventOf("online_time")
	if online == nil {
		return events.New(map[string]any{"type": "Unknown"}), nil
	}
	return online, nil
}

// GetContactsLastOnline returns the last-online time for a list of
// user GUIDs.
func (a *Account) GetContactsLastOnline(ctx context.Context, userGUIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getContactsLastOnline",
		Input:   map[string]any{"user_guids": userGUIDs},
		Encrypt: true,
	})
}

// GetTopUsers returns the user's top chat partners.
func (a *Account) GetTopUsers(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getTopChatUsers", Input: map[string]any{}, Encrypt: true})
}

// RemoveFromTopUsers removes a user from the top-chats list.
func (a *Account) RemoveFromTopUsers(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "removeFromTopChatUsers", Input: map[string]any{"user_guid": userGUID}, Encrypt: true})
}

// RequestVoiceCall initiates a voice call with the supplied user.
func (a *Account) RequestVoiceCall(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "requestCall",
		Input: map[string]any{
			"call_type":        "Voice",
			"user_guid":        userGUID,
			"library_versions": []string{"2.7.7", "2.4.4"},
			"max_layer":        92,
			"min_layer":        65,
			"sip_version":      1,
			"support_call_out": true,
		},
		Encrypt: true,
	})
}

// RequestVideoCall initiates a video call with the supplied user.
func (a *Account) RequestVideoCall(ctx context.Context, userGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "requestCall",
		Input: map[string]any{
			"call_type":        "Video",
			"user_guid":        userGUID,
			"library_versions": []string{"2.7.7", "2.4.4"},
			"max_layer":        92,
			"min_layer":        65,
			"sip_version":      1,
			"support_call_out": true,
		},
		Encrypt: true,
	})
}

// DiscardCall ends an active call. `reason` is "Missed" or "Disconnect".
func (a *Account) DiscardCall(ctx context.Context, callID string, duration int, reason string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "discardCall",
		Input: map[string]any{
			"call_id":  callID,
			"duration": duration,
			"reason":   reason,
		},
		Encrypt: true,
	})
}

// GetBotInfo returns info about a bot (GUID must start with "b0").
func (a *Account) GetBotInfo(ctx context.Context, botGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getBotInfo", Input: map[string]any{"bot_guid": botGUID}, Encrypt: true})
}

// StartBot sends a "/start" message to a bot.
func (a *Account) StartBot(ctx context.Context, botGUID string) (*events.Event, error) {
	return a.SendMessage(ctx, botGUID, "/start")
}

// StopBot stops a bot session.
func (a *Account) StopBot(ctx context.Context, botGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "stopBot", Input: map[string]any{"bot_guid": botGUID}, Encrypt: true})
}

// DeleteBotChat deletes the chat history with a bot.
func (a *Account) DeleteBotChat(ctx context.Context, botGUID, lastMessageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "deleteBotChat",
		Input:   map[string]any{"bot_guid": botGUID, "last_deleted_message_id": lastMessageID},
		Encrypt: true,
	})
}

// GetInlineBots returns info about multiple inline bots.
func (a *Account) GetInlineBots(ctx context.Context, botGUIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getInlineBots", Input: map[string]any{"bot_guids": botGUIDs}, Encrypt: true})
}

// GetServiceInfo returns info about a service chat (GUID must start
// with "s0").
func (a *Account) GetServiceInfo(ctx context.Context, serviceGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getServiceInfo", Input: map[string]any{"service_guid": serviceGUID}, Encrypt: true})
}

// DeleteServiceChat deletes the chat history with a service chat.
func (a *Account) DeleteServiceChat(ctx context.Context, serviceGUID, lastMessageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "deleteServiceChat",
		Input:   map[string]any{"service_guid": serviceGUID, "last_deleted_message_id": lastMessageID},
		Encrypt: true,
	})
}
