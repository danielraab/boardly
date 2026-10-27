package widgets

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
)

// KindLinkList shows the links of a folder of the owner's favourites as a list.
const KindLinkList = "link_list"

// Sort orders of a link list. Saved configs without a sort use sortFavourites.
const (
	sortFavourites   = "favourites"
	sortNewest       = "newest"
	sortAlphabetical = "alphabetical"
)

type linkListConfig struct {
	FolderID          string  `json:"folderId"`
	Title             *string `json:"title"`
	IncludeSubfolders bool    `json:"includeSubfolders"`
	// TagID, when set, keeps only the links with that tag.
	TagID        *string `json:"tagId"`
	Sort         string  `json:"sort"`
	Layout       string  `json:"layout"`
	Size         Size    `json:"size"`
	ShowIfPublic bool    `json:"showIfPublic"`
}

// listLink is one link of a link list. Folders and tags are left out, so
// visitors see nothing but the links shown.
type listLink struct {
	ID          string  `json:"id"`
	Title       *string `json:"title"`
	URL         *string `json:"url"`
	IconImageID *string `json:"iconImageId,omitempty"`
}

// linkListData is the data of a link list whose folder exists.
type linkListData struct {
	Links []listLink `json:"links"`
}

func validLayout(layout string) bool {
	return layout == "list" || layout == "compact" || layout == "tiles"
}

// missingTag is the data of a link list whose folder exists but whose tag
// doesn't.
var missingTag = map[string]string{"missing": "tag"}

// folderLinksKey caches a folder's links within one lookup.
type folderLinksKey struct {
	ownerID, folderID string
	subfolders        bool
}

// loadFolderLinks returns the links of the owner's folder with their
// favicons, or nil when the folder doesn't exist. The result is shared by the
// widgets of one lookup and must not be changed.
func loadFolderLinks(ctx context.Context, fav *favourites.Store, icons *images.Favicons, ownerID, folderID string, subfolders bool) ([]favourites.FolderLink, error) {
	return dashboards.Cached(ctx, folderLinksKey{ownerID, folderID, subfolders}, func() ([]favourites.FolderLink, error) {
		links, err := fav.FolderLinks(ctx, ownerID, folderID, subfolders)
		if errors.Is(err, favourites.ErrNotFound) {
			return nil, nil
		}
		if err == nil && icons != nil {
			nodes := make([]*favourites.Node, len(links))
			for i, l := range links {
				nodes[i] = l.Node
			}
			err = icons.SetIcons(ctx, nodes...)
		}
		return links, err
	})
}

// shownText is what a link list shows for a link: its title, or its URL.
func shownText(n *favourites.Node) string {
	if n.Title != nil && *n.Title != "" {
		return *n.Title
	}
	return deref(n.URL)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// sortLinks sorts links in place by the given sort order; the favourites
// order is kept as it is.
func sortLinks(links []favourites.FolderLink, order string) {
	switch order {
	case sortNewest:
		slices.SortStableFunc(links, func(a, b favourites.FolderLink) int {
			return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.Node.ID, a.Node.ID))
		})
	case sortAlphabetical:
		// A collator isn't safe for concurrent use, so each sort has its own.
		c := collate.New(language.Und, collate.IgnoreCase, collate.IgnoreDiacritics)
		slices.SortStableFunc(links, func(a, b favourites.FolderLink) int {
			return cmp.Or(
				c.CompareString(shownText(a.Node), shownText(b.Node)),
				cmp.Compare(deref(a.Node.URL), deref(b.Node.URL)),
				cmp.Compare(a.Node.ID, b.Node.ID),
			)
		})
	}
}

// LinkList returns the link list kind. Like the explorer's, its folder and tag
// don't have to exist when the layout is saved.
func LinkList(fav *favourites.Store, icons *images.Favicons) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c linkListConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if !validLayout(c.Layout) {
				return nil, fmt.Errorf("%w: layout must be list, compact or tiles", dashboards.ErrInvalidLayout)
			}
			if c.FolderID == "" || len(c.FolderID) > maxIDLength {
				return nil, fmt.Errorf("%w: invalid folder", dashboards.ErrInvalidLayout)
			}
			if c.TagID != nil && *c.TagID == "" {
				c.TagID = nil
			}
			if c.TagID != nil && len(*c.TagID) > maxIDLength {
				return nil, fmt.Errorf("%w: invalid tag", dashboards.ErrInvalidLayout)
			}
			switch c.Sort {
			case "":
				c.Sort = sortFavourites
			case sortFavourites, sortNewest, sortAlphabetical:
			default:
				return nil, fmt.Errorf("%w: sort must be favourites, newest or alphabetical", dashboards.ErrInvalidLayout)
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c linkListConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			found, err := loadFolderLinks(ctx, fav, icons, viewer.OwnerID, c.FolderID, c.IncludeSubfolders)
			switch {
			case err != nil:
				return nil, false, err
			case found == nil && viewer.IsOwner:
				return missingFolder, true, nil
			case found == nil:
				return nil, false, nil
			}
			// The loaded links are shared, so filter and sort a copy.
			links := slices.Clone(found)
			if c.TagID != nil {
				exists, err := fav.TagExists(ctx, viewer.OwnerID, *c.TagID)
				switch {
				case err != nil:
					return nil, false, err
				case !exists && viewer.IsOwner:
					return missingTag, true, nil
				case !exists:
					return nil, false, nil
				}
				links = slices.DeleteFunc(links, func(l favourites.FolderLink) bool {
					return !slices.Contains(l.TagIDs, *c.TagID)
				})
			}
			sortLinks(links, c.Sort)
			out := make([]listLink, len(links))
			for i, l := range links {
				out[i] = listLink{ID: l.Node.ID, Title: l.Node.Title, URL: l.Node.URL, IconImageID: l.Node.IconImageID}
			}
			return linkListData{Links: out}, true, nil
		},
	}
}
