package account

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/enums"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/files"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/text"
)

// ===== Messages methods (corrected payloads) ===========================

// SendOptions carries the optional arguments for SendMessage.
type SendOptions struct {
	ReplyToMessageID string
	Sticker          map[string]any
	FileInline       *FileInlineInput
	IsSpoil          bool
	AutoDelete       time.Duration
	ParseMode        enums.ParseMode
	Metadata         *text.Result
}

// FileInlineInput bundles everything SendMessage needs to know about
// attaching a file to the outgoing message.
type FileInlineInput struct {
	Update    *events.Event // upload metadata
	Type      string        // "File", "Image", "Voice", "Music", "Gif", "Video", "VideoMessage"
	Width     int
	Height    int
	Time      int
	Performer string
}

// SendMessage sends a message to the given object GUID.
func (a *Account) SendMessage(ctx context.Context, objectGUID, msg string, opts ...SendOptions) (*events.Event, error) {
	o := SendOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}

	// Allow magic aliases "me" / "cloud" / "self" → current user.
	if g := normalizeSelf(objectGUID, a.guid); g != objectGUID {
		objectGUID = g
	}

	// Build the base input. IMPORTANT: `reply_to_message_id` MUST be
	// omitted when empty — the server rejects the literal empty string
	// with `INVALID_INPUT: sendMessage failed`. The wire payload must
	// use JSON null (or simply omit the key) to mean "no reply";
	// sending `""` instead fails the request.
	input := map[string]any{
		"object_guid": objectGUID,
		"rnd":         randInt63(),
	}
	if o.ReplyToMessageID != "" {
		input["reply_to_message_id"] = o.ReplyToMessageID
	}
	if msg != "" {
		if o.Metadata != nil {
			mergeMetadata(input, o.Metadata)
		}
		input["text"] = strings.TrimSpace(msg)
		mode := o.ParseMode
		if mode == "" {
			mode = a.parseMode
		}
		if mode == enums.ParseModeHTML {
			md := a.formatter.Format(a.formatter.ToMarkdown(msg))
			mergeMetadata(input, &md)
		} else if mode == enums.ParseModeMarkdown {
			md := a.formatter.Format(msg)
			mergeMetadata(input, &md)
		}
	}
	if o.Sticker != nil {
		input["sticker"] = o.Sticker
	}
	if o.FileInline != nil && o.FileInline.Update != nil {
		upd := o.FileInline.Update
		// IMPORTANT: `file_id` and `access_hash_rec` may come back as
		// JSON numbers depending on the server; use AnyStr to coerce.
		// Also include `file_name` — the upload returns it and the
		// wire payload includes it.
		file := map[string]any{
			"file_id":         upd.AnyStr("file_id"),
			"dc_id":           upd.Int("dc_id"),
			"size":            upd.Int("size"),
			"mime":            upd.Str("mime"),
			"file_name":       upd.Str("file_name"),
			"access_hash_rec": upd.AnyStr("access_hash_rec"),
		}
		// When a thumbnail was generated, attach its base64 payload
		// together with the real image dimensions.
		if thumb := upd.Str("thumb_inline"); thumb != "" {
			file["thumb_inline"] = thumb
		}

		t := o.FileInline.Type
		if t == "" {
			t = "File"
		}
		// Wire type: 'VideoMessage' → 'Video' (with is_round), otherwise
		// the type as-is ('Image', 'Video', 'Voice', 'Music', 'Gif',
		// 'File'). Hard-coding 'Video' for every type caused the server
		// to reject /photo with INVALID_INPUT: sendMessage failed.
		wireType := t
		if t == "VideoMessage" {
			wireType = "Video"
			file["is_round"] = true
		}
		file["type"] = wireType
		// Sensible defaults are applied below: sending 0 here makes the
		// server reject the request.
		if o.FileInline.Time > 0 {
			file["time"] = o.FileInline.Time
		} else {
			file["time"] = 1
		}
		// Width/height: prefer the explicit user-supplied value, then
		// the actual image dimensions (from the thumbnail), then 200.
		imgWidth := upd.Int("width")
		imgHeight := upd.Int("height")
		if o.FileInline.Width > 0 {
			file["width"] = o.FileInline.Width
		} else if imgWidth > 0 {
			file["width"] = imgWidth
		} else {
			file["width"] = 200
		}
		if o.FileInline.Height > 0 {
			file["height"] = o.FileInline.Height
		} else if imgHeight > 0 {
			file["height"] = imgHeight
		} else {
			file["height"] = 200
		}
		file["music_performer"] = o.FileInline.Performer
		file["is_spoil"] = o.IsSpoil
		input["file_inline"] = file

		if a.debug {
			if jb, err := json.Marshal(file); err == nil {
				a.debugf("sendMessage file_inline=%s", string(jb))
			}
		}
	}

	// Chunk long messages into 4200-char pieces.
	if msg != "" && len(msg) > core.MaxMessageTextLength && o.FileInline == nil {
		var last *events.Event
		var err error
		for start := 0; start < len(msg); start += core.MaxMessageTextLength {
			end := start + core.MaxMessageTextLength
			if end > len(msg) {
				end = len(msg)
			}
			input["text"] = strings.TrimSpace(msg[start:end])
			last, err = a.Request(ctx, RequestOptions{Method: "sendMessage", Input: input, Encrypt: true})
			if err != nil {
				return last, err
			}
		}
		return last, nil
	}

	result, err := a.Request(ctx, RequestOptions{Method: "sendMessage", Input: input, Encrypt: true})
	if err != nil {
		// If the server rejected the message with INVALID_INPUT and
		// the payload contained metadata, retry once WITHOUT metadata.
		// The Rubika server is picky about metadata: a single bad
		// part (e.g. a Pre block whose language it doesn't like, or
		// overlapping parts) causes it to reject the whole message.
		// Retrying without metadata guarantees the text still goes
		// through — it just renders without formatting.
		if md, hasMD := input["metadata"]; hasMD && md != nil {
			if s := err.Error(); strings.Contains(s, "INVALID_INPUT") {
				plainInput := map[string]any{
					"object_guid": input["object_guid"],
					"rnd":         randInt63(),
					"text":        input["text"],
				}
				if rt, ok := input["reply_to_message_id"]; ok && rt != "" {
					plainInput["reply_to_message_id"] = rt
				}
				if a.debug {
					a.debugf("sendMessage retrying without metadata (INVALID_INPUT)")
				}
				result, err = a.Request(ctx, RequestOptions{Method: "sendMessage", Input: plainInput, Encrypt: true})
				if err != nil {
					return nil, err
				}
				return result, nil
			}
		}
		return nil, err
	}
	if o.AutoDelete > 0 {
		go func(obj, msgID string, d time.Duration) {
			ctx2, cancel := context.WithTimeout(context.Background(), d)
			defer cancel()
			_ = a.AutoDeleteMessage(ctx2, obj, msgID, d)
		}(result.Str("object_guid"), result.Str("message_id"), o.AutoDelete)
	}
	return result, nil
}

// SendPhoto sends a photo.
func (a *Account) SendPhoto(ctx context.Context, objectGUID string, photo any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, photo, caption, "Image", opts...)
}

// SendVideo sends a video.
func (a *Account) SendVideo(ctx context.Context, objectGUID string, video any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, video, caption, "Video", opts...)
}

// SendVoice sends a voice message.
func (a *Account) SendVoice(ctx context.Context, objectGUID string, voice any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, voice, caption, "Voice", opts...)
}

// SendMusic sends an audio file.
func (a *Account) SendMusic(ctx context.Context, objectGUID string, music any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, music, caption, "Music", opts...)
}

// SendGif sends a GIF.
func (a *Account) SendGif(ctx context.Context, objectGUID string, gif any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, gif, caption, "Gif", opts...)
}

// SendDocument sends a generic file.
func (a *Account) SendDocument(ctx context.Context, objectGUID string, file any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, file, caption, "File", opts...)
}

// SendVideoMessage sends a round video (video-message).
func (a *Account) SendVideoMessage(ctx context.Context, objectGUID string, video any, caption string, opts ...SendOptions) (*events.Event, error) {
	return a.sendFileType(ctx, objectGUID, video, caption, "VideoMessage", opts...)
}

func (a *Account) sendFileType(ctx context.Context, objectGUID string, file any, caption, fileType string, opts ...SendOptions) (*events.Event, error) {
	o := SendOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	var fileInline *FileInlineInput
	if upd, ok := file.(*events.Event); ok {
		fileInline = &FileInlineInput{Update: upd, Type: fileType}
	} else {
		var src files.Source
		var fileName string
		switch v := file.(type) {
		case string:
			src.Path = v
			// IMPORTANT: the server rejects `file_name` values that
			// contain path separators (e.g. "examples/mybot/hero-barber.jpg").
			// Only the basename is sent.
			fileName = filepath.Base(v)
		case []byte:
			src.Bytes = v
			fileName = "upload.bin"
		default:
			return nil, fmt.Errorf("account: unsupported file type %T", file)
		}
		uploader := files.NewUploader(a.transport.HTTPClient(), core.DefaultChunkSize, a.maxRetries, time.Second,
			map[string]string{"auth": a.auth, "user-agent": a.userAgent})
		uploadResult, err := uploader.Upload(ctx, src, "", fileName, func(ctx context.Context, name string, size int64, mime string) (string, int, string, string, error) {
			ev, err := a.RequestSendFile(ctx, name, size, mime)
			if err != nil {
				return "", 0, "", "", err
			}
			// IMPORTANT: `id` and `access_hash_send` may be returned
			// as JSON numbers (e.g. `"id": 12345`) — `ev.Str()` only
			// matches string values and would return "" for those,
			// causing the server to reject the subsequent sendMessage
			// with INVALID_INPUT. `AnyStr` coerces numbers to strings.
			fileID := ev.AnyStr("id")
			dcID := ev.Int("dc_id")
			uploadURL := ev.Str("upload_url")
			accessHashSend := ev.AnyStr("access_hash_send")
			// Debug log of the requestSendFile response.
			if a.debug {
				a.debugf("requestSendFile result: id=%q dc_id=%d upload_url=%q access_hash_send=%q",
					fileID, dcID, uploadURL, accessHashSend)
			}
			return fileID, dcID, uploadURL, accessHashSend, nil
		}, nil)
		if err != nil {
			return nil, err
		}
		// Debug log of the final upload result.
		if a.debug {
			a.debugf("upload result: file_id=%q dc_id=%d size=%d mime=%q file_name=%q access_hash_rec=%q",
				uploadResult.FileID, uploadResult.DCID, uploadResult.Size, uploadResult.MIME, uploadResult.FileName, uploadResult.AccessHashRec)
		}
		// Generate a thumbnail for image types — the server REQUIRES
		// `thumb_inline` and the REAL `width`/`height` for Image, Video,
		// Gif, VideoMessage types. Without it the server rejects the
		// request with INVALID_INPUT: sendMessage failed.
		extraFields := map[string]any{}
		needsThumb := fileType == "Image" || fileType == "Video" || fileType == "Gif" || fileType == "VideoMessage"
		if needsThumb {
			thumb, err := files.GenerateThumbnail(src)
			if err != nil {
				// Fall back to a plain File so the message can still go out.
				a.warnf("thumbnail generation failed (%v); downgrading type to File", err)
				fileType = "File"
			} else {
				extraFields["thumb_inline"] = thumb.ThumbInline
				extraFields["width"] = thumb.Width
				extraFields["height"] = thumb.Height
				a.debugf("thumbnail generated: width=%d height=%d thumb_len=%d",
					thumb.Width, thumb.Height, len(thumb.ThumbInline))
			}
		}
		// Wrap the upload result + any thumbnail fields into an *events.Event.
		upd := events.New(map[string]any{
			"mime":            uploadResult.MIME,
			"size":            uploadResult.Size,
			"dc_id":           uploadResult.DCID,
			"file_id":         uploadResult.FileID,
			"file_name":       uploadResult.FileName,
			"access_hash_rec": uploadResult.AccessHashRec,
		})
		for k, v := range extraFields {
			upd.Set(k, v)
		}
		fileInline = &FileInlineInput{Update: upd, Type: fileType}
	}
	o.FileInline = fileInline
	return a.SendMessage(ctx, objectGUID, caption, o)
}

// EditMessage edits the text of an existing message.
func (a *Account) EditMessage(ctx context.Context, objectGUID, messageID, msg string, parseMode enums.ParseMode) (*events.Event, error) {
	if parseMode == "" {
		parseMode = a.parseMode
	}
	input := map[string]any{
		"object_guid": objectGUID,
		"message_id":  messageID,
		"text":        strings.TrimSpace(msg),
	}
	if parseMode == enums.ParseModeHTML {
		md := a.formatter.Format(a.formatter.ToMarkdown(msg))
		mergeMetadata(input, &md)
	} else if parseMode == enums.ParseModeMarkdown {
		md := a.formatter.Format(msg)
		mergeMetadata(input, &md)
	}
	return a.Request(ctx, RequestOptions{Method: "editMessage", Input: input, Encrypt: true})
}

// DeleteMessages deletes messages. `deleteType` is "Global" (default,
// deletes for everyone) or "Local" (deletes only for the current user).
func (a *Account) DeleteMessages(ctx context.Context, objectGUID string, messageIDs []string) (*events.Event, error) {
	if len(messageIDs) == 0 {
		return nil, fmt.Errorf("no message IDs supplied")
	}
	return a.Request(ctx, RequestOptions{
		Method: "deleteMessages",
		Input: map[string]any{
			"object_guid": objectGUID,
			"message_ids": messageIDs,
			"type":        "Global", // delete for everyone (the server default)
		},
		Encrypt: true,
	})
}

// DeleteMessagesLocal deletes messages only for the current user
// (Local delete — other participants can still see them).
func (a *Account) DeleteMessagesLocal(ctx context.Context, objectGUID string, messageIDs []string) (*events.Event, error) {
	if len(messageIDs) == 0 {
		return nil, fmt.Errorf("no message IDs supplied")
	}
	return a.Request(ctx, RequestOptions{
		Method: "deleteMessages",
		Input: map[string]any{
			"object_guid": objectGUID,
			"message_ids": messageIDs,
			"type":        "Local",
		},
		Encrypt: true,
	})
}

// ForwardMessages forwards messages between chats. When `hideAuthor`
// is true, the original sender is hidden.
func (a *Account) ForwardMessages(ctx context.Context, fromObjectGUID string, messageIDs []string, toObjectGUID string, hideAuthor bool, isMute bool) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "forwardMessages",
		Input: map[string]any{
			"from_object_guid": fromObjectGUID,
			"message_ids":      messageIDs,
			"to_object_guid":   toObjectGUID,
			"rnd":              randInt63(),
			"drop_author":      hideAuthor,
			"is_mute":          isMute,
		},
		Encrypt: true,
	})
}

// GetMessages fetches a window of messages from an object chat.
func (a *Account) GetMessages(ctx context.Context, objectGUID, maxID string, limit int, sort string) (*events.Event, error) {
	if sort == "" {
		sort = "FromMax"
	}
	// Build the input conditionally — the Rubika server is picky about
	// extra/empty fields and may reject a literal "" max_id or 0 limit.
	in := map[string]any{"object_guid": objectGUID, "sort": sort}
	if maxID != "" {
		in["max_id"] = maxID
	}
	if limit > 0 {
		in["limit"] = limit
	}
	return a.Request(ctx, RequestOptions{
		Method:  "getMessages",
		Input:   in,
		Encrypt: true,
	})
}

// GetMessagesByID fetches specific messages by ID.
func (a *Account) GetMessagesByID(ctx context.Context, objectGUID string, messageIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getMessagesByID",
		Input:   map[string]any{"object_guid": objectGUID, "message_ids": messageIDs},
		Encrypt: true,
	})
}

// GetMessagesInterval fetches a range of messages around the supplied
// middle message ID. The API uses `middle_message_id`, not `mid` /
// `from_ticket_id`.
func (a *Account) GetMessagesInterval(ctx context.Context, objectGUID, middleMessageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getMessagesInterval",
		Input:   map[string]any{"object_guid": objectGUID, "middle_message_id": middleMessageID},
		Encrypt: true,
	})
}

// GetMessagesUpdates returns the latest message updates.
func (a *Account) GetMessagesUpdates(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getMessagesUpdates", Input: map[string]any{"object_guid": objectGUID}, Encrypt: true})
}

// SetPinMessage pins a message (action = "Pin").
func (a *Account) SetPinMessage(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "setPinMessage",
		Input:   map[string]any{"object_guid": objectGUID, "message_id": messageID, "action": "Pin"},
		Encrypt: true,
	})
}

// SetUnpinMessage unpins a message (action = "Unpin").
func (a *Account) SetUnpinMessage(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "setPinMessage",
		Input:   map[string]any{"object_guid": objectGUID, "message_id": messageID, "action": "Unpin"},
		Encrypt: true,
	})
}

// UnpinAllMessages unpins all messages in a chat.
func (a *Account) UnpinAllMessages(ctx context.Context, objectGUID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "setPinMessage",
		Input:   map[string]any{"object_guid": objectGUID, "action": "UnpinAllMessages"},
		Encrypt: true,
	})
}

// GetMessageShareURL returns the share URL of a message.
func (a *Account) GetMessageShareURL(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "getMessageShareUrl",
		Input:      map[string]any{"object_guid": objectGUID, "message_id": messageID},
		TmpSession: true,
		Encrypt:    true,
	})
}

// AutoDeleteMessage schedules a message for deletion after `delay`.
func (a *Account) AutoDeleteMessage(ctx context.Context, objectGUID, messageID string, delay time.Duration) error {
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	_, err := a.DeleteMessages(ctx, objectGUID, []string{messageID})
	return err
}

// CreatePoll creates and sends a poll message.
func (a *Account) CreatePoll(ctx context.Context, objectGUID, question string, options []string, pollType string, isAnonymous, allowMulti bool, correctIdx *int, explanation, replyToMessageID string) (*events.Event, error) {
	if pollType == "" {
		pollType = "Regular"
	}
	in := map[string]any{
		"object_guid":             objectGUID,
		"question":                question,
		"options":                 options,
		"type":                    pollType,
		"allows_multiple_answers": allowMulti,
		"is_anonymous":            isAnonymous,
		"rnd":                     randInt63(),
	}
	if correctIdx != nil {
		in["correct_option_index"] = *correctIdx
	}
	if explanation != "" {
		in["explanation"] = explanation
	}
	if replyToMessageID != "" {
		in["reply_to_message_id"] = replyToMessageID
	}
	return a.Request(ctx, RequestOptions{Method: "createPoll", Input: in, Encrypt: true})
}

// VotePoll votes on a single poll option. The API expects a single
// `selection_index: int`, not a list.
func (a *Account) VotePoll(ctx context.Context, pollID string, selectionIndex int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "votePoll",
		Input:   map[string]any{"poll_id": pollID, "selection_index": selectionIndex},
		Encrypt: true,
	})
}

// GetPollStatus fetches the status of a poll. The API only needs
// `{poll_id}` — no `object_guid`.
func (a *Account) GetPollStatus(ctx context.Context, pollID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getPollStatus", Input: map[string]any{"poll_id": pollID}, Encrypt: true})
}

// GetPollOptionVoters returns the voters of a poll option, paginated
// by `startID` (the API uses `start_id`, not `cursor`).
func (a *Account) GetPollOptionVoters(ctx context.Context, pollID string, optionIndex int, startID string) (*events.Event, error) {
	in := map[string]any{"poll_id": pollID, "selection_index": optionIndex}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getPollOptionVoters", Input: in, Encrypt: true})
}

// AddReaction adds a reaction to a message. The API uses
// `actionOnMessageReaction` with `action: "Add"` and an int `reaction_id`.
func (a *Account) AddReaction(ctx context.Context, objectGUID, messageID string, reactionID int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "actionOnMessageReaction",
		Input: map[string]any{
			"object_guid": objectGUID,
			"message_id":  messageID,
			"action":      "Add",
			"reaction_id": reactionID,
		},
		Encrypt: true,
	})
}

// RemoveReaction removes a reaction from a message.
func (a *Account) RemoveReaction(ctx context.Context, objectGUID, messageID string, reactionID int) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method: "actionOnMessageReaction",
		Input: map[string]any{
			"object_guid": objectGUID,
			"message_id":  messageID,
			"action":      "Remove",
			"reaction_id": reactionID,
		},
		Encrypt: true,
	})
}

// GetMessageReactions returns the reactions attached to a message.
func (a *Account) GetMessageReactions(ctx context.Context, objectGUID, messageID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "getMessageReactions",
		Input:   map[string]any{"object_guid": objectGUID, "message_id": messageID},
		Encrypt: true,
	})
}

// RequestSendFile asks the server to allocate an upload slot.
func (a *Account) RequestSendFile(ctx context.Context, fileName string, size int64, mime string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:     "requestSendFile",
		Input:      map[string]any{"file_name": fileName, "size": size, "mime": mime},
		TmpSession: a.auth == "",
		Encrypt:    true,
	})
}

// SendSticker sends a sticker with the full sticker payload.
func (a *Account) SendSticker(ctx context.Context, objectGUID, emojiCharacter, stickerID, stickerSetID, wHRatio string, stickerFile map[string]any, replyToMessageID string) (*events.Event, error) {
	if wHRatio == "" {
		wHRatio = "1.0"
	}
	in := map[string]any{
		"object_guid": objectGUID,
		"rnd":         randInt63(),
		"sticker": map[string]any{
			"emoji_character": emojiCharacter,
			"sticker_id":      stickerID,
			"sticker_set_id":  stickerSetID,
			"w_h_ratio":       wHRatio,
			"file":            stickerFile,
		},
	}
	if replyToMessageID != "" {
		in["reply_to_message_id"] = replyToMessageID
	}
	return a.Request(ctx, RequestOptions{Method: "sendMessage", Input: in, Encrypt: true})
}

// SendSimpleSticker sends a sticker by sticker_id only.
func (a *Account) SendSimpleSticker(ctx context.Context, objectGUID, stickerID, replyToMessageID string) (*events.Event, error) {
	in := map[string]any{
		"object_guid": objectGUID,
		"sticker_id":  stickerID,
		"rnd":         randInt63(),
	}
	if replyToMessageID != "" {
		in["reply_to_message_id"] = replyToMessageID
	}
	return a.Request(ctx, RequestOptions{Method: "sendMessage", Input: in, Encrypt: true})
}

// ===== Helpers ==========================================================

func normalizeSelf(guid, selfGUID string) string {
	switch strings.ToLower(guid) {
	case "me", "cloud", "self":
		return selfGUID
	}
	return guid
}

func mergeMetadata(in map[string]any, md *text.Result) {
	if md == nil {
		return
	}
	if md.Text != "" {
		in["text"] = md.Text
	}
	if md.Metadata != nil {
		in["metadata"] = md.Metadata
	}
}

func randInt63() int64 { return rand.Int63n(1_000_000) + 1 }

// GetChatHistory is a convenience wrapper around GetMessages that
// fetches the most recent `limit` messages from a chat, starting from
// the given `maxID` (pass "" to start from the newest). It uses
// `sort=FromMax` so messages are returned newest-first.
func (a *Account) GetChatHistory(ctx context.Context, objectGUID, maxID string, limit int) (*events.Event, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.GetMessages(ctx, objectGUID, maxID, limit, "FromMax")
}

// SearchChats searches for messages within a specific chat. `query`
// is the search term; `limit` caps the number of results (use 0 for
// the server default).
func (a *Account) SearchChats(ctx context.Context, objectGUID, query string, limit int) (*events.Event, error) {
	in := map[string]any{
		"object_guid": objectGUID,
		"search_text": query,
	}
	if limit > 0 {
		in["limit"] = limit
	}
	return a.Request(ctx, RequestOptions{Method: "searchInChat", Input: in, Encrypt: true})
}

// GetChatFolder returns a single chat folder by its ID. This is the
// per-folder counterpart of GetFolders.
func (a *Account) GetChatFolder(ctx context.Context, folderID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getFolder", Input: map[string]any{
		"folder_id": folderID,
	}, Encrypt: true})
}
