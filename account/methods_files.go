package account

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/files"
)

// ===== File helpers =====================
//
// These are convenience wrappers around Upload/Download that accept
// URLs, file paths, or raw bytes.

// UploadFromPath uploads a local file by path and returns the upload
// result event. The MIME type is auto-detected from the file extension.
func (a *Account) UploadFromPath(ctx context.Context, path string) (*UploadResult, error) {
	mime := mimeFromExt(filepath.Ext(path))
	src := files.Source{Path: path}
	ev, err := a.Upload(ctx, src, mime, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	return &UploadResult{
		Event: ev,
		MIME:  mime,
		Name:  filepath.Base(path),
	}, nil
}

// UploadFromBytes uploads raw bytes as a file.
func (a *Account) UploadFromBytes(ctx context.Context, data []byte, fileName, mime string) (*UploadResult, error) {
	if mime == "" {
		mime = mimeFromExt(filepath.Ext(fileName))
	}
	src := files.Source{Bytes: data, Size: int64(len(data))}
	ev, err := a.Upload(ctx, src, mime, fileName)
	if err != nil {
		return nil, err
	}
	return &UploadResult{
		Event: ev,
		MIME:  mime,
		Name:  fileName,
	}, nil
}

// UploadResult wraps an upload response with metadata.
type UploadResult struct {
	Event interface{ AnyStr(keys ...string) string }
	MIME  string
	Name  string
}

// DownloadToPath downloads a file inline to the given path.
func (a *Account) DownloadToPath(ctx context.Context, fileInline *events.Event, saveAs string) error {
	if fileInline == nil {
		return fmt.Errorf("no file to download")
	}
	_, err := a.Download(ctx, fileInline, saveAs, 0, nil)
	return err
}

// DownloadToBytes downloads a file inline and returns the raw bytes.
func (a *Account) DownloadToBytes(ctx context.Context, fileInline *events.Event) ([]byte, error) {
	if fileInline == nil {
		return nil, fmt.Errorf("no file to download")
	}
	result, err := a.Download(ctx, fileInline, "", 0, nil)
	if err != nil {
		return nil, err
	}
	return result.Bytes, nil
}

// UploadFromURL downloads a file from a URL and uploads it to Rubika.
// This is useful for forwarding media from other sources.
func (a *Account) UploadFromURL(ctx context.Context, url, fileName string) (*UploadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ParsRubika/4.0")
	resp, err := a.transport.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if fileName == "" {
		// Try to extract from Content-Disposition
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			fileName = extractFileName(cd)
		}
		if fileName == "" {
			fileName = "download"
		}
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = mimeFromExt(filepath.Ext(fileName))
	}
	return a.UploadFromBytes(ctx, data, fileName, mime)
}

// mimeFromExt returns a MIME type for common file extensions.
func mimeFromExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mov":
		return "video/quicktime"
	case "avi":
		return "video/x-msvideo"
	case "mkv":
		return "video/x-matroska"
	case "mp3":
		return "audio/mpeg"
	case "ogg":
		return "audio/ogg"
	case "wav":
		return "audio/wav"
	case "m4a":
		return "audio/mp4"
	case "flac":
		return "audio/flac"
	case "pdf":
		return "application/pdf"
	case "zip":
		return "application/zip"
	case "rar":
		return "application/vnd.rar"
	case "7z":
		return "application/x-7z-compressed"
	case "tar":
		return "application/x-tar"
	case "gz":
		return "application/gzip"
	case "txt":
		return "text/plain"
	case "html":
		return "text/html"
	case "css":
		return "text/css"
	case "js":
		return "application/javascript"
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xls":
		return "application/vnd.ms-excel"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "ppt":
		return "application/vnd.ms-powerpoint"
	case "pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/octet-stream"
	}
}

// extractFileName parses a Content-Disposition header for the filename.
func extractFileName(cd string) string {
	// Look for filename="..." or filename=...
	idx := strings.Index(strings.ToLower(cd), "filename=")
	if idx < 0 {
		return ""
	}
	rest := cd[idx+9:]
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "\"") {
		rest = rest[1:]
		if end := strings.Index(rest, "\""); end >= 0 {
			return rest[:end]
		}
	}
	if end := strings.IndexAny(rest, "; "); end >= 0 {
		return rest[:end]
	}
	return rest
}

// FileExists reports whether a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// FileSize returns the size of a file in bytes, or 0 on error.
func FileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
