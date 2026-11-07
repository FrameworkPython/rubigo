package account

import (
	"context"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Voice-chat methods (dispatching to Group/Channel) ===============

// JoinVoiceChat joins a voice chat. The API method is dispatched to
// `joinGroupVoiceChat` or `joinChannelVoiceChat` based on the GUID
// prefix. `sdpOfferData` is the WebRTC SDP offer (may be empty for a
// simple presence-only join).
func (a *Account) JoinVoiceChat(ctx context.Context, objectGUID, voiceChatID, sdpOfferData string) (*events.Event, error) {
	method := "joinGroupVoiceChat"
	key := "group_guid"
	if strings.HasPrefix(objectGUID, "c0") {
		method = "joinChannelVoiceChat"
		key = "channel_guid"
	}
	in := map[string]any{
		key:                objectGUID,
		"voice_chat_id":    voiceChatID,
		"sdp_offer_data":   sdpOfferData,
		"self_object_guid": a.guid,
	}
	return a.Request(ctx, RequestOptions{Method: method, Input: in, Encrypt: true})
}

// DiscardVoiceChat closes a voice chat (dispatches to Group/Channel).
func (a *Account) DiscardVoiceChat(ctx context.Context, objectGUID, voiceChatID string) (*events.Event, error) {
	method := "discardGroupVoiceChat"
	key := "group_guid"
	if strings.HasPrefix(objectGUID, "c0") {
		method = "discardChannelVoiceChat"
		key = "channel_guid"
	}
	return a.Request(ctx, RequestOptions{
		Method:  method,
		Input:   map[string]any{key: objectGUID, "voice_chat_id": voiceChatID},
		Encrypt: true,
	})
}

// SetVoiceChatState updates the voice-chat state. `action` is "Mute"
// or "Unmute". The API method is dispatched to
// `setGroupVoiceChatState` / `setChannelVoiceChatState`.
func (a *Account) SetVoiceChatState(ctx context.Context, objectGUID, voiceChatID, action, participantGUID string) (*events.Event, error) {
	method := "setGroupVoiceChatState"
	key := "group_guid"
	if strings.HasPrefix(objectGUID, "c0") {
		method = "setChannelVoiceChatState"
		key = "channel_guid"
	}
	if participantGUID == "" {
		participantGUID = a.guid
	}
	return a.Request(ctx, RequestOptions{
		Method: method,
		Input: map[string]any{
			key:                       objectGUID,
			"voice_chat_id":           voiceChatID,
			"action":                  action,
			"participant_object_guid": participantGUID,
		},
		Encrypt: true,
	})
}

// SendGroupVoiceChatActivity sends an activity notification in a voice
// chat. `participantGUID` defaults to the current user's GUID.
func (a *Account) SendGroupVoiceChatActivity(ctx context.Context, objectGUID, voiceChatID, action, participantGUID string) (*events.Event, error) {
	if participantGUID == "" {
		participantGUID = a.guid
	}
	return a.Request(ctx, RequestOptions{
		Method: "sendGroupVoiceChatActivity",
		Input: map[string]any{
			"object_guid":             objectGUID,
			"voice_chat_id":           voiceChatID,
			"action":                  action,
			"participant_object_guid": participantGUID,
		},
		Encrypt: true,
	})
}
