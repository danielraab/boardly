package widgets

import (
	"context"
	"encoding/json"
	"fmt"

	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
)

// KindLinkList shows the links of a folder of the owner's favourites as a list.
const KindLinkList = "link_list"

type linkListConfig struct {
	FolderID          string  `json:"folderId"`
	Title             *string `json:"title"`
	IncludeSubfolders bool    `json:"includeSubfolders"`
	Layout            string  `json:"layout"`
	Size              Size    `json:"size"`
	ShowIfPublic      bool    `json:"showIfPublic"`
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

// flattenLinks appends the links in nodes to links, depth first, including
// those in subfolders when subfolders is set.
func flattenLinks(links []listLink, nodes []*favourites.Node, subfolders bool) []listLink {
	for _, n := range nodes {
		switch {
		case n.Kind == favourites.KindLink:
			links = append(links, listLink{ID: n.ID, Title: n.Title, URL: n.URL, IconImageID: n.IconImageID})
		case subfolders:
			links = flattenLinks(links, n.Children, true)
		}
	}
	return links
}

// LinkList returns the link list kind. Like the explorer's, its folder doesn't
// have to exist when the layout is saved.
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
			folder, err := loadFolder(ctx, fav, icons, viewer.OwnerID, c.FolderID)
			switch {
			case err != nil:
				return nil, false, err
			case folder == nil && viewer.IsOwner:
				return missingFolder, true, nil
			case folder == nil:
				return nil, false, nil
			}
			return linkListData{Links: flattenLinks([]listLink{}, folder.Children, c.IncludeSubfolders)}, true, nil
		},
	}
}
