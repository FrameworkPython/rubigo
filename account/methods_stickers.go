package account

import (
	"context"

	"github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/events"
)

// ===== Stickers methods (with the missing pagination/suggest_by) ======

// GetMyStickerSets returns the user's installed sticker sets.
func (a *Account) GetMyStickerSets(ctx context.Context) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getMyStickerSets", Input: map[string]any{}, Encrypt: true})
}

// GetStickerSetByID fetches a sticker set by id.
func (a *Account) GetStickerSetByID(ctx context.Context, setID string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getStickerSetByID", Input: map[string]any{"sticker_set_id": setID}, Encrypt: true})
}

// GetStickersByEmoji returns stickers matching an emoji. `suggestBy`
// defaults to "All" when empty.
func (a *Account) GetStickersByEmoji(ctx context.Context, emoji, suggestBy string) (*events.Event, error) {
	if suggestBy == "" {
		suggestBy = "All"
	}
	return a.Request(ctx, RequestOptions{
		Method:  "getStickersByEmoji",
		Input:   map[string]any{"emoji_character": emoji, "suggest_by": suggestBy},
		Encrypt: true,
	})
}

// GetStickersBySetIDs returns stickers from multiple sets.
func (a *Account) GetStickersBySetIDs(ctx context.Context, setIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{Method: "getStickersBySetIDs", Input: map[string]any{"sticker_set_ids": setIDs}, Encrypt: true})
}

// SearchStickers searches for stickers by keyword (paged).
func (a *Account) SearchStickers(ctx context.Context, query, startID string) (*events.Event, error) {
	in := map[string]any{"search_text": query}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "searchStickers", Input: in, Encrypt: true})
}

// GetTrendStickerSets returns the trending sticker sets (paged).
func (a *Account) GetTrendStickerSets(ctx context.Context, startID string) (*events.Event, error) {
	in := map[string]any{}
	if startID != "" {
		in["start_id"] = startID
	}
	return a.Request(ctx, RequestOptions{Method: "getTrendStickerSets", Input: in, Encrypt: true})
}

// ActionOnStickerSet installs/uninstalls/archives a sticker set.
// `action` is "Add", "Remove", or "Archive".
func (a *Account) ActionOnStickerSet(ctx context.Context, setID, action string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "actionOnStickerSet",
		Input:   map[string]any{"sticker_set_id": setID, "action": action},
		Encrypt: true,
	})
}

// AddStickerSet installs a sticker set (action = "Add").
func (a *Account) AddStickerSet(ctx context.Context, setID string) (*events.Event, error) {
	return a.ActionOnStickerSet(ctx, setID, "Add")
}

// DeleteStickerSet uninstalls a sticker set (action = "Remove").
func (a *Account) DeleteStickerSet(ctx context.Context, setID string) (*events.Event, error) {
	return a.ActionOnStickerSet(ctx, setID, "Remove")
}

// ArchiveStickerSet archives a sticker set (action = "Archive").
func (a *Account) ArchiveStickerSet(ctx context.Context, setID string) (*events.Event, error) {
	return a.ActionOnStickerSet(ctx, setID, "Archive")
}

// ReorderStickerSets reorders the user's installed sticker sets.
func (a *Account) ReorderStickerSets(ctx context.Context, orderedSetIDs []string) (*events.Event, error) {
	return a.Request(ctx, RequestOptions{
		Method:  "reorderStickerSets",
		Input:   map[string]any{"ordered_sticker_set_ids": orderedSetIDs},
		Encrypt: true,
	})
}
