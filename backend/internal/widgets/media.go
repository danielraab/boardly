package widgets

import (
	"context"
	"encoding/json"
	"fmt"

	"boardly/internal/dashboards"
	"boardly/internal/media"
	"boardly/internal/notes"
)

// KindMedia shows media sources of the owner as play buttons.
const KindMedia = "media"

// maxMediaSources bounds the sources of a media widget.
const maxMediaSources = 50

type mediaConfig struct {
	SourceIDs    []string `json:"sourceIds"`
	Title        *string  `json:"title"`
	Layout       string   `json:"layout"`
	Size         Size     `json:"size"`
	ShowIfPublic bool     `json:"showIfPublic"`
}

// mediaItem is what anyone gets of a shown source: what the player needs.
type mediaItem struct {
	ID    string      `json:"id"`
	Kind  string      `json:"kind"`
	Title string      `json:"title"`
	URL   string      `json:"url"`
	Icon  *notes.Icon `json:"icon"`
}

type mediaData struct {
	Sources []mediaItem `json:"sources"`
}

// Media returns the media kind. Like the selected notes, its sources don't
// have to exist when the layout is saved; those that don't are left out.
func Media(store *media.Store) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c mediaConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.Layout != "list" && c.Layout != "tiles" {
				return nil, fmt.Errorf("%w: layout must be list or tiles", dashboards.ErrInvalidLayout)
			}
			if len(c.SourceIDs) < 1 || len(c.SourceIDs) > maxMediaSources {
				return nil, fmt.Errorf("%w: sources must be 1 to %d", dashboards.ErrInvalidLayout, maxMediaSources)
			}
			seen := make(map[string]bool, len(c.SourceIDs))
			for _, id := range c.SourceIDs {
				if id == "" || len(id) > maxIDLength || seen[id] {
					return nil, fmt.Errorf("%w: invalid media source", dashboards.ErrInvalidLayout)
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
			var c mediaConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			sources, err := store.ByIDs(ctx, viewer.OwnerID, c.SourceIDs)
			if err != nil {
				return nil, false, err
			}
			if len(sources) == 0 && !viewer.IsOwner {
				return nil, false, nil
			}
			data := mediaData{Sources: make([]mediaItem, len(sources))}
			for i, s := range sources {
				data.Sources[i] = mediaItem{ID: s.ID, Kind: s.Kind, Title: s.Title, URL: s.URL, Icon: s.Icon}
			}
			return data, true, nil
		},
	}
}
