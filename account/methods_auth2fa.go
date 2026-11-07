package account

import (
	"context"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Missing auth / 2FA / account management methods ================

// Logout logs out the current session.
func (a *Account) Logout(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "logout", Input: map[string]any{}, Encrypt: true})
}

// DeleteAccount requests account deletion.
func (a *Account) DeleteAccount(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "requestDeleteAccount", Input: map[string]any{}, Encrypt: true})
}

// UpgradeToAPI6 upgrades an API5 auth to API6 by uploading a new RSA
// public key. Returns the private key (PEM) that the caller should
// store alongside the auth.
func (a *Account) UpgradeToAPI6(ctx context.Context) (privateKeyPEM string, err error) {
	pub, priv, err := generateKeyPair()
	if err != nil {
		return "", err
	}
	_, err = a.Request(ctx, RequestOptions{
		Method:  "setPublicKey",
		Input:   map[string]any{"public_key": pub},
		Encrypt: true,
	})
	if err != nil {
		return "", err
	}
	return priv, nil
}

// TurnOffTwoStep disables 2FA with the current password.
func (a *Account) TurnOffTwoStep(ctx context.Context, password string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "turnOffTwoStep", Input: map[string]any{"password": password}, Encrypt: true})
}

// ChangePassword changes the 2FA password.
func (a *Account) ChangePassword(ctx context.Context, oldPassword, newPassword, hint string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "changePassword", Input: map[string]any{
		"password":     oldPassword,
		"new_password": newPassword,
		"new_hint":     hint,
	}, Encrypt: true})
}

// AbortTwoStepSetup aborts an in-progress 2FA setup.
func (a *Account) AbortTwoStepSetup(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "abortTwoStepSetup", Input: map[string]any{}, Encrypt: true})
}

// CheckTwoStepPasscode verifies the 2FA password without changing it.
func (a *Account) CheckTwoStepPasscode(ctx context.Context, password string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "checkTwoStepPasscode", Input: map[string]any{"password": password}, Encrypt: true})
}

// RequestRecoveryEmail sets or changes the 2FA recovery email.
func (a *Account) RequestRecoveryEmail(ctx context.Context, password, recoveryEmail string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "requestRecoveryEmail", Input: map[string]any{
		"password":       password,
		"recovery_email": recoveryEmail,
	}, Encrypt: true})
}

// VerifyRecoveryEmail verifies the recovery email with a code.
func (a *Account) VerifyRecoveryEmail(ctx context.Context, password, code string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "verifyRecoveryEmail", Input: map[string]any{
		"password": password,
		"code":     code,
	}, Encrypt: true})
}

// RequestForgetPassword initiates the 2FA password recovery flow.
func (a *Account) RequestForgetPassword(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "requestForgetPassword", Input: map[string]any{}, Encrypt: true})
}

// LoginTwoStepForgetPassword recovers 2FA via phone number.
func (a *Account) LoginTwoStepForgetPassword(ctx context.Context, phoneNumber string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "loginTwoStepForgetPassword", Input: map[string]any{
		"phone_number": normalizePhone(phoneNumber),
	}, TmpSession: true, Encrypt: true})
}

// LoginDisableTwoStep disables 2FA via recovery flow.
func (a *Account) LoginDisableTwoStep(ctx context.Context, phoneNumber, emailCode, forgetPasswordCodeHash string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "loginDisableTwoStep", Input: map[string]any{
		"phone_number":              normalizePhone(phoneNumber),
		"email_code":                emailCode,
		"forget_password_code_hash": forgetPasswordCodeHash,
	}, TmpSession: true, Encrypt: true})
}

// RequestChangePhoneNumber requests a phone number change.
func (a *Account) RequestChangePhoneNumber(ctx context.Context, newPhoneNumber string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "requestChangePhoneNumber", Input: map[string]any{
		"new_phone_number": normalizePhone(newPhoneNumber),
	}, Encrypt: true})
}

// VerifyChangePhoneNumber verifies the phone number change with a code.
func (a *Account) VerifyChangePhoneNumber(ctx context.Context, code, hash string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "verifyChangePhoneNumber", Input: map[string]any{
		"code": code,
		"hash": hash,
	}, Encrypt: true})
}

// ===== Missing settings / sessions methods ============================

// GetNotificationSetting returns the user's notification settings.
func (a *Account) GetNotificationSetting(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getNotificationSetting", Input: map[string]any{}, Encrypt: true})
}

// SetNotification sets notification preferences.
func (a *Account) SetNotification(ctx context.Context, settings map[string]any, updateParameters []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setSetting", Input: map[string]any{
		"settings":          settings,
		"update_parameters": updateParameters,
	}, Encrypt: true})
}

// GetUnconfirmedSessions returns the unconfirmed session list.
func (a *Account) GetUnconfirmedSessions(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getUnconfirmedSessions", Input: map[string]any{}, Encrypt: true})
}

// ActionOnUnconfirmedSession confirms or denies an unconfirmed session.
func (a *Account) ActionOnUnconfirmedSession(ctx context.Context, action, unconfirmedSessionKey string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "actionOnUnconfirmedSession", Input: map[string]any{
		"action":                  action, // "Confirm" or "Deny"
		"unconfirmed_session_key": unconfirmedSessionKey,
	}, Encrypt: true})
}

// ConfirmUnconfirmedSession is a convenience wrapper (action = "Confirm").
func (a *Account) ConfirmUnconfirmedSession(ctx context.Context, key string) (*events.Event, error) {
	return a.ActionOnUnconfirmedSession(ctx, "Confirm", key)
}

// DenyUnconfirmedSession is a convenience wrapper (action = "Deny").
func (a *Account) DenyUnconfirmedSession(ctx context.Context, key string) (*events.Event, error) {
	return a.ActionOnUnconfirmedSession(ctx, "Deny", key)
}

// ===== Saved music ====================================================

// GetSavedMusicPlaylist returns the user's saved music playlist.
func (a *Account) GetSavedMusicPlaylist(ctx context.Context, userGUID string) (*events.Event, error) {
	guid := userGUID
	if guid == "" {
		guid = a.guid
	}
	return a.Request(ctx, RequestOptions{Method: "getSavedMusicPlaylist", Input: map[string]any{
		"object_guid": guid,
	}, Encrypt: true})
}

// SetSavedMusicPlaylist replaces the entire saved music playlist.
func (a *Account) SetSavedMusicPlaylist(ctx context.Context, playlistTracks []map[string]any) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setSavedMusicPlaylist", Input: map[string]any{
		"object_guid":     a.guid,
		"playlist_tracks": playlistTracks,
	}, Encrypt: true})
}

// AddSavedMusicTrack adds a single track to the saved music playlist.
func (a *Account) AddSavedMusicTrack(ctx context.Context, track map[string]any) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "addSavedMusicTrack", Input: map[string]any{
		"object_guid": a.guid,
		"added_track": track,
	}, Encrypt: true})
}

// DeleteSavedMusicPlaylist deletes the entire playlist (or a single position).
func (a *Account) DeleteSavedMusicPlaylist(ctx context.Context, position *int) (*events.Event, error) {
	if position == nil {
		return a.SetSavedMusicPlaylist(ctx, []map[string]any{})
	}
	res, err := a.GetSavedMusicPlaylist(ctx, "")
	if err != nil {
		return nil, err
	}
	tracks := res.Slice("playlist_tracks")
	var newTracks []map[string]any
	for i, t := range tracks {
		if i == *position {
			continue
		}
		if m, ok := t.(map[string]any); ok {
			newTracks = append(newTracks, m)
		}
	}
	return a.SetSavedMusicPlaylist(ctx, newTracks)
}

// ===== Missing sticker methods =========================================

// GetStickerSetting returns the user's sticker settings.
func (a *Account) GetStickerSetting(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getStickerSetting", Input: map[string]any{}, Encrypt: true})
}

// GetMyArchivedStickerSets returns the user's archived sticker sets.
func (a *Account) GetMyArchivedStickerSets(ctx context.Context, startID string) (*events.Event, error) {
	in := map[string]any{}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getMyArchivedStickerSets", Input: in, Encrypt: true})
}

// ===== Missing reaction / wallpaper methods ==========================

// GetAvailableReactions returns the reactions available to the user.
func (a *Account) GetAvailableReactions(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getAvailableReactions", Input: map[string]any{}, Encrypt: true})
}

// GetWallpapers returns the available wallpapers.
func (a *Account) GetWallpapers(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getWallpapers", Input: map[string]any{}, Encrypt: true})
}

// ResetWallpapers resets the user's wallpaper to default.
func (a *Account) ResetWallpapers(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "resetWallpapers", Input: map[string]any{}, Encrypt: true})
}

// ===== Missing wallet / live methods =================================

// GetWalletTransactionMessage returns info about a wallet transaction.
func (a *Account) GetWalletTransactionMessage(ctx context.Context, transferID, accessTransfer string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getWalletTransactionMessage", Input: map[string]any{
		"transfer_id":     transferID,
		"access_transfer": accessTransfer,
	}, Encrypt: true})
}

// SendLive starts a live stream in a chat.
func (a *Account) SendLive(ctx context.Context, objectGUID, title string, thumbInline any) (*events.Event, error) {
	in := map[string]any{
		"object_guid": objectGUID,
		"device_type": "Software",
		"rnd":         randInt63(),
	}
	if title != "" {
		in["title"] = title
	}
	if thumbInline != nil {
		in["thumb_inline"] = thumbInline
	}
	return a.Request(ctx, RequestOptions{Method: "sendLive", Input: in, Encrypt: true})
}

// GetLiveStatus returns the status of a live stream.
func (a *Account) GetLiveStatus(ctx context.Context, liveID, accessToken string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getLiveStatus", Input: map[string]any{
		"live_id":      liveID,
		"access_token": accessToken,
	}, Encrypt: true})
}

// GetLivePlayURL returns the playback URL of a live stream.
func (a *Account) GetLivePlayURL(ctx context.Context, accessToken, liveID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getLivePlayUrl", Input: map[string]any{
		"access_token": accessToken,
		"live_id":      liveID,
	}, Encrypt: true})
}

// GetLiveComments returns the comments of a live stream.
func (a *Account) GetLiveComments(ctx context.Context, liveID, accessToken string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getLiveComments", Input: map[string]any{
		"live_id":      liveID,
		"access_token": accessToken,
	}, Encrypt: true})
}

// AddLiveComment adds a comment to a live stream.
func (a *Account) AddLiveComment(ctx context.Context, accessToken, liveID, text string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "addLiveComment", Input: map[string]any{
		"access_token": accessToken,
		"live_id":      liveID,
		"text":         text,
	}, Encrypt: true})
}

// SetCurrentLiveLocation shares a live location with map tiles.
func (a *Account) SetCurrentLiveLocation(ctx context.Context, objectGUID string, lat, lon float64, tileSideCount int, tileURLs []string, xLoc, yLoc float64) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "setCurrentLiveLocation", Input: map[string]any{
		"object_guid": objectGUID,
		"location": map[string]any{
			"latitude":  lat,
			"longitude": lon,
			"map_view": map[string]any{
				"tile_side_count": tileSideCount,
				"tile_urls":       tileURLs,
				"x_loc":           xLoc,
				"y_loc":           yLoc,
			},
		},
	}, Encrypt: true})
}

// GetMapView returns a map view for the given coordinates.
func (a *Account) GetMapView(ctx context.Context, lat, lon float64) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getMapView", Input: map[string]any{
		"location": map[string]any{"latitude": lat, "longitude": lon},
	}, Encrypt: true})
}

// generateKeyPair is a local alias to avoid importing security in this file.
func generateKeyPair() (pub, priv string, err error) {
	return generateKeyPairImpl()
}

// stripLeadingAt removes a leading @ from a username.
func stripLeadingAt(s string) string { return strings.TrimPrefix(s, "@") }
