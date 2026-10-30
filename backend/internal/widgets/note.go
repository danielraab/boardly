package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

// KindNote shows one of the owner's notes.
const KindNote = "note"

type noteConfig struct {
	NoteID       string `json:"noteId"`
	Size         Size   `json:"size"`
	ShowIfPublic bool   `json:"showIfPublic"`
}

// noteCategory is the category of a shown note. Path, the titles from the
// top-level category down, is only sent to the owner.
type noteCategory struct {
	Title string      `json:"title"`
	Icon  *notes.Icon `json:"icon"`
	Color *string     `json:"color"`
	Path  []string    `json:"path,omitempty"`
}

// noteData is what a visitor gets of a shown note: only what the widget
// shows.
type noteData struct {
	Title   *string     `json:"title"`
	Icon    *notes.Icon `json:"icon"`
	Color   *string     `json:"color"`
	Content notes.Doc   `json:"content"`
	// Attachments are those the content contains.
	Attachments []notes.NoteAttachment `json:"attachments"`
	Pinned      bool                   `json:"pinned"`
	Archived    bool                   `json:"archived"`
	Category    *noteCategory          `json:"category"`
}

// ownerNoteData adds what the owner needs to toggle the note's tasks.
type ownerNoteData struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	noteData
}

// missingNote is the data of a note widget whose note doesn't exist.
var missingNote = map[string]bool{"missing": true}

// Note returns the note kind. Like the link list's folder, the note doesn't
// have to exist when the layout is saved.
func Note(store *notes.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c noteConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.NoteID == "" || len(c.NoteID) > maxIDLength {
				return nil, fmt.Errorf("%w: invalid note", dashboards.ErrInvalidLayout)
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c noteConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			n, err := store.Note(ctx, viewer.OwnerID, c.NoteID)
			switch {
			case errors.Is(err, notes.ErrNotFound) && viewer.IsOwner:
				return missingNote, true, nil
			case errors.Is(err, notes.ErrNotFound):
				return nil, false, nil
			case err != nil:
				return nil, false, err
			}
			data := noteData{
				Title: n.Title, Icon: n.Icon, Color: n.Color, Content: n.Content, Attachments: n.Attachments,
				Pinned: n.Pinned, Archived: n.Archived,
			}
			if n.CategoryID != nil {
				categories, err := store.Categories(ctx, viewer.OwnerID)
				if err != nil {
					return nil, false, err
				}
				data.Category = categoryOf(categories, *n.CategoryID, viewer.IsOwner)
			}
			if !viewer.IsOwner {
				return data, true, nil
			}
			return ownerNoteData{ID: n.ID, Version: n.Version, noteData: data}, true, nil
		},
	}
}

// categoryOf returns the category id of categories, with its path when
// withPath is set, or nil when there is none.
func categoryOf(categories []notes.Category, id string, withPath bool) *noteCategory {
	byID := make(map[string]notes.Category, len(categories))
	for _, cat := range categories {
		byID[cat.ID] = cat
	}
	cat, ok := byID[id]
	if !ok {
		return nil
	}
	shown := &noteCategory{Title: cat.Title, Icon: cat.Icon, Color: cat.Color}
	if withPath {
		// Categories nest at most a few levels; the bound only guards
		// against a cycle.
		for c, ok := cat, true; ok && len(shown.Path) <= len(categories); {
			shown.Path = append([]string{c.Title}, shown.Path...)
			if c.ParentID == nil {
				break
			}
			c, ok = byID[*c.ParentID]
		}
	}
	return shown
}
