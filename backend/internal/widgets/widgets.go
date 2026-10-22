// Package widgets implements the content widget kinds of dashboards.
package widgets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"boardly/internal/dashboards"
	"boardly/internal/favourites"
	"boardly/internal/images"
)

// Register adds every content widget kind to the dashboard store. icons may
// be nil, and then links have no favicons.
func Register(d *dashboards.Store, fav *favourites.Store, icons *images.Favicons) {
	d.Kinds[KindFavouriteExplorer] = FavouriteExplorer(fav, icons)
	d.Kinds[KindLink] = Link(fav, icons)
	d.Kinds[KindLinkList] = LinkList(fav, icons)
}

// Limits shared by the kinds' configs.
const (
	// maxIDLength bounds the IDs of favourites and images.
	maxIDLength = 64
	// maxTitle bounds a widget's own title, in characters.
	maxTitle = 200
)

// normalizeTitle trims an optional title; an empty one is no title.
func normalizeTitle(title *string) (*string, error) {
	if title == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*title)
	if utf8.RuneCountInString(t) > maxTitle {
		return nil, fmt.Errorf("%w: title too long", dashboards.ErrInvalidLayout)
	}
	if t == "" {
		return nil, nil
	}
	return &t, nil
}

// Size is the S/M/L size of a widget's content.
type Size string

func (s Size) valid() bool {
	return s == "S" || s == "M" || s == "L"
}

// decodeConfig decodes a widget config strictly into c.
func decodeConfig(raw json.RawMessage, c any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(c); err != nil {
		return fmt.Errorf("%w: %v", dashboards.ErrInvalidLayout, err)
	}
	return nil
}
