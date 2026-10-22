package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"famodash/internal/dashboards"
	"famodash/internal/favourites"
	"famodash/internal/images"
)

// KindFavouriteExplorer shows a folder of the owner's favourites as a tree.
const KindFavouriteExplorer = "favourite_explorer"

type explorerConfig struct {
	RootFolderID string  `json:"rootFolderId"`
	Title        *string `json:"title"`
	Size         Size    `json:"size"`
	ShowIfPublic bool    `json:"showIfPublic"`
}

// explorerData is the data of an explorer whose folder exists.
type explorerData struct {
	Children []*favourites.Node `json:"children"`
}

// missingFolder is the data of an explorer whose folder doesn't exist.
var missingFolder = map[string]bool{"missing": true}

// subtreeKey caches a resolved folder within one lookup.
type subtreeKey struct{ ownerID, folderID string }

// FavouriteExplorer returns the favourite explorer kind. The folder doesn't
// have to exist when the layout is saved, so a deleted folder never blocks a
// save; it is resolved as missing instead.
func FavouriteExplorer(fav *favourites.Store, icons *images.Favicons) dashboards.Kind {
	return dashboards.Kind{
		Normalize: func(_ context.Context, _ string, raw json.RawMessage) (json.RawMessage, error) {
			var c explorerConfig
			if err := decodeConfig(raw, &c); err != nil {
				return nil, err
			}
			if !c.Size.valid() {
				return nil, fmt.Errorf("%w: size must be S, M or L", dashboards.ErrInvalidLayout)
			}
			if c.RootFolderID == "" || len(c.RootFolderID) > maxIDLength {
				return nil, fmt.Errorf("%w: invalid root folder", dashboards.ErrInvalidLayout)
			}
			var err error
			if c.Title, err = normalizeTitle(c.Title); err != nil {
				return nil, err
			}
			return json.Marshal(c)
		},
		Resolve: func(ctx context.Context, viewer dashboards.Viewer, raw json.RawMessage) (any, bool, error) {
			var c explorerConfig
			if err := json.Unmarshal(raw, &c); err != nil {
				return nil, false, err
			}
			if !viewer.IsOwner && !c.ShowIfPublic {
				return nil, false, nil
			}
			// Only the owner's folders are looked up, so another user's folder ID is missing.
			folder, err := dashboards.Cached(ctx, subtreeKey{viewer.OwnerID, c.RootFolderID}, func() (*favourites.Node, error) {
				n, err := fav.Subtree(ctx, viewer.OwnerID, c.RootFolderID)
				if errors.Is(err, favourites.ErrNotFound) {
					return nil, nil
				}
				if err == nil && icons != nil {
					err = icons.SetIcons(ctx, n)
				}
				return n, err
			})
			switch {
			case err != nil:
				return nil, false, err
			case folder == nil && viewer.IsOwner:
				return missingFolder, true, nil
			case folder == nil:
				return nil, false, nil
			}
			children := folder.Children
			if children == nil {
				children = []*favourites.Node{}
			}
			return explorerData{Children: children}, true, nil
		},
	}
}
