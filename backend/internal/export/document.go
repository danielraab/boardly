// Package export builds the download of all data of a user (openspec change
// export-all-data): a JSON document in the Boardly export format, optionally
// packed into a ZIP archive together with the user's image and attachment
// files.
//
// The types in this file define version 1 of the format. They are separate
// from the API types on purpose, so that API changes don't silently change
// the export; a change of their shape needs a new version.
package export

import (
	"encoding/json"

	"boardly/internal/notes"
)

// Format and Version identify the export document.
const (
	Format  = "boardly"
	Version = 1
)

// Document is an export of all data of one user. Times are RFC 3339 in UTC.
type Document struct {
	Format         string        `json:"format"`
	Version        int           `json:"version"`
	ExportedAt     string        `json:"exportedAt"`
	WithFiles      bool          `json:"withFiles"`
	Profile        Profile       `json:"profile"`
	Favourites     *Node         `json:"favourites"`
	Tags           []Tag         `json:"tags"`
	Dashboards     []Dashboard   `json:"dashboards"`
	NoteCategories []Category    `json:"noteCategories"`
	Notes          []Note        `json:"notes"`
	MediaSources   []MediaSource `json:"mediaSources"`
	Images         []Image       `json:"images"`
	Attachments    []Attachment  `json:"attachments"`
}

// Profile holds the account settings of the user.
type Profile struct {
	Email                 string  `json:"email"`
	Name                  *string `json:"name"`
	Slug                  *string `json:"slug"`
	CreatedAt             string  `json:"createdAt"`
	DefaultFolderID       *string `json:"defaultFolderId"`
	DefaultNoteCategoryID *string `json:"defaultNoteCategoryId"`
}

// Node is a folder or a link of the favourites tree.
type Node struct {
	ID        string
	Kind      string // "folder" or "link"
	Title     *string
	CreatedAt string
	UpdatedAt string

	// Links only.
	URL            string
	Description    *string
	Tags           []string
	PreviewImageID *string
	// OfflineCopy is nil when the link has no offline copy with a part.
	OfflineCopy *OfflineCopy

	// Folders only, in their order.
	Children []*Node
}

// MarshalJSON writes a folder with its children and a link with its link fields.
func (n *Node) MarshalJSON() ([]byte, error) {
	if n.Kind == "folder" {
		children := n.Children
		if children == nil {
			children = []*Node{}
		}
		return json.Marshal(struct {
			ID        string  `json:"id"`
			Kind      string  `json:"kind"`
			Title     *string `json:"title"`
			CreatedAt string  `json:"createdAt"`
			UpdatedAt string  `json:"updatedAt"`
			Children  []*Node `json:"children"`
		}{n.ID, n.Kind, n.Title, n.CreatedAt, n.UpdatedAt, children})
	}
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(struct {
		ID             string       `json:"id"`
		Kind           string       `json:"kind"`
		URL            string       `json:"url"`
		Title          *string      `json:"title"`
		Description    *string      `json:"description"`
		Tags           []string     `json:"tags"`
		PreviewImageID *string      `json:"previewImageId"`
		OfflineCopy    *OfflineCopy `json:"offlineCopy"`
		CreatedAt      string       `json:"createdAt"`
		UpdatedAt      string       `json:"updatedAt"`
	}{n.ID, n.Kind, n.URL, n.Title, n.Description, tags, n.PreviewImageID, n.OfflineCopy, n.CreatedAt, n.UpdatedAt})
}

// UnmarshalJSON reads a folder or a link as MarshalJSON writes it.
func (n *Node) UnmarshalJSON(data []byte) error {
	var v struct {
		ID             string       `json:"id"`
		Kind           string       `json:"kind"`
		Title          *string      `json:"title"`
		CreatedAt      string       `json:"createdAt"`
		UpdatedAt      string       `json:"updatedAt"`
		URL            string       `json:"url"`
		Description    *string      `json:"description"`
		Tags           []string     `json:"tags"`
		PreviewImageID *string      `json:"previewImageId"`
		OfflineCopy    *OfflineCopy `json:"offlineCopy"`
		Children       []*Node      `json:"children"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = Node{
		ID: v.ID, Kind: v.Kind, Title: v.Title, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
		URL: v.URL, Description: v.Description, Tags: v.Tags, PreviewImageID: v.PreviewImageID,
		OfflineCopy: v.OfflineCopy, Children: v.Children,
	}
	return nil
}

// OfflineCopy is the offline copy of a link: when it was captured, the
// page title, and the paths of its article and screenshot in the archive,
// nil without files or when the copy doesn't have that part. Its state
// (pending, failed) isn't exported.
type OfflineCopy struct {
	CapturedAt string  `json:"capturedAt"`
	Title      *string `json:"title"`
	Article    *string `json:"article"`
	Screenshot *string `json:"screenshot"`

	// The stored files, for the archive.
	articleID, screenshotID string
}

// Tag is a tag of the user, used or not.
type Tag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// Dashboard is a dashboard with its widgets.
type Dashboard struct {
	ID               string    `json:"id"`
	Slug             string    `json:"slug"`
	Title            string    `json:"title"`
	IsDefault        bool      `json:"isDefault"`
	IsPublic         bool      `json:"isPublic"`
	DesktopColumns   int       `json:"desktopColumns"`
	MobileColumns    int       `json:"mobileColumns"`
	MobileBreakpoint int       `json:"mobileBreakpoint"`
	CreatedAt        string    `json:"createdAt"`
	UpdatedAt        string    `json:"updatedAt"`
	Widgets          []*Widget `json:"widgets"`
}

// Widget is a section or a content widget with its configuration as stored.
type Widget struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	Width    int             `json:"width"`
	Config   json.RawMessage `json:"config"`
	Children []*Widget       `json:"children,omitempty"`
}

// Category is a note category.
type Category struct {
	ID        string      `json:"id"`
	ParentID  *string     `json:"parentId"`
	Title     string      `json:"title"`
	Icon      *notes.Icon `json:"icon"`
	Color     *string     `json:"color"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}

// Note is a note with its content document as stored.
type Note struct {
	ID            string          `json:"id"`
	CategoryID    *string         `json:"categoryId"`
	Title         *string         `json:"title"`
	Icon          *notes.Icon     `json:"icon"`
	Color         *string         `json:"color"`
	Content       json.RawMessage `json:"content"`
	Pinned        bool            `json:"pinned"`
	Archived      bool            `json:"archived"`
	AttachmentIDs []string        `json:"attachmentIds"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
}

// MediaSource is a media source, such as a radio station.
type MediaSource struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"`
	Title     string      `json:"title"`
	URL       string      `json:"url"`
	Icon      *notes.Icon `json:"icon"`
	Position  int         `json:"position"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}

// Image is an image the user uploaded or a link preview fetched for them.
// File is its path in the export archive, or nil.
type Image struct {
	ID          string  `json:"id"`
	Source      string  `json:"source"`
	ContentType string  `json:"contentType"`
	Size        int64   `json:"size"`
	CreatedAt   string  `json:"createdAt"`
	File        *string `json:"file"`
}

// Attachment is a note attachment that isn't deleted. File is its path in the
// export archive, or nil.
type Attachment struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ContentType string  `json:"contentType"`
	Size        int64   `json:"size"`
	CreatedAt   string  `json:"createdAt"`
	File        *string `json:"file"`
}
