package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"famodash/internal/dashboards"
	"famodash/internal/favourites"
)

// KindLink shows one link, from the owner's favourites or a plain URL.
const KindLink = "link"

// Link sources. "New favourite" is a flow of the web app that stores a
// favourite source.
const (
	SourceFavourite = "favourite"
	SourceURL       = "url"
)

// Limits of a link widget's config.
const (
	maxFavouriteTitle = 1000
)

// linkIcons are the icons a link widget may show. Keep in sync with
// frontend/src/features/dashboards/widgets/link/icons.ts.
var linkIcons = map[string]bool{}

func init() {
	for _, icon := range strings.Fields(`
		book-open bookmark briefcase calendar camera chart-bar cloud code cpu
		database file-text film folder gamepad globe hard-drive heart home image
		inbox key lock mail map-pin message-square monitor music network newspaper
		phone printer router rss search server settings shield shopping-cart star
		terminal tv users wifi wrench`) {
		linkIcons[icon] = true
	}
}

// linkConfig is a link widget's config. URL is always set: for a favourite it
// is the favourite's last known URL, so the widget still works when the
// favourite is deleted.
type linkConfig struct {
	Source         string  `json:"source"`
	FavouriteID    string  `json:"favouriteId,omitempty"`
	URL            string  `json:"url"`
	FavouriteTitle *string `json:"favouriteTitle,omitempty"`
	Title          *string `json:"title"`
	ImageID        *string `json:"imageId"`
	Icon           *string `json:"icon"`
	Target         string  `json:"target"`
	Size           Size    `json:"size"`
}

// linkData is the current URL and title of a link widget's favourite.
type linkData struct {
	URL   string  `json:"url"`
	Title *string `json:"title"`
}

// favouriteKey caches a favourite link within one lookup.
type favouriteKey struct{ ownerID, id string }

// Link returns the link kind. On every save, a favourite's URL and title are
// copied into the config, and a widget whose favourite no longer exists is
// stored as a plain URL with the last copies.
func Link(fav *favourites.Store) dashboards.Kind {
	invalid := func(msg string) error { return fmt.Errorf("%w: %s", dashboards.ErrInvalidLayout, msg) }
	return dashboards.Kind{
		// A link is one or two lines; a fixed height only adds space or cuts it off.
		FitContent: true,
		Normalize: func(ctx context.Context, userID string, raw json.RawMessage) (json.RawMessage, error) {
			var c linkConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			switch c.Source {
			case SourceFavourite:
				if c.FavouriteID == "" || len(c.FavouriteID) > maxIDLength {
					return nil, invalid("invalid favourite")
				}
				if c.FavouriteTitle != nil && utf8.RuneCountInString(*c.FavouriteTitle) > maxFavouriteTitle {
					return nil, invalid("favourite title too long")
				}
				n, err := fav.Link(ctx, userID, c.FavouriteID)
				switch {
				case err == nil:
					c.URL, c.FavouriteTitle = *n.URL, n.Title
				case errors.Is(err, favourites.ErrNotFound):
					// Gone, or not the owner's: keep the last copies as a plain URL.
					if c.Title == nil {
						c.Title = c.FavouriteTitle
					}
					c.Source, c.FavouriteID, c.FavouriteTitle = SourceURL, "", nil
				default:
					return nil, err
				}
			case SourceURL:
				c.FavouriteID, c.FavouriteTitle = "", nil
			default:
				return nil, invalid("source must be favourite or url")
			}
			url, err := favourites.NormalizeURL(c.URL)
			if err != nil {
				return nil, invalid("invalid url")
			}
			c.URL = url
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			if c.ImageID != nil && (*c.ImageID == "" || len(*c.ImageID) > maxIDLength) {
				return nil, invalid("invalid image")
			}
			if c.Icon != nil && !linkIcons[*c.Icon] {
				return nil, invalid("unknown icon")
			}
			if c.Target != "_blank" && c.Target != "_self" {
				return nil, invalid("target must be _blank or _self")
			}
			if !c.Size.valid() {
				return nil, invalid("size must be S, M or L")
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c linkConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if c.Source != SourceFavourite {
				return nil, true, nil
			}
			n, err := dashboards.Cached(ctx, favouriteKey{viewer.OwnerID, c.FavouriteID}, func() (*favourites.Node, error) {
				n, err := fav.Link(ctx, viewer.OwnerID, c.FavouriteID)
				if errors.Is(err, favourites.ErrNotFound) {
					return nil, nil
				}
				return n, err
			})
			if err != nil {
				return nil, false, err
			}
			if n == nil {
				return linkData{URL: c.URL, Title: c.FavouriteTitle}, true, nil
			}
			return linkData{URL: *n.URL, Title: n.Title}, true, nil
		},
	}
}
