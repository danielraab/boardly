package widgets

import (
	"context"
	"encoding/json"
	"fmt"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

// KindSelectedNotes shows notes the owner picked, in the owner's order.
const KindSelectedNotes = "selected_notes"

type selectedNotesConfig struct {
	NoteIDs      []string `json:"noteIds"`
	Title        *string  `json:"title"`
	Size         Size     `json:"size"`
	ShowIfPublic bool     `json:"showIfPublic"`
}

// SelectedNotes returns the selected notes kind. Like the note widget's note,
// its notes don't have to exist when the layout is saved. Its rows are those
// of the note list.
func SelectedNotes(store *notes.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c selectedNotesConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if len(c.NoteIDs) < 1 || len(c.NoteIDs) > maxRecentLimit {
				return nil, fmt.Errorf("%w: notes must be 1 to %d", dashboards.ErrInvalidLayout, maxRecentLimit)
			}
			// Duplicates are rejected rather than removed, so the order is
			// always the one the owner saw.
			seen := make(map[string]bool, len(c.NoteIDs))
			for _, id := range c.NoteIDs {
				if id == "" || len(id) > maxIDLength || seen[id] {
					return nil, fmt.Errorf("%w: invalid note", dashboards.ErrInvalidLayout)
				}
				seen[id] = true
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c selectedNotesConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			found, err := store.NotesByID(ctx, viewer.OwnerID, c.NoteIDs)
			if err != nil {
				return nil, false, err
			}
			if len(found) == 0 {
				if viewer.IsOwner {
					return missingNote, true, nil
				}
				return nil, false, nil
			}
			byID := make(map[string]notes.Note, len(found))
			categorized := false
			for _, n := range found {
				byID[n.ID] = n
				categorized = categorized || n.CategoryID != nil
			}
			var categories []notes.Category
			if categorized {
				if categories, err = store.Categories(ctx, viewer.OwnerID); err != nil {
					return nil, false, err
				}
			}
			// Pinned notes first, each group in the owner's order.
			ordered := make([]notes.Note, 0, len(found))
			for _, pinned := range []bool{true, false} {
				for _, id := range c.NoteIDs {
					if n, ok := byID[id]; ok && n.Pinned == pinned {
						ordered = append(ordered, n)
					}
				}
			}
			rows := make([]listedNote, len(ordered))
			for i, n := range ordered {
				rows[i] = listedNote{Title: notes.DisplayTitle(n), Icon: n.Icon, Color: n.Color, Pinned: n.Pinned, Archived: n.Archived}
				if n.CategoryID != nil {
					if cat := categoryOf(categories, *n.CategoryID, viewer.IsOwner); cat != nil {
						rows[i].Category = &recentNoteCategory{Title: cat.Title, Path: cat.Path}
					}
				}
			}
			if !viewer.IsOwner {
				return noteListData{Notes: rows}, true, nil
			}
			owned := make([]ownerListedNote, len(ordered))
			for i, n := range ordered {
				owned[i] = ownerListedNote{ID: n.ID, listedNote: rows[i]}
			}
			return ownerNoteListData{Notes: owned}, true, nil
		},
	}
}
