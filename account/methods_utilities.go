package account

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/files"
)

// ===== File utilities ===================================================

// Upload uploads a file and returns the metadata wrapped in an
// *events.Event.
func (a *Account) Upload(ctx context.Context, src files.Source, mime, fileName string) (*events.Event, error) {
	uploader := files.NewUploader(a.transport.HTTPClient(), core.DefaultChunkSize, a.maxRetries, time.Second,
		map[string]string{"auth": a.auth, "user-agent": a.userAgent})
	result, err := uploader.Upload(ctx, src, mime, fileName, func(ctx context.Context, name string, size int64, mime string) (string, int, string, string, error) {
		ev, err := a.RequestSendFile(ctx, name, size, mime)
		if err != nil {
			return "", 0, "", "", err
		}
		return ev.AnyStr("id"), ev.Int("dc_id"), ev.AnyStr("upload_url"), ev.AnyStr("access_hash_send"), nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return events.New(map[string]any{
		"mime":            result.MIME,
		"size":            result.Size,
		"dc_id":           result.DCID,
		"file_id":         result.FileID,
		"file_name":       result.FileName,
		"access_hash_rec": result.AccessHashRec,
	}), nil
}

// Download downloads a file by its file_inline metadata. When `saveAs`
// is empty the raw bytes are returned.
func (a *Account) Download(ctx context.Context, fileInline *events.Event, saveAs string, chunkSize int64, cb files.ProgressCallback) (*files.Result, error) {
	if fileInline == nil {
		return nil, fmt.Errorf("account: file_inline is nil")
	}
	if chunkSize <= 0 {
		chunkSize = core.DefaultChunkSize
	}
	d := files.NewDownloader(a.transport.HTTPClient(), chunkSize, a.maxRetries, time.Second,
		map[string]string{"auth": a.auth, "user-agent": a.userAgent})
	res, err := d.Download(ctx,
		fileInline.Int("dc_id"),
		fileInline.Str("file_id"),
		fileInline.Str("access_hash_rec"),
		int64(fileInline.Int("size")),
		saveAs, false, cb)
	if err != nil {
		return nil, err
	}
	if saveAs != "" && !strings.Contains(saveAs, ".") {
		newName := saveAs + "." + fileInline.Str("mime")
		if err := os.Rename(res.Path, newName); err == nil {
			res.Path = newName
		}
	}
	return res, nil
}

// GetMembers returns the full member list of a chat (group/channel).
func (a *Account) GetMembers(ctx context.Context, objectGUID string) ([]map[string]any, error) {
	var out []map[string]any
	startID := ""
	for {
		var res *events.Event
		var err error
		switch events.GUIDType(objectGUID) {
		case "Channel":
			res, err = a.GetChannelAllMembers(ctx, objectGUID, startID, "", 100)
		case "Group":
			res, err = a.GetGroupAllMembers(ctx, objectGUID, startID, "")
		default:
			return nil, fmt.Errorf("account: cannot list members for user chats")
		}
		if err != nil {
			return nil, err
		}
		for _, item := range res.Slice("in_chat") {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		next := res.Str("next_start_id")
		if next == "" || next == startID {
			break
		}
		startID = next
	}
	return out, nil
}
