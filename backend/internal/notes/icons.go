package notes

import (
	"context"
	"database/sql"
	"errors"

	"boardly/internal/db"
	"boardly/internal/symbols"
)

// Icon types.
const (
	IconSymbol = "symbol"
	IconImage  = "image"
)

// Icon is the icon of a note or a category: a symbol or an uploaded image.
type Icon struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	ImageID string `json:"imageId,omitempty"`
}

// noteColours are the colours a note or category can have. Keep in sync with
// frontend/src/features/notes/colours.ts.
var noteColours = map[string]bool{
	"red": true, "orange": true, "yellow": true, "green": true, "teal": true,
	"blue": true, "purple": true, "pink": true, "grey": true,
}

// checkIcon validates an icon and returns its columns. An image must be an
// upload of the user.
func checkIcon(ctx context.Context, q db.Querier, userID string, icon *Icon) (symbol, imageID *string, err error) {
	if icon == nil {
		return nil, nil, nil
	}
	switch icon.Type {
	case IconSymbol:
		if icon.ImageID != "" || !symbols.Valid(icon.Name) {
			return nil, nil, ErrInvalidIcon
		}
		return &icon.Name, nil, nil
	case IconImage:
		if icon.Name != "" || icon.ImageID == "" {
			return nil, nil, ErrInvalidIcon
		}
		var one int
		err := q.QueryRowContext(ctx,
			`SELECT 1 FROM images WHERE id = ? AND source = 'upload' AND user_id = ?`,
			icon.ImageID, userID,
		).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrInvalidIcon
		}
		if err != nil {
			return nil, nil, err
		}
		return nil, &icon.ImageID, nil
	default:
		return nil, nil, ErrInvalidIcon
	}
}

// iconOf returns the icon stored in the columns, or nil.
func iconOf(symbol, imageID *string) *Icon {
	switch {
	case symbol != nil:
		return &Icon{Type: IconSymbol, Name: *symbol}
	case imageID != nil:
		return &Icon{Type: IconImage, ImageID: *imageID}
	}
	return nil
}

// checkColor returns ErrInvalidColor for a colour outside the palette.
func checkColor(color *string) error {
	if color != nil && !noteColours[*color] {
		return ErrInvalidColor
	}
	return nil
}
