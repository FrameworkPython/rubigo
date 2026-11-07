package account

import (
	"context"
	"fmt"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Missing group methods ==========================================

// EditSlowMode sets the slow-mode interval for a group. `slowMode`
// is in seconds (0 to disable).
func (a *Account) EditSlowMode(ctx context.Context, groupGUID string, slowMode int) (*events.Event, error) {
	return a.EditGroupInfoFull(ctx, groupGUID, "", "", map[string]any{"slow_mode": slowMode}, []string{"slow_mode"})
}

// EditGroupTimer is an alias for EditSlowMode.
func (a *Account) EditGroupTimer(ctx context.Context, groupGUID string, timer int) (*events.Event, error) {
	return a.EditSlowMode(ctx, groupGUID, timer)
}

// LockGroup removes SendMessages from the default access list.
func (a *Account) LockGroup(ctx context.Context, groupGUID string) (*events.Event, error) {
	res, err := a.GetGroupDefaultAccess(ctx, groupGUID)
	if err != nil {
		return nil, err
	}
	access := res.Slice("access_list")
	var newAccess []string
	for _, item := range access {
		if s, ok := item.(string); ok && s != "SendMessages" {
			newAccess = append(newAccess, s)
		}
	}
	return a.SetGroupDefaultAccess(ctx, groupGUID, newAccess)
}

// UnlockGroup adds SendMessages to the default access list.
func (a *Account) UnlockGroup(ctx context.Context, groupGUID string) (*events.Event, error) {
	res, err := a.GetGroupDefaultAccess(ctx, groupGUID)
	if err != nil {
		return nil, err
	}
	access := res.Slice("access_list")
	var newAccess []string
	hasSend := false
	for _, item := range access {
		if s, ok := item.(string); ok {
			newAccess = append(newAccess, s)
			if s == "SendMessages" {
				hasSend = true
			}
		}
	}
	if !hasSend {
		newAccess = append(newAccess, "SendMessages")
	}
	return a.SetGroupDefaultAccess(ctx, groupGUID, newAccess)
}

// GetGroupMessageReadParticipants returns who has read a specific message.
func (a *Account) GetGroupMessageReadParticipants(ctx context.Context, groupGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getGroupMessageReadParticipants", Input: map[string]any{
		"group_guid": groupGUID,
		"message_id": messageID,
	}, Encrypt: true})
}

// EditGroupInfoFull is the full-featured version of EditGroupInfo that
// accepts extra parameters (slow_mode, event_messages, reactions, etc.).
func (a *Account) EditGroupInfoFull(ctx context.Context, groupGUID, title, description string, extraValues map[string]any, updatedParameters []string) (*events.Event, error) {
	in := map[string]any{"group_guid": groupGUID}
	if title != "" {
		in["title"] = title
	}
	if description != "" {
		in["description"] = description
	}
	for k, v := range extraValues {
		in[k] = v
	}
	in["updated_parameters"] = updatedParameters
	return a.Request(ctx, RequestOptions{Method: "editGroupInfo", Input: in, Encrypt: true})
}

// ===== Missing channel methods ========================================

// GetChannelSeenCount returns the seen count for a range of messages.
func (a *Account) GetChannelSeenCount(ctx context.Context, channelGUID, minID, maxID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getChannelSeenCount", Input: map[string]any{
		"channel_guid": channelGUID,
		"min_id":       minID,
		"max_id":       maxID,
	}, Encrypt: true})
}

// GetChannelStatistics returns channel-specific statistics. The
// Rubika API exposes a dedicated `channelStatistics` method for this
// (distinct from the account-wide `getAllStatistics`), so we call it
// directly to avoid returning global stats when the caller asked for
// channel-only data.
func (a *Account) GetChannelStatistics(ctx context.Context, channelGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "channelStatistics", Input: map[string]any{
		"channel_guid": channelGUID,
	}, Encrypt: true})
}

// SetChannelAdmin promotes or demotes a channel admin.
func (a *Account) SetChannelAdmin(ctx context.Context, channelGUID, memberGUID, action string, accessList []string) (*events.Event, error) {
	in := map[string]any{
		"channel_guid": channelGUID,
		"member_guid":  memberGUID,
		"action":       action, // "SetAdmin" or "UnsetAdmin"
	}
	if len(accessList) > 0 {
		in["access_list"] = accessList
	}
	return a.Request(ctx, RequestOptions{Method: "setChannelAdmin", Input: in, Encrypt: true})
}

// PromoteChannelAdmin promotes a channel admin.
func (a *Account) PromoteChannelAdmin(ctx context.Context, channelGUID, memberGUID string, accessList []string) (*events.Event, error) {
	return a.SetChannelAdmin(ctx, channelGUID, memberGUID, "SetAdmin", accessList)
}

// DemoteChannelAdmin demotes a channel admin.
func (a *Account) DemoteChannelAdmin(ctx context.Context, channelGUID, memberGUID string) (*events.Event, error) {
	return a.SetChannelAdmin(ctx, channelGUID, memberGUID, "UnsetAdmin", nil)
}

// SetChannelType sets the channel type (Public or Private).
func (a *Account) SetChannelType(ctx context.Context, channelGUID, channelType string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "editChannelInfo", Input: map[string]any{
		"channel_guid":       channelGUID,
		"channel_type":       channelType,
		"updated_parameters": []string{"channel_type"},
	}, Encrypt: true})
}

// EditChannelTitle is a convenience wrapper for editing just the title.
func (a *Account) EditChannelTitle(ctx context.Context, channelGUID, title string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "editChannelInfo", Input: map[string]any{
		"channel_guid":       channelGUID,
		"title":              title,
		"updated_parameters": []string{"title"},
	}, Encrypt: true})
}

// ===== Missing voice chat methods ====================================

// GetGroupVoiceChat returns info about a group voice chat.
func (a *Account) GetGroupVoiceChat(ctx context.Context, groupGUID, voiceChatID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getGroupVoiceChat", Input: map[string]any{
		"group_guid":    groupGUID,
		"voice_chat_id": voiceChatID,
	}, Encrypt: true})
}

// GetGroupVoiceChatParticipants returns the participants of a voice chat.
func (a *Account) GetGroupVoiceChatParticipants(ctx context.Context, groupGUID, voiceChatID, startID string) (*events.Event, error) {
	in := map[string]any{
		"group_guid":    groupGUID,
		"voice_chat_id": voiceChatID,
	}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getGroupVoiceChatParticipants", Input: in, Encrypt: true})
}

// GetChatVoiceChat returns the voice chat ID for a chat. Only groups
// and channels support voice chats; user/bot/service GUIDs return an
// explicit error rather than being mis-dispatched to GetGroupInfo.
func (a *Account) GetChatVoiceChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	switch {
	case strings.HasPrefix(objectGUID, "c0"):
		return a.GetChannelInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "g0"):
		return a.GetGroupInfo(ctx, objectGUID)
	default:
		return nil, fmt.Errorf("voice chats are only available for groups and channels, got GUID: %s", objectGUID)
	}
}

// Strings import guard.
