// Package linkwarden maps a token's subtree of the favourites tree to what the
// Linkwarden mobile app and browser extension expect: links are links,
// folders (including the token's root folder) are collections and the tags on
// links are tags. IDs are the integer IDs of nodes and tags, and the token's
// owner is always the user 1. The shapes follow Linkwarden v2.16.3
// (linkwarden/linkwarden commit 952ac45: packages/prisma/schema.prisma,
// packages/types, packages/lib/schemaValidation.ts).
package linkwarden

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"boardly/internal/favourites"
	"boardly/internal/images"
)

// Errors returned by the service, besides favourites' validation errors and
// InputError.
var (
	ErrLinkNotFound       = errors.New("Link not found.")
	ErrCollectionNotFound = errors.New("Collection not found.")
	ErrTagNotFound        = errors.New("Tag not found.")
	ErrPreviewNotFound    = errors.New("Preview not found.")
	// ErrRootCollection is returned when the token's root collection would be
	// deleted.
	ErrRootCollection = errors.New("The folder of this connected app can't be deleted.")
	// ErrTagCreate is returned by CreateTag: tags only exist on links.
	ErrTagCreate = errors.New("Boardly creates tags when you add them to a link.")
)

// InputError is an input that Boardly can't handle, such as a field it
// doesn't support. Its message is shown to the user by the clients.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

func unsupported(field string) error {
	return &InputError{"Boardly doesn't support " + field + "."}
}

// checkUnsupported returns an error for the first set field, in name order.
func checkUnsupported(fields map[string]bool) error {
	var set []string
	for name, isSet := range fields {
		if isSet {
			set = append(set, name)
		}
	}
	if len(set) == 0 {
		return nil
	}
	slices.Sort(set)
	return unsupported(set[0])
}

func nonEmpty(s *string) bool { return s != nil && *s != "" }

// errNotAccessible is returned for a collection ID that isn't in the scope.
var errNotAccessible = &InputError{"Collection is not accessible."}

// userID is the ID of the token's owner in every response.
const userID = 1

// unorganized is the name of the user's top-level folder, which has no title.
const unorganized = "Unorganized"

// Favicons looks up and fetches favicons by origin (images.Favicons).
type Favicons interface {
	LookupImages(ctx context.Context, origins []string) (map[string]images.FaviconImage, error)
}

// previewTypes are the favicon types the mobile app can show as previews.
var previewTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

// Previews takes preview images that clients send (images.Previews).
type Previews interface {
	Replace(ctx context.Context, userID, nodeID, rawURL string, data []byte) error
}

// Service implements the Linkwarden endpoints on the favourites tree.
type Service struct {
	Favourites *favourites.Store
	Favicons   Favicons
	// Previews, when set, stores uploaded screenshots as link previews.
	Previews Previews
}

// Time is a time in JavaScript's toISOString format, as Linkwarden sends it.
type Time time.Time

// MarshalJSON writes the time in UTC with milliseconds.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z"))), nil
}

// Count is Prisma's _count of links.
type Count struct {
	Links int `json:"links"`
}

// Tag is Linkwarden's tag. Count is set in tag lists and left out on links.
type Tag struct {
	ID                      int64  `json:"id"`
	Name                    string `json:"name"`
	OwnerID                 int    `json:"ownerId"`
	ArchiveAsScreenshot     *bool  `json:"archiveAsScreenshot"`
	ArchiveAsMonolith       *bool  `json:"archiveAsMonolith"`
	ArchiveAsPDF            *bool  `json:"archiveAsPDF"`
	ArchiveAsReadable       *bool  `json:"archiveAsReadable"`
	ArchiveAsWaybackMachine *bool  `json:"archiveAsWaybackMachine"`
	AiTag                   *bool  `json:"aiTag"`
	AiGenerated             bool   `json:"aiGenerated"`
	CreatedAt               Time   `json:"createdAt"`
	UpdatedAt               Time   `json:"updatedAt"`
	Count                   *Count `json:"_count,omitempty"`
}

// ShortCollection is a collection as it is embedded in a link.
type ShortCollection struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        *string `json:"icon"`
	IconWeight  *string `json:"iconWeight"`
	Color       string  `json:"color"`
	ParentID    *int64  `json:"parentId"`
	IsPublic    bool    `json:"isPublic"`
	OwnerID     int     `json:"ownerId"`
	CreatedByID int     `json:"createdById"`
	CreatedAt   Time    `json:"createdAt"`
	UpdatedAt   Time    `json:"updatedAt"`
}

// Parent is the parent of a collection.
type Parent struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Collection is Linkwarden's collection with its parent, members and count.
type Collection struct {
	ShortCollection
	Parent  *Parent    `json:"parent"`
	Members []struct{} `json:"members"`
	Count   Count      `json:"_count"`
}

// Link is Linkwarden's link with its tags and collection.
type Link struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Description     string          `json:"description"`
	CreatedByID     int             `json:"createdById"`
	CollectionID    int64           `json:"collectionId"`
	Icon            *string         `json:"icon"`
	IconWeight      *string         `json:"iconWeight"`
	Color           *string         `json:"color"`
	URL             string          `json:"url"`
	TextContent     *string         `json:"textContent"`
	Preview         string          `json:"preview"`
	Image           string          `json:"image"`
	PDF             string          `json:"pdf"`
	Readable        string          `json:"readable"`
	Monolith        string          `json:"monolith"`
	ClientSide      bool            `json:"clientSide"`
	AiTagged        bool            `json:"aiTagged"`
	MetaDescription *string         `json:"metaDescription"`
	IndexVersion    *int            `json:"indexVersion"`
	LastPreserved   Time            `json:"lastPreserved"`
	ImportDate      *Time           `json:"importDate"`
	CreatedAt       Time            `json:"createdAt"`
	UpdatedAt       Time            `json:"updatedAt"`
	Tags            []Tag           `json:"tags"`
	Collection      ShortCollection `json:"collection"`
	PinnedBy        []struct{}      `json:"pinnedBy"`
}

// unavailable marks a preserved format that doesn't exist, as Linkwarden does
// when preservation is turned off.
const unavailable = "unavailable"

// tree is a loaded subtree.
type tree struct {
	scope    favourites.Scope
	nodes    []*favourites.ScopedNode
	byID     map[string]*favourites.ScopedNode
	byNum    map[int64]*favourites.ScopedNode
	children map[string][]*favourites.ScopedNode // in position order
}

func (s *Service) load(ctx context.Context, sc favourites.Scope) (*tree, error) {
	nodes, err := s.Favourites.SubtreeNodes(ctx, sc)
	if err != nil {
		return nil, err
	}
	t := &tree{
		scope: sc, byID: map[string]*favourites.ScopedNode{}, byNum: map[int64]*favourites.ScopedNode{},
		children: map[string][]*favourites.ScopedNode{},
	}
	for i := range nodes {
		n := &nodes[i]
		t.nodes = append(t.nodes, n)
		t.byID[n.ID] = n
		t.byNum[n.Num] = n
		if n.ID != sc.RootID {
			t.children[n.ParentID] = append(t.children[n.ParentID], n)
		}
	}
	for _, c := range t.children {
		slices.SortFunc(c, func(a, b *favourites.ScopedNode) int { return a.Position - b.Position })
	}
	return t, nil
}

// root returns the token's root folder. A token whose root folder no longer
// exists can't authenticate, so it is always there.
func (t *tree) root() *favourites.ScopedNode { return t.byID[t.scope.RootID] }

// links returns every link of the subtree.
func (t *tree) links() []*favourites.ScopedNode {
	var out []*favourites.ScopedNode
	for _, n := range t.nodes {
		if n.Kind == favourites.KindLink {
			out = append(out, n)
		}
	}
	return out
}

// link returns the link with the given integer ID, or ErrLinkNotFound.
func (t *tree) link(num int64) (*favourites.ScopedNode, error) {
	n, ok := t.byNum[num]
	if !ok || n.Kind != favourites.KindLink {
		return nil, ErrLinkNotFound
	}
	return n, nil
}

// collection returns the folder with the given integer ID, including the
// token's root folder, or ErrCollectionNotFound.
func (t *tree) collection(num int64) (*favourites.ScopedNode, error) {
	n, ok := t.byNum[num]
	if !ok || n.Kind != favourites.KindFolder {
		return nil, ErrCollectionNotFound
	}
	return n, nil
}

// folders returns every folder of the subtree, parents before their children
// and siblings in position order.
func (t *tree) folders() []*favourites.ScopedNode {
	var out []*favourites.ScopedNode
	var walk func(n *favourites.ScopedNode)
	walk = func(n *favourites.ScopedNode) {
		out = append(out, n)
		for _, c := range t.children[n.ID] {
			if c.Kind == favourites.KindFolder {
				walk(c)
			}
		}
	}
	if r := t.root(); r != nil {
		walk(r)
	}
	return out
}

// folderName is a folder's title, or "Unorganized" for the top-level folder.
func folderName(n *favourites.ScopedNode) string {
	if n.Title == nil {
		return unorganized
	}
	return *n.Title
}

// linkName is a link's title, or its URL when it has none, for sorting.
func linkName(n *favourites.ScopedNode) string {
	if n.Title != nil {
		return *n.Title
	}
	return *n.URL
}

func (t *tree) shortCollection(n *favourites.ScopedNode) ShortCollection {
	c := ShortCollection{
		ID: n.Num, Name: folderName(n), Color: "#0ea5e9", OwnerID: userID, CreatedByID: userID,
		CreatedAt: Time(n.CreatedAt), UpdatedAt: Time(n.UpdatedAt),
	}
	if p, ok := t.byID[n.ParentID]; ok && n.ID != t.scope.RootID {
		c.ParentID = &p.Num
	}
	return c
}

func (t *tree) fullCollection(n *favourites.ScopedNode) Collection {
	c := Collection{ShortCollection: t.shortCollection(n), Members: []struct{}{}}
	if p, ok := t.byID[n.ParentID]; ok && n.ID != t.scope.RootID {
		c.Parent = &Parent{ID: p.Num, Name: folderName(p)}
	}
	for _, child := range t.children[n.ID] {
		if child.Kind == favourites.KindLink {
			c.Count.Links++
		}
	}
	return c
}

func tagOf(tag favourites.Tag) Tag {
	return Tag{
		ID: tag.Num, Name: tag.Name, OwnerID: userID,
		CreatedAt: Time(tag.CreatedAt), UpdatedAt: Time(tag.CreatedAt),
	}
}

// mapLinks maps links of the tree, with their preview images, or else their
// favicons, as previews.
func (s *Service) mapLinks(ctx context.Context, t *tree, links []*favourites.ScopedNode) ([]Link, error) {
	var origins []string
	for _, n := range links {
		if o, err := images.Origin(*n.URL); err == nil {
			origins = append(origins, o)
		}
	}
	icons, err := s.Favicons.LookupImages(ctx, origins)
	if err != nil {
		return nil, err
	}
	out := make([]Link, len(links))
	for i, n := range links {
		parent := t.byID[n.ParentID]
		preview := unavailable
		hasPreview := n.PreviewImageID != nil
		if o, err := images.Origin(*n.URL); err == nil && !hasPreview {
			img, ok := icons[o]
			hasPreview = ok && slices.Contains(previewTypes, img.ContentType)
		}
		if hasPreview {
			preview = "archives/preview/" + strconv.FormatInt(parent.Num, 10) + "/" + strconv.FormatInt(n.Num, 10) + ".jpeg"
		}
		tags := make([]Tag, len(n.Tags))
		for j, tag := range n.Tags {
			tags[j] = tagOf(tag)
		}
		name, description := "", ""
		if n.Title != nil {
			name = *n.Title
		}
		if n.Description != nil {
			description = *n.Description
		}
		out[i] = Link{
			ID: n.Num, Name: name, Type: "url", Description: description, CreatedByID: userID, CollectionID: parent.Num, URL: *n.URL,
			Preview: preview, Image: unavailable, PDF: unavailable, Readable: unavailable, Monolith: unavailable,
			LastPreserved: Time(n.CreatedAt), CreatedAt: Time(n.CreatedAt), UpdatedAt: Time(n.UpdatedAt),
			Tags: tags, Collection: t.shortCollection(parent), PinnedBy: []struct{}{},
		}
	}
	return out, nil
}

// linkByID loads the subtree and returns the link with the given node ID.
func (s *Service) linkByID(ctx context.Context, sc favourites.Scope, id string) (Link, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Link{}, err
	}
	n, ok := t.byID[id]
	if !ok || n.Kind != favourites.KindLink {
		return Link{}, ErrLinkNotFound
	}
	links, err := s.mapLinks(ctx, t, []*favourites.ScopedNode{n})
	if err != nil {
		return Link{}, err
	}
	return links[0], nil
}
