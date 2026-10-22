// Package karakeep maps a token's subtree of the favourites tree to what the
// Karakeep browser extension and mobile app expect: links are bookmarks,
// folders below the token's root are lists and the tags on links are tags.
// The shapes follow Karakeep's packages/shared/types (commit 85054d9).
package karakeep

import (
	"context"
	"errors"
	"slices"
	"time"

	"boardly/internal/favourites"
	"boardly/internal/images"
	"boardly/internal/trpc"
)

// Errors returned by the service, besides favourites' validation errors and
// favourites.ErrNotFound for anything outside the scope.
var (
	// ErrTagShared is returned when a tag can't be renamed because links
	// outside the scope have it too.
	ErrTagShared = errors.New("the tag is also used outside this connected app's folder")
	// ErrTagCreate is returned by CreateTag: tags only exist on links.
	ErrTagCreate = errors.New("tags are created by adding them to a bookmark")
)

// InputError is an input that Boardly can't handle, such as a field it
// doesn't support. Its message is shown to the user by the clients.
type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }

func unsupported(field string) error {
	return &InputError{"Boardly doesn't support " + field}
}

// Favicons looks up and fetches favicons by origin (images.Favicons).
type Favicons interface {
	Lookup(ctx context.Context, origins []string) (map[string]string, error)
	Queue(rawURL string)
}

// Service implements the Karakeep procedures on the favourites tree.
type Service struct {
	Favourites *favourites.Store
	Favicons   Favicons
	// Origin is Boardly's origin, for favicon URLs.
	Origin string
	Now    func() time.Time
}

// Tag is a tag on a bookmark.
type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AttachedBy string `json:"attachedBy"`
}

// LinkContent is the content of a link bookmark.
type LinkContent struct {
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Title       *string   `json:"title"`
	Favicon     *string   `json:"favicon"`
	CrawlStatus string    `json:"crawlStatus"`
	CrawledAt   trpc.Date `json:"crawledAt"`
}

// Bookmark is Karakeep's bookmark shape for a link.
type Bookmark struct {
	ID                  string      `json:"id"`
	CreatedAt           trpc.Date   `json:"createdAt"`
	ModifiedAt          trpc.Date   `json:"modifiedAt"`
	Title               *string     `json:"title"`
	Archived            bool        `json:"archived"`
	Favourited          bool        `json:"favourited"`
	TaggingStatus       *string     `json:"taggingStatus"`
	SummarizationStatus *string     `json:"summarizationStatus"`
	EmbeddingStatus     *string     `json:"embeddingStatus"`
	Note                *string     `json:"note"`
	Summary             *string     `json:"summary"`
	Source              *string     `json:"source"`
	UserID              string      `json:"userId"`
	Tags                []Tag       `json:"tags"`
	Content             LinkContent `json:"content"`
	Assets              []struct{}  `json:"assets"`
}

// tree is a loaded subtree.
type tree struct {
	scope    favourites.Scope
	nodes    []*favourites.ScopedNode
	byID     map[string]*favourites.ScopedNode
	children map[string][]*favourites.ScopedNode // in position order
}

func (s *Service) load(ctx context.Context, sc favourites.Scope) (*tree, error) {
	nodes, err := s.Favourites.SubtreeNodes(ctx, sc)
	if err != nil {
		return nil, err
	}
	t := &tree{scope: sc, byID: map[string]*favourites.ScopedNode{}, children: map[string][]*favourites.ScopedNode{}}
	for i := range nodes {
		n := &nodes[i]
		t.nodes = append(t.nodes, n)
		t.byID[n.ID] = n
		if n.ID != sc.RootID {
			t.children[n.ParentID] = append(t.children[n.ParentID], n)
		}
	}
	for _, c := range t.children {
		slices.SortFunc(c, func(a, b *favourites.ScopedNode) int { return a.Position - b.Position })
	}
	return t, nil
}

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

// link returns the link with the given ID, or favourites.ErrNotFound.
func (t *tree) link(id string) (*favourites.ScopedNode, error) {
	n, ok := t.byID[id]
	if !ok || n.Kind != favourites.KindLink {
		return nil, favourites.ErrNotFound
	}
	return n, nil
}

// list returns the folder below the root with the given ID, or
// favourites.ErrNotFound.
func (t *tree) list(id string) (*favourites.ScopedNode, error) {
	n, ok := t.byID[id]
	if !ok || n.Kind != favourites.KindFolder || id == t.scope.RootID {
		return nil, favourites.ErrNotFound
	}
	return n, nil
}

// bookmarks maps links to bookmarks, with their favicons.
func (s *Service) bookmarks(ctx context.Context, userID string, links []*favourites.ScopedNode) ([]Bookmark, error) {
	var origins []string
	for _, n := range links {
		if o, err := images.Origin(*n.URL); err == nil {
			origins = append(origins, o)
		}
	}
	icons, err := s.Favicons.Lookup(ctx, origins)
	if err != nil {
		return nil, err
	}
	out := make([]Bookmark, len(links))
	for i, n := range links {
		var favicon *string
		if o, err := images.Origin(*n.URL); err == nil {
			if id, ok := icons[o]; ok {
				u := s.Origin + "/api/images/" + id
				favicon = &u
			}
		}
		tags := make([]Tag, len(n.Tags))
		for j, t := range n.Tags {
			tags[j] = Tag{ID: t.ID, Name: t.Name, AttachedBy: "human"}
		}
		out[i] = Bookmark{
			ID:         n.ID,
			CreatedAt:  trpc.Date(n.CreatedAt),
			ModifiedAt: trpc.Date(n.UpdatedAt),
			Title:      n.Title,
			UserID:     userID,
			Tags:       tags,
			Content: LinkContent{
				Type: "link", URL: *n.URL, Title: n.Title, Favicon: favicon,
				CrawlStatus: "success", CrawledAt: trpc.Date(n.CreatedAt),
			},
			Assets: []struct{}{},
		}
	}
	return out, nil
}

// bookmark loads the subtree and returns the bookmark with the given ID.
func (s *Service) bookmark(ctx context.Context, sc favourites.Scope, id string) (Bookmark, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Bookmark{}, err
	}
	n, err := t.link(id)
	if err != nil {
		return Bookmark{}, err
	}
	b, err := s.bookmarks(ctx, sc.UserID, []*favourites.ScopedNode{n})
	if err != nil {
		return Bookmark{}, err
	}
	return b[0], nil
}
