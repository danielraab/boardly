// Package widgets implements the content widget kinds of dashboards.
package widgets

import (
	"bytes"
	"encoding/json"
	"fmt"

	"famodash/internal/dashboards"
	"famodash/internal/favourites"
	"famodash/internal/images"
)

// Register adds every content widget kind to the dashboard store. icons may
// be nil, and then links have no favicons.
func Register(d *dashboards.Store, fav *favourites.Store, icons *images.Favicons) {
	d.Kinds[KindFavouriteExplorer] = FavouriteExplorer(fav, icons)
	d.Kinds[KindLink] = Link(fav)
}

// maxIDLength bounds the IDs of favourites and images in configs.
const maxIDLength = 64

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
