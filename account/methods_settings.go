package account

import (
	"context"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Settings methods (corrected payloads) ============================

// UpdateProfile updates the current user's profile. Pass empty strings
// for fields you don't want to change. The `updated_parameters` list
// is built automatically from the non-empty fields.
func (a *Account) UpdateProfile(ctx context.Context, firstName, lastName, bio, birthDate string) (*events.Event, error) {
	in := map[string]any{}
	updated := []string{}
	if firstName != "" {
		in["first_name"] = firstName
		updated = append(updated, "first_name")
	}
	if lastName != "" {
		in["last_name"] = lastName
		updated = append(updated, "last_name")
	}
	if bio != "" {
		in["bio"] = bio
		updated = append(updated, "bio")
	}
	if birthDate != "" {
		in["birth_date"] = birthDate
		updated = append(updated, "birth_date")
	}
	in["updated_parameters"] = updated
	return a.Request(ctx, RequestOptions{Method: "updateProfile", Input: in, Encrypt: true})
}

// UpdateUsername changes the user's username (strips leading `@`).
func (a *Account) UpdateUsername(ctx context.Context, username string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "updateUsername", Input: map[string]any{"username": stripAt(username)}, Encrypt: true})
}

// PrivacyValue is one of "Everybody", "MyContacts", "Nobody".
type PrivacyValue string

const (
	PrivacyEverybody  PrivacyValue = "Everybody"
	PrivacyMyContacts PrivacyValue = "MyContacts"
	PrivacyNobody     PrivacyValue = "Nobody"
)

// PrivacyExceptions is the include/exclude list for a privacy setting.
type PrivacyExceptions struct {
	IncludeUsers []string
	ExcludeUsers []string
}

// SetPrivacySetting sets a single privacy setting. `settingType` is one
// of `ShowLastOnline`, `ShowPhone`, `ShowPhoto`, `CanJoinChat`,
// `ShowBirthDate`, `CanCalled`. The `exceptions` parameter is optional.
func (a *Account) SetPrivacySetting(ctx context.Context, settingType string, value PrivacyValue, exceptions *PrivacyExceptions) (*events.Event, error) {
	settings := map[string]any{settingType: string(value)}
	updateParameters := []string{settingType}
	if exceptions != nil {
		settings[settingType+"_exceptions"] = map[string]any{
			"setting_type":  settingType,
			"include_users": exceptions.IncludeUsers,
			"exclude_users": exceptions.ExcludeUsers,
		}
	}
	return a.Request(ctx, RequestOptions{
		Method: "setSetting",
		Input: map[string]any{
			"settings":          settings,
			"update_parameters": updateParameters,
		},
		Encrypt: true,
	})
}

// SetSetting is a low-level escape hatch that lets the caller build
// the `settings` dict manually.
func (a *Account) SetSetting(ctx context.Context, settings map[string]any, updateParameters []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "setSetting",
		Input: map[string]any{
			"settings":          settings,
			"update_parameters": updateParameters,
		},
		Encrypt: true,
	})
}

// GetPrivacySetting returns the user's privacy settings.
func (a *Account) GetPrivacySetting(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getPrivacySetting", Input: map[string]any{}, Encrypt: true})
}

// GetBlockedUsers returns the list of blocked users.
func (a *Account) GetBlockedUsers(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getBlockedUsers", Input: map[string]any{}, Encrypt: true})
}

// TerminateSession terminates a single active session by id.
func (a *Account) TerminateSession(ctx context.Context, sessionID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "terminateSession", Input: map[string]any{"session_id": sessionID}, Encrypt: true})
}

// TerminateOtherSessions terminates every other active session.
func (a *Account) TerminateOtherSessions(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "terminateOtherSessions", Input: map[string]any{}, Encrypt: true})
}

// GetMySessions returns the active sessions list.
func (a *Account) GetMySessions(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getMySessions", Input: map[string]any{}, Encrypt: true})
}

// SetupTwoStepVerification configures 2FA. `recoveryEmail` is optional.
func (a *Account) SetupTwoStepVerification(ctx context.Context, password, hint, recoveryEmail string) (*events.Event, error) {
	in := map[string]any{"password": password, "hint": hint}
	if recoveryEmail != "" {
		in["recovery_email"] = recoveryEmail
	}
	return a.Request(ctx, RequestOptions{Method: "setupTwoStepVerification", Input: in, Encrypt: true})
}

// GetTwoPasscodeStatus returns the 2FA status.
func (a *Account) GetTwoPasscodeStatus(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getTwoPasscodeStatus", Input: map[string]any{}, Encrypt: true})
}

// GetFolders returns the user's chat folders. `lastState` is the
// server-time baseline (defaults to 0).
func (a *Account) GetFolders(ctx context.Context, lastState int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getFolders", Input: map[string]any{"last_state": lastState}, Encrypt: true})
}

// GetSuggestedFolders returns the suggested folders.
func (a *Account) GetSuggestedFolders(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getSuggestedFolders", Input: map[string]any{}, Encrypt: true})
}

// DeleteFolder deletes a chat folder.
func (a *Account) DeleteFolder(ctx context.Context, folderID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "deleteFolder", Input: map[string]any{"folder_id": folderID}, Encrypt: true})
}

// AddFolder creates a new chat folder.
func (a *Account) AddFolder(ctx context.Context, name string, includeChats, excludeChats, includeChatTypes, excludeChatTypes []string, isAddToTop bool) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "addFolder",
		Input: map[string]any{
			"name":                 name,
			"is_add_to_top":        isAddToTop,
			"include_object_guids": includeChats,
			"exclude_object_guids": excludeChats,
			"include_chat_types":   includeChatTypes,
			"exclude_chat_types":   excludeChatTypes,
			"updated_parameters":   []string{"name", "include_chat_types", "exclude_chat_types"},
		},
		Encrypt: true,
	})
}

// ReorderFolder reorders the user's chat folders.
func (a *Account) ReorderFolder(ctx context.Context, folderIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "reorderFolder", Input: map[string]any{"folder_ids": folderIDs}, Encrypt: true})
}
