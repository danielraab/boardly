package widgets

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

// KindNoteList shows the owner's notes of one category, of all categories or
// of none.
const KindNoteList = "note_list"

type noteListConfig struct {
	Category      notes.SelectionMode `json:"category"`
	CategoryID    *string             `json:"categoryId"`
	Subcategories bool                `json:"subcategories"`
	Sort          string              `json:"sort"`
	Limit         int                 `json:"limit"`
	Title         *string             `json:"title"`
	Size          Size                `json:"size"`
	ShowIfPublic  bool                `json:"showIfPublic"`
}

// listedNote is what a visitor gets of a shown note: only what the row
// shows. Title is the note's display title, and Category is only set for
// notes that aren't directly in the chosen category.
type listedNote struct {
	Title    *string             `json:"title"`
	Icon     *notes.Icon         `json:"icon"`
	Category *recentNoteCategory `json:"category"`
}

// ownerListedNote adds the ID the owner needs to open the note.
type ownerListedNote struct {
	ID string `json:"id"`
	listedNote
}

type noteListData struct {
	Notes []listedNote `json:"notes"`
}

type ownerNoteListData struct {
	Notes []ownerListedNote `json:"notes"`
}

// NoteList returns the note list kind. Like the link list's folder, its
// category doesn't have to exist when the layout is saved.
func NoteList(store *notes.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c noteListConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			switch c.Category {
			case notes.SelectAll, notes.SelectUncategorized:
				c.CategoryID = nil
			case notes.SelectCategory:
				if c.CategoryID == nil || *c.CategoryID == "" || len(*c.CategoryID) > maxIDLength {
					return nil, fmt.Errorf("%w: invalid category", dashboards.ErrInvalidLayout)
				}
			default:
				return nil, fmt.Errorf("%w: category must be all, none or category", dashboards.ErrInvalidLayout)
			}
			if c.Sort != notes.SortUpdated && c.Sort != notes.SortCreated && c.Sort != notes.SortTitle {
				return nil, fmt.Errorf("%w: sort must be updated, created or title", dashboards.ErrInvalidLayout)
			}
			if c.Limit < 1 || c.Limit > maxRecentLimit {
				return nil, fmt.Errorf("%w: limit must be 1 to %d", dashboards.ErrInvalidLayout, maxRecentLimit)
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c noteListConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			categories, err := store.Categories(ctx, viewer.OwnerID)
			if err != nil {
				return nil, false, err
			}
			sel := notes.Selection{Mode: c.Category, Subcategories: c.Subcategories}
			if c.Category == notes.SelectCategory {
				sel.CategoryID = *c.CategoryID
				if !slices.ContainsFunc(categories, func(cat notes.Category) bool { return cat.ID == sel.CategoryID }) {
					if viewer.IsOwner {
						return missingFolder, true, nil
					}
					return nil, false, nil
				}
			}
			found, err := store.ListedNotes(ctx, viewer.OwnerID, sel, c.Sort, c.Limit)
			if err != nil {
				return nil, false, err
			}
			rows := make([]listedNote, len(found))
			for i, n := range found {
				rows[i] = listedNote{Title: notes.DisplayTitle(n), Icon: n.Icon}
				// Only notes of another category than the chosen one say
				// which they are in.
				if n.CategoryID != nil && *n.CategoryID != sel.CategoryID {
					if cat := categoryOf(categories, *n.CategoryID, viewer.IsOwner); cat != nil {
						rows[i].Category = &recentNoteCategory{Title: cat.Title, Path: cat.Path}
					}
				}
			}
			if !viewer.IsOwner {
				return noteListData{Notes: rows}, true, nil
			}
			owned := make([]ownerListedNote, len(found))
			for i, n := range found {
				owned[i] = ownerListedNote{ID: n.ID, listedNote: rows[i]}
			}
			return ownerNoteListData{Notes: owned}, true, nil
		},
	}
}
