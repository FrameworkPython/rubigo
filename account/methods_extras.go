package account

import (
	"context"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Extras methods (corrected payloads) ==============================

// GetObjectByUsername resolves a username to an object GUID. The
// leading `@` is stripped automatically.
func (a *Account) GetObjectByUsername(ctx context.Context, username string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "getObjectByUsername",
		Input:      map[string]any{"username": stripAt(username)},
		TmpSession: true,
		Encrypt:    true,
	})
}

// SearchGlobalObjects performs a global object search.
func (a *Account) SearchGlobalObjects(ctx context.Context, query string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "searchGlobalObjects", Input: map[string]any{"search_text": query}, Encrypt: true})
}

// GetProfileLinkItems returns the items shown on a profile link. The
// API uses `object_guid`, not `profile_link`.
func (a *Account) GetProfileLinkItems(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "getProfileLinkItems",
		Input:      map[string]any{"object_guid": objectGUID},
		TmpSession: true,
		Encrypt:    true,
	})
}

// GetInfo returns generic info for the supplied object — it dispatches
// to getUserInfo / getGroupInfo / getChannelInfo / getBotInfo /
// getServiceInfo based on the GUID prefix, or `getObjectByUsername`
// when the input is a username. We check prefixes directly (rather than
// relying on events.GUIDType) because GUIDType returns "User" for
// anything that is not a group or channel — including bots (b0) and
// service chats (s0), which would be mis-dispatched to getUserInfo.
func (a *Account) GetInfo(ctx context.Context, objectGUID string) (*events.Event, error) {
	switch {
	case strings.HasPrefix(objectGUID, "c0"):
		return a.GetChannelInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "g0"):
		return a.GetGroupInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "b0"):
		return a.GetBotInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "s0"):
		return a.GetServiceInfo(ctx, objectGUID)
	case strings.HasPrefix(objectGUID, "u0"):
		return a.GetUserInfo(ctx, objectGUID)
	default:
		// Otherwise treat as a username.
		return a.GetObjectByUsername(ctx, objectGUID)
	}
}

// Join joins an object. If the input looks like a link, it dispatches
// to `joinGroup` / `joinChannelByLink`. Otherwise (GUID), it dispatches
// to `joinChannelAction(channelGUID, "Join")`.
func (a *Account) Join(ctx context.Context, input string) (*events.Event, error) {
	if strings.HasPrefix(input, "http") || strings.Contains(input, "/") {
		switch events.GUIDType(input) {
		case "Channel":
			return a.JoinChannelByLink(ctx, input)
		default:
			return a.JoinGroup(ctx, input)
		}
	}
	if strings.HasPrefix(input, "c0") {
		return a.JoinChannelAction(ctx, input, "Join")
	}
	return a.JoinGroup(ctx, input)
}

// LeaveChat leaves a chat — it dispatches to `joinChannelAction(channel,
// "Remove")` (channel) or `leaveGroup` (group).
func (a *Account) LeaveChat(ctx context.Context, objectGUID string) (*events.Event, error) {
	if strings.HasPrefix(objectGUID, "c0") {
		return a.JoinChannelAction(ctx, objectGUID, "Remove")
	}
	if strings.HasPrefix(objectGUID, "g0") {
		return a.LeaveGroup(ctx, objectGUID)
	}
	return nil, nil
}

// BanMember bans a member from a chat — it dispatches based on GUID
// prefix to `banGroupMember` / `banChannelMember`.
func (a *Account) BanMember(ctx context.Context, objectGUID, memberGUID string) (*events.Event, error) {
	if strings.HasPrefix(objectGUID, "c0") {
		return a.BanChannelMember(ctx, objectGUID, memberGUID)
	}
	return a.BanGroupMember(ctx, objectGUID, memberGUID)
}

// UnbanMember unbans a member from a chat.
func (a *Account) UnbanMember(ctx context.Context, objectGUID, memberGUID string) (*events.Event, error) {
	if strings.HasPrefix(objectGUID, "c0") {
		return a.UnbanChannelMember(ctx, objectGUID, memberGUID)
	}
	return a.UnbanGroupMember(ctx, objectGUID, memberGUID)
}

// GetTime returns the server time.
func (a *Account) GetTime(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getTime", Input: map[string]any{}, TmpSession: true, Encrypt: true})
}

// GetRelatedObjects returns the related objects of a GUID.
func (a *Account) GetRelatedObjects(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "getRelatedObjects",
		Input:      map[string]any{"object_guid": objectGUID},
		TmpSession: true,
		Encrypt:    true,
	})
}

// TranscribeVoice sends a voice for transcription.
func (a *Account) TranscribeVoice(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "transcribeVoice",
		Input:   map[string]any{"object_guid": objectGUID, "message_id": messageID},
		Encrypt: true,
	})
}

// GetTranscription fetches the result of a previous transcription.
// The API expects `{message_id, transcription_id}`.
func (a *Account) GetTranscription(ctx context.Context, messageID, transcriptionID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getTranscription",
		Input:   map[string]any{"message_id": messageID, "transcription_id": transcriptionID},
		Encrypt: true,
	})
}

// UserIsAdmin reports whether `userGUID` is an admin of `objectGUID`.
// The API does not have a dedicated `userIsAdmin` method, so we paginate
// `getGroupAdminMembers` / `getChannelAdminMembers` and check membership
// client-side.
func (a *Account) UserIsAdmin(ctx context.Context, objectGUID, userGUID string) (bool, error) {
	var startID string
	for {
		var res *events.Event
		var err error
		if strings.HasPrefix(objectGUID, "c0") {
			res, err = a.GetChannelAdminMembers(ctx, objectGUID, startID)
		} else {
			res, err = a.GetGroupAdminMembers(ctx, objectGUID, startID)
		}
		if err != nil {
			return false, err
		}
		for _, item := range res.Slice("in_chat") {
			if m, ok := item.(map[string]any); ok {
				if g, _ := m["member_guid"].(string); g == userGUID {
					return true, nil
				}
				// Some responses use a different key.
				if g, _ := m["user_guid"].(string); g == userGUID {
					return true, nil
				}
			}
		}
		next := res.Str("next_start_id")
		if next == "" || next == startID {
			break
		}
		startID = next
	}
	return false, nil
}

// ReportObject reports an object for abuse. The API uses
// `{object_guid, report_type, report_description, report_type_object,
// message_id?}`.
func (a *Account) ReportObject(ctx context.Context, objectGUID, reportType, reportDescription, reportTypeObject, messageID string) (*events.Event, error) {
	in := map[string]any{
		"object_guid":        objectGUID,
		"report_type":        reportType,
		"report_description": reportDescription,
		"report_type_object": reportTypeObject,
	}
	if messageID != "" {
		in["message_id"] = messageID
	}
	return a.Request(ctx, RequestOptions{Method: "reportObject", Input: in, Encrypt: true})
}

// ReportMessage is a convenience wrapper for reporting a single
// message.
func (a *Account) ReportMessage(ctx context.Context, objectGUID, messageID, reportType, reportDescription string) (*events.Event, error) {
	return a.ReportObject(ctx, objectGUID, reportType, reportDescription, "Message", messageID)
}

// ReportChat is a convenience wrapper for reporting a chat.
func (a *Account) ReportChat(ctx context.Context, objectGUID, reportType, reportDescription string) (*events.Event, error) {
	return a.ReportObject(ctx, objectGUID, reportType, reportDescription, "Object", "")
}

// GetJoinLinks returns the join links of a chat.
func (a *Account) GetJoinLinks(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getJoinLinks", Input: map[string]any{"object_guid": objectGUID}, Encrypt: true})
}

// CreateJoinLink creates a new join link for a chat. The API uses
// `usage_limit` (not `uses_limit`) and a `request_needed` boolean.
func (a *Account) CreateJoinLink(ctx context.Context, objectGUID, title string, expirationSec, usageLimit int, requestNeeded bool) (*events.Event, error) {
	in := map[string]any{
		"object_guid":    objectGUID,
		"request_needed": requestNeeded,
		"usage_limit":    usageLimit,
	}
	if title != "" {
		in["title"] = title
	}
	if expirationSec > 0 {
		in["expire_time"] = expirationSec
	}
	return a.Request(ctx, RequestOptions{Method: "createJoinLink", Input: in, Encrypt: true})
}

// GetJoinRequests returns the pending join requests.
func (a *Account) GetJoinRequests(ctx context.Context, objectGUID, cursor string) (*events.Event, error) {
	in := map[string]any{"object_guid": objectGUID}
	if cursor != "" {
		in["cursor"] = cursor
	}
	return a.Request(ctx, RequestOptions{Method: "getJoinRequests", Input: in, Encrypt: true})
}

// ActionOnJoinRequest accepts or declines a join request. The API uses
// `{object_guid, object_type, user_guid, action}` where `object_type`
// is derived from the GUID prefix ("Group" / "Channel") and `action`
// is "Accept" or "Reject".
func (a *Account) ActionOnJoinRequest(ctx context.Context, objectGUID, userGUID, action string) (*events.Event, error) {
	objType := "Group"
	if strings.HasPrefix(objectGUID, "c0") {
		objType = "Channel"
	}
	return a.Request(ctx, RequestOptions{
		Method: "actionOnJoinRequest",
		Input: map[string]any{
			"object_guid": objectGUID,
			"object_type": objType,
			"user_guid":   userGUID,
			"action":      action,
		},
		Encrypt: true,
	})
}

// AcceptJoinRequest accepts a join request.
func (a *Account) AcceptJoinRequest(ctx context.Context, objectGUID, userGUID string) (*events.Event, error) {
	return a.ActionOnJoinRequest(ctx, objectGUID, userGUID, "Accept")
}

// RejectJoinRequest rejects a join request.
func (a *Account) RejectJoinRequest(ctx context.Context, objectGUID, userGUID string) (*events.Event, error) {
	return a.ActionOnJoinRequest(ctx, objectGUID, userGUID, "Reject")
}

// stripAt removes a leading `@` from a username.
func stripAt(s string) string {
	return strings.TrimPrefix(s, "@")
}
