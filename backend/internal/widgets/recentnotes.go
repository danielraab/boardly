package widgets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/notes"
)

// KindRecentNotes shows the owner's newest notes, by creation time or time of
// last change.
const KindRecentNotes = "recent_notes"

// The times a recent notes widget can sort by.
const (
	sortByCreated = "created"
	sortByUpdated = "updated"
)

type recentNotesConfig struct {
	SortBy       string  `json:"sortBy"`
	Limit        int     `json:"limit"`
	Days         *int    `json:"days"`
	Title        *string `json:"title"`
	Size         Size    `json:"size"`
	ShowIfPublic bool    `json:"showIfPublic"`
}

// recentNoteCategory is the category of a shown note. Path, the titles from
// the top-level category down, is only sent to the owner.
type recentNoteCategory struct {
	Title string   `json:"title"`
	Path  []string `json:"path,omitempty"`
}

// recentNote is what a visitor gets of a shown note: only what the row
// shows. Title is the note's display title, and Time the time sorted by.
type recentNote struct {
	Title    *string             `json:"title"`
	Icon     *notes.Icon         `json:"icon"`
	Color    *string             `json:"color"`
	Time     time.Time           `json:"time"`
	Category *recentNoteCategory `json:"category"`
}

// ownerRecentNote adds the ID the owner needs to open the note.
type ownerRecentNote struct {
	ID string `json:"id"`
	recentNote
}

type recentNotesData struct {
	Notes []recentNote `json:"notes"`
}

type ownerRecentNotesData struct {
	Notes []ownerRecentNote `json:"notes"`
}

// RecentNotes returns the recent notes kind.
func RecentNotes(store *notes.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c recentNotesConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.SortBy != sortByCreated && c.SortBy != sortByUpdated {
				return nil, fmt.Errorf("%w: sortBy must be created or updated", dashboards.ErrInvalidLayout)
			}
			if c.Limit < 1 || c.Limit > maxRecentLimit {
				return nil, fmt.Errorf("%w: limit must be 1 to %d", dashboards.ErrInvalidLayout, maxRecentLimit)
			}
			if c.Days != nil && (*c.Days < 1 || *c.Days > maxRecentDays) {
				return nil, fmt.Errorf("%w: days must be 1 to %d", dashboards.ErrInvalidLayout, maxRecentDays)
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c recentNotesConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			byUpdated := c.SortBy == sortByUpdated
			var since time.Time
			if c.Days != nil {
				since = store.Now().Add(-time.Duration(*c.Days) * 24 * time.Hour)
			}
			found, err := store.RecentNotes(ctx, viewer.OwnerID, byUpdated, since, c.Limit)
			if err != nil {
				return nil, false, err
			}
			var categories []notes.Category
			for _, n := range found {
				if n.CategoryID != nil {
					if categories, err = store.Categories(ctx, viewer.OwnerID); err != nil {
						return nil, false, err
					}
					break
				}
			}
			rows := make([]recentNote, len(found))
			for i, n := range found {
				rows[i] = recentNote{Title: notes.DisplayTitle(n), Icon: n.Icon, Color: n.Color, Time: n.CreatedAt}
				if byUpdated {
					rows[i].Time = n.UpdatedAt
				}
				if n.CategoryID != nil {
					if cat := categoryOf(categories, *n.CategoryID, viewer.IsOwner); cat != nil {
						rows[i].Category = &recentNoteCategory{Title: cat.Title, Path: cat.Path}
					}
				}
			}
			if !viewer.IsOwner {
				return recentNotesData{Notes: rows}, true, nil
			}
			owned := make([]ownerRecentNote, len(found))
			for i, n := range found {
				owned[i] = ownerRecentNote{ID: n.ID, recentNote: rows[i]}
			}
			return ownerRecentNotesData{Notes: owned}, true, nil
		},
	}
}
