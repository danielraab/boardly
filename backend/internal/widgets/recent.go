package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
)

// KindRecentFavourites shows the newest links of a folder of the owner's
// favourites, including its subfolders.
const KindRecentFavourites = "recent_favourites"

// Limits of a recent favourites widget's config.
const (
	maxRecentLimit = 100
	maxRecentDays  = 365
)

// now is the current time; tests replace it.
var now = time.Now

type recentConfig struct {
	FolderID     string  `json:"folderId"`
	Limit        int     `json:"limit"`
	Days         *int    `json:"days"`
	Title        *string `json:"title"`
	Size         Size    `json:"size"`
	ShowIfPublic bool    `json:"showIfPublic"`
}

// recentLink is one link of a recent favourites widget. Like a link list's,
// it carries nothing but what is shown.
type recentLink struct {
	ID          string    `json:"id"`
	Title       *string   `json:"title"`
	URL         *string   `json:"url"`
	IconImageID *string   `json:"iconImageId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// recentData is the data of a recent favourites widget whose folder exists.
type recentData struct {
	Links []recentLink `json:"links"`
}

// RecentFavourites returns the recent favourites kind. Like the link list's,
// its folder doesn't have to exist when the layout is saved.
func RecentFavourites(fav *favourites.Store, icons *images.Favicons) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c recentConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.FolderID == "" || len(c.FolderID) > maxIDLength {
				return nil, fmt.Errorf("%w: invalid folder", dashboards.ErrInvalidLayout)
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
			var c recentConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			var since time.Time
			if c.Days != nil {
				since = now().Add(-time.Duration(*c.Days) * 24 * time.Hour)
			}
			found, err := fav.RecentLinks(ctx, viewer.OwnerID, c.FolderID, since, c.Limit)
			switch {
			case errors.Is(err, favourites.ErrNotFound) && viewer.IsOwner:
				return missingFolder, true, nil
			case errors.Is(err, favourites.ErrNotFound):
				return nil, false, nil
			case err != nil:
				return nil, false, err
			}
			nodes := make([]*favourites.Node, len(found))
			for i, l := range found {
				nodes[i] = l.Node
			}
			if icons != nil {
				if err := icons.SetIcons(ctx, nodes...); err != nil {
					return nil, false, err
				}
			}
			links := make([]recentLink, len(found))
			for i, l := range found {
				links[i] = recentLink{ID: l.Node.ID, Title: l.Node.Title, URL: l.Node.URL, IconImageID: l.Node.IconImageID, CreatedAt: l.CreatedAt}
			}
			return recentData{Links: links}, true, nil
		},
	}
}
