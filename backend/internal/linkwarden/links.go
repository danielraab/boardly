package linkwarden

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"boardly/internal/favourites"
)

// pageSize is Linkwarden's default PAGINATION_TAKE_COUNT.
const pageSize = 50

// Sort orders of links (Linkwarden's Sort).
const (
	SortNewest = 0
	SortOldest = 1
	SortNameAZ = 2
	SortNameZA = 3
)

// SearchInput is the query of GET /api/v1/search. Zero values mean "not set".
type SearchInput struct {
	Sort         int
	Cursor       int64
	CollectionID int64
	TagID        int64
	PinnedOnly   bool
	Query        string
}

// LinkPage is a page of links.
type LinkPage struct {
	Links      []Link `json:"links"`
	NextCursor *int64 `json:"nextCursor"`
}

// sortLinks orders links by one of the sort orders, by integer ID for equal
// values.
func sortLinks(links []*favourites.ScopedNode, order int) {
	slices.SortFunc(links, func(a, b *favourites.ScopedNode) int {
		switch order {
		case SortOldest:
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.Num, b.Num))
		case SortNameAZ:
			return cmp.Or(strings.Compare(strings.ToLower(linkName(a)), strings.ToLower(linkName(b))), cmp.Compare(a.Num, b.Num))
		case SortNameZA:
			return cmp.Or(strings.Compare(strings.ToLower(linkName(b)), strings.ToLower(linkName(a))), cmp.Compare(b.Num, a.Num))
		default:
			return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.Num, a.Num))
		}
	})
}

// page returns the items after the one with the cursor's ID, and the next
// cursor when more items follow. An unknown cursor returns nothing.
func page[T any](items []T, cursor int64, num func(T) int64) ([]T, *int64) {
	if cursor != 0 {
		i := slices.IndexFunc(items, func(item T) bool { return num(item) == cursor })
		if i < 0 {
			return nil, nil
		}
		items = items[i+1:]
	}
	if len(items) <= pageSize {
		return items, nil
	}
	items = items[:pageSize]
	next := num(items[len(items)-1])
	return items, &next
}

// terms splits a search query at whitespace, keeping double-quoted parts
// together and dropping the quotes.
func terms(query string) []string {
	var out []string
	var b strings.Builder
	quoted := false
	for _, r := range query {
		switch {
		case r == '"':
			quoted = !quoted
		case !quoted && (r == ' ' || r == '\t' || r == '\n'):
			if b.Len() > 0 {
				out = append(out, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// matches reports whether a link matches one search term.
func (t *tree) matches(l *favourites.ScopedNode, term string) bool {
	if key, value, ok := strings.Cut(term, ":"); ok && value != "" {
		switch strings.ToLower(key) {
		case "url":
			return favourites.SameURL(*l.URL, value)
		case "name":
			return l.Title != nil && containsFold(*l.Title, value)
		case "tag":
			return slices.ContainsFunc(l.Tags, func(tag favourites.Tag) bool { return strings.EqualFold(tag.Name, value) })
		case "collection":
			return containsFold(folderName(t.byID[l.ParentID]), value)
		}
	}
	if l.Title != nil && containsFold(*l.Title, term) {
		return true
	}
	if containsFold(*l.URL, term) {
		return true
	}
	return slices.ContainsFunc(l.Tags, func(tag favourites.Tag) bool { return containsFold(tag.Name, term) })
}

// Search is GET /api/v1/search.
func (s *Service) Search(ctx context.Context, sc favourites.Scope, in SearchInput) (LinkPage, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return LinkPage{}, err
	}
	empty := LinkPage{Links: []Link{}}
	if in.PinnedOnly {
		return empty, nil
	}
	var collection string
	if in.CollectionID != 0 {
		c, err := t.collection(in.CollectionID)
		if err != nil {
			return empty, nil
		}
		collection = c.ID
	}
	queryTerms := terms(in.Query)
	var found []*favourites.ScopedNode
	for _, l := range t.links() {
		if collection != "" && l.ParentID != collection {
			continue
		}
		if in.TagID != 0 && !slices.ContainsFunc(l.Tags, func(tag favourites.Tag) bool { return tag.Num == in.TagID }) {
			continue
		}
		if !slices.ContainsFunc(queryTerms, func(term string) bool { return !t.matches(l, term) }) {
			found = append(found, l)
		}
	}
	sortLinks(found, in.Sort)
	found, next := page(found, in.Cursor, func(n *favourites.ScopedNode) int64 { return n.Num })
	links, err := s.mapLinks(ctx, t, found)
	if err != nil {
		return LinkPage{}, err
	}
	return LinkPage{Links: links, NextCursor: next}, nil
}

// Link is GET /api/v1/links/{id}.
func (s *Service) Link(ctx context.Context, sc favourites.Scope, num int64) (Link, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Link{}, err
	}
	n, err := t.link(num)
	if err != nil {
		return Link{}, err
	}
	links, err := s.mapLinks(ctx, t, []*favourites.ScopedNode{n})
	if err != nil {
		return Link{}, err
	}
	return links[0], nil
}

// TagRef is a tag on a link in a request: by name, or by ID when the name is
// empty.
type TagRef struct {
	ID   *int64 `json:"id"`
	Name string `json:"name"`
}

// CollectionRef is the collection of a link in a request.
type CollectionRef struct {
	ID   *int64  `json:"id"`
	Name *string `json:"name"`
}

// CreateLinkInput is the body of POST /api/v1/links (Linkwarden's PostLinkSchema).
type CreateLinkInput struct {
	Type        *string        `json:"type"`
	URL         *string        `json:"url"`
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	Image       *string        `json:"image"`
	Collection  *CollectionRef `json:"collection"`
	Tags        []TagRef       `json:"tags"`
}

// tagNames resolves tag references to names; a reference by ID must be a tag
// visible in the subtree.
func (t *tree) tagNames(refs []TagRef) ([]string, error) {
	visible := t.tags()
	names := []string{}
	for _, ref := range refs {
		name := strings.TrimSpace(ref.Name)
		if name == "" && ref.ID != nil {
			tag, ok := visible[*ref.ID]
			if !ok {
				return nil, ErrTagNotFound
			}
			name = tag.Name
		}
		names = append(names, name)
	}
	if _, err := favourites.NormalizeTags(names); err != nil {
		return nil, err
	}
	return names, nil
}

// CreateLink is POST /api/v1/links: a new link first in the chosen collection.
func (s *Service) CreateLink(ctx context.Context, sc favourites.Scope, in CreateLinkInput) (Link, error) {
	if in.Type != nil && *in.Type != "" && *in.Type != "url" {
		return Link{}, &InputError{"Boardly only stores links, not " + *in.Type + " files."}
	}
	if in.URL == nil {
		return Link{}, &InputError{"A link needs a URL."}
	}
	rawURL, err := favourites.NormalizeURL(*in.URL)
	if err != nil {
		return Link{}, err
	}
	title, err := favourites.NormalizeLinkTitle(in.Name)
	if err != nil {
		return Link{}, err
	}
	description, err := favourites.NormalizeDescription(in.Description)
	if err != nil {
		return Link{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return Link{}, err
	}
	tags, err := t.tagNames(in.Tags)
	if err != nil {
		return Link{}, err
	}

	parent := sc.RootID
	var newFolder string
	switch c := in.Collection; {
	case c != nil && c.ID != nil:
		folder, err := t.collection(*c.ID)
		if err != nil {
			return Link{}, errNotAccessible
		}
		parent = folder.ID
	case c != nil && c.Name != nil && strings.TrimSpace(*c.Name) != "" && !strings.EqualFold(strings.TrimSpace(*c.Name), unorganized):
		name, err := favourites.NormalizeFolderName(*c.Name)
		if err != nil {
			return Link{}, err
		}
		i := slices.IndexFunc(t.children[sc.RootID], func(n *favourites.ScopedNode) bool {
			return n.Kind == favourites.KindFolder && strings.EqualFold(folderName(n), name)
		})
		if i >= 0 {
			parent = t.children[sc.RootID][i].ID
		} else {
			newFolder = name
		}
	}
	if newFolder != "" {
		folder, err := s.Favourites.Create(ctx, sc.UserID, favourites.NewNode{ParentID: sc.RootID, Kind: favourites.KindFolder, Title: &newFolder})
		if err != nil {
			return Link{}, err
		}
		parent = folder.ID
	}
	node, err := s.Favourites.Create(ctx, sc.UserID, favourites.NewNode{
		ParentID: parent, Kind: favourites.KindLink, URL: &rawURL, Title: title, Tags: tags, Description: description,
		First: true,
	})
	if err != nil {
		return Link{}, err
	}
	return s.linkByID(ctx, sc, node.ID)
}

// PinRef is an entry of pinnedBy.
type PinRef struct {
	ID *int64 `json:"id"`
}

// UpdateLinkInput is the body of PUT /api/v1/links/{id} (Linkwarden's
// UpdateLinkSchema). Name and Description are raw JSON to tell a missing
// value from null.
type UpdateLinkInput struct {
	Name        json.RawMessage `json:"name"`
	URL         *string         `json:"url"`
	Description json.RawMessage `json:"description"`
	Icon        *string         `json:"icon"`
	IconWeight  *string         `json:"iconWeight"`
	Color       *string         `json:"color"`
	Collection  *CollectionRef  `json:"collection"`
	Tags        *[]TagRef       `json:"tags"`
	PinnedBy    []*PinRef       `json:"pinnedBy"`
}

// UpdateLink is PUT /api/v1/links/{id}: title, URL, description, tags and
// collection.
func (s *Service) UpdateLink(ctx context.Context, sc favourites.Scope, num int64, in UpdateLinkInput) (Link, error) {
	if err := checkUnsupported(map[string]bool{
		"link icons":   nonEmpty(in.Icon) || nonEmpty(in.IconWeight),
		"link colors":  nonEmpty(in.Color),
		"pinned links": slices.ContainsFunc(in.PinnedBy, func(p *PinRef) bool { return p != nil && p.ID != nil }),
	}); err != nil {
		return Link{}, err
	}
	if in.Collection == nil || in.Collection.ID == nil {
		return Link{}, &InputError{"A link needs a collection."}
	}
	if in.Tags == nil {
		return Link{}, &InputError{"A link needs its list of tags."}
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return Link{}, err
	}
	link, err := t.link(num)
	if err != nil {
		return Link{}, err
	}
	target, err := t.collection(*in.Collection.ID)
	if err != nil {
		return Link{}, errNotAccessible
	}
	tags, err := t.tagNames(*in.Tags)
	if err != nil {
		return Link{}, err
	}
	patch := favourites.Patch{Tags: &tags}
	if patch.Title, err = clearable(in.Name, "Invalid name."); err != nil {
		return Link{}, err
	}
	if _, err := favourites.NormalizeLinkTitle(patch.Title); err != nil {
		return Link{}, err
	}
	if patch.Description, err = clearable(in.Description, "Invalid description."); err != nil {
		return Link{}, err
	}
	if _, err := favourites.NormalizeDescription(patch.Description); err != nil {
		return Link{}, err
	}
	if in.URL != nil && *in.URL != "" {
		if _, err := favourites.NormalizeURL(*in.URL); err != nil {
			return Link{}, err
		}
		patch.URL = in.URL
	}
	if target.ID != link.ParentID {
		if err := s.Favourites.CheckMove(ctx, sc, link.ID, target.ID); err != nil {
			return Link{}, err
		}
	}
	if _, err := s.Favourites.Update(ctx, sc.UserID, link.ID, patch); err != nil {
		return Link{}, err
	}
	if target.ID != link.ParentID {
		if err := s.Favourites.Move(ctx, sc.UserID, link.ID, target.ID, 0); err != nil {
			return Link{}, err
		}
	}
	return s.linkByID(ctx, sc, link.ID)
}

// DeleteLink is DELETE /api/v1/links/{id}. It returns the deleted link.
func (s *Service) DeleteLink(ctx context.Context, sc favourites.Scope, num int64) (Link, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Link{}, err
	}
	n, err := t.link(num)
	if err != nil {
		return Link{}, err
	}
	links, err := s.mapLinks(ctx, t, []*favourites.ScopedNode{n})
	if err != nil {
		return Link{}, err
	}
	if err := s.Favourites.Delete(ctx, sc.UserID, n.ID, favourites.DeleteKeepNone); err != nil {
		return Link{}, err
	}
	return links[0], nil
}

// clearable decodes an optional string field: nil when it is missing, and an
// empty string, which clears the value, for null.
func clearable(raw json.RawMessage, invalid string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	var v *string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, &InputError{invalid}
	}
	if v == nil {
		v = new(string)
	}
	return v, nil
}
