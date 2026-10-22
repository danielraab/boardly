package karakeep

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"famodash/internal/favourites"
)

// TagInfo is Karakeep's tag shape with its bookmark counts in the subtree.
type TagInfo struct {
	ID                         string         `json:"id"`
	Name                       string         `json:"name"`
	NumBookmarks               int            `json:"numBookmarks"`
	NumBookmarksByAttachedType map[string]int `json:"numBookmarksByAttachedType"`
}

// TagsInput is the input of tags.list.
type TagsInput struct {
	NameContains string   `json:"nameContains"`
	IDs          []string `json:"ids"`
	AttachedBy   string   `json:"attachedBy"`
	SortBy       string   `json:"sortBy"`
	Cursor       *struct {
		Page int `json:"page"`
	} `json:"cursor"`
	Limit int `json:"limit"`
}

// TagCursor is the page of the next tags.list call.
type TagCursor struct {
	Page int `json:"page"`
}

// TagPage is a page of tags.list.
type TagPage struct {
	Tags       []TagInfo  `json:"tags"`
	NextCursor *TagCursor `json:"nextCursor"`
}

// tagInfos counts the bookmarks of every tag on the subtree's links.
func (t *tree) tagInfos() map[string]*TagInfo {
	out := map[string]*TagInfo{}
	for _, l := range t.links() {
		for _, tag := range l.Tags {
			info, ok := out[tag.ID]
			if !ok {
				info = &TagInfo{ID: tag.ID, Name: tag.Name}
				out[tag.ID] = info
			}
			info.NumBookmarks++
		}
	}
	for _, info := range out {
		info.NumBookmarksByAttachedType = map[string]int{"human": info.NumBookmarks, "ai": 0}
	}
	return out
}

// Tags is tags.list.
func (s *Service) Tags(ctx context.Context, sc favourites.Scope, in TagsInput) (TagPage, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return TagPage{}, err
	}
	tags := []TagInfo{}
	// Every tag is attached by a human, and none is on no bookmark.
	if in.AttachedBy == "" || in.AttachedBy == "human" {
		contains := strings.ToLower(in.NameContains)
		for _, info := range t.tagInfos() {
			if !strings.Contains(strings.ToLower(info.Name), contains) {
				continue
			}
			if in.IDs != nil && !slices.Contains(in.IDs, info.ID) {
				continue
			}
			tags = append(tags, *info)
		}
	}
	byName := func(a, b TagInfo) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.ID, b.ID))
	}
	if in.SortBy == "name" {
		slices.SortFunc(tags, byName)
	} else {
		slices.SortFunc(tags, func(a, b TagInfo) int {
			return cmp.Or(cmp.Compare(b.NumBookmarks, a.NumBookmarks), byName(a, b))
		})
	}

	page := TagPage{Tags: tags}
	if in.Limit > 0 {
		p := 0
		if in.Cursor != nil {
			p = max(in.Cursor.Page, 0)
		}
		start := min(p*in.Limit, len(tags))
		end := min(start+in.Limit, len(tags))
		page.Tags = tags[start:end]
		if end < len(tags) {
			page.NextCursor = &TagCursor{Page: p + 1}
		}
	}
	return page, nil
}

// Tag is tags.get.
func (s *Service) Tag(ctx context.Context, sc favourites.Scope, id string) (TagInfo, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return TagInfo{}, err
	}
	info, ok := t.tagInfos()[id]
	if !ok {
		return TagInfo{}, favourites.ErrNotFound
	}
	return *info, nil
}

// RenameTag is tags.update. A tag that links outside the subtree have too
// can't be renamed, since the rename would change them as well.
func (s *Service) RenameTag(ctx context.Context, sc favourites.Scope, id string, name *string) (favourites.Tag, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return favourites.Tag{}, err
	}
	info, ok := t.tagInfos()[id]
	if !ok {
		return favourites.Tag{}, favourites.ErrNotFound
	}
	if name == nil {
		return favourites.Tag{ID: info.ID, Name: info.Name}, nil
	}
	total, err := s.Favourites.TagLinkCount(ctx, sc.UserID, id)
	if err != nil {
		return favourites.Tag{}, err
	}
	if total != info.NumBookmarks {
		return favourites.Tag{}, ErrTagShared
	}
	return s.Favourites.RenameTag(ctx, sc.UserID, id, *name)
}

// DeleteTag is tags.delete: the tag is removed from the subtree's links only.
func (s *Service) DeleteTag(ctx context.Context, sc favourites.Scope, id string) error {
	t, err := s.load(ctx, sc)
	if err != nil {
		return err
	}
	var links []string
	for _, l := range t.links() {
		if slices.ContainsFunc(l.Tags, func(tag favourites.Tag) bool { return tag.ID == id }) {
			links = append(links, l.ID)
		}
	}
	if len(links) == 0 {
		return favourites.ErrNotFound
	}
	return s.Favourites.DetachTag(ctx, sc.UserID, id, links)
}

// CreateTag is tags.create, which Famodash refuses: a tag only exists while a
// link has it.
func (s *Service) CreateTag() error {
	return ErrTagCreate
}
