package linkwarden

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"boardly/internal/favourites"
)

// Sort orders of tags (Linkwarden's TagSort).
const (
	TagSortNewest    = 0
	TagSortOldest    = 1
	TagSortNameAZ    = 2
	TagSortNameZA    = 3
	TagSortMostLinks = 4
	TagSortFewLinks  = 5
)

// tagInfo is a tag visible in the subtree with its number of links there.
type tagInfo struct {
	favourites.Tag
	links []string // node IDs of the links in the subtree that have it
}

func (i *tagInfo) tag() Tag {
	t := tagOf(i.Tag)
	t.Count = &Count{Links: len(i.links)}
	return t
}

// tags returns the tags on links of the subtree by integer ID.
func (t *tree) tags() map[int64]*tagInfo {
	out := map[int64]*tagInfo{}
	for _, l := range t.links() {
		for _, tag := range l.Tags {
			info, ok := out[tag.Num]
			if !ok {
				info = &tagInfo{Tag: tag}
				out[tag.Num] = info
			}
			info.links = append(info.links, l.ID)
		}
	}
	return out
}

// TagsInput is the query of GET /api/v1/tags. Zero values mean "not set".
type TagsInput struct {
	Sort   int
	Cursor int64
	Search string
}

// TagPage is a page of tags.
type TagPage struct {
	Tags       []Tag  `json:"tags"`
	NextCursor *int64 `json:"nextCursor"`
}

// Tags is GET /api/v1/tags.
func (s *Service) Tags(ctx context.Context, sc favourites.Scope, in TagsInput) (TagPage, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return TagPage{}, err
	}
	search := strings.TrimSpace(in.Search)
	var infos []*tagInfo
	for _, info := range t.tags() {
		if search == "" || containsFold(info.Name, search) {
			infos = append(infos, info)
		}
	}
	slices.SortFunc(infos, func(a, b *tagInfo) int {
		byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		byCount := cmp.Compare(len(a.links), len(b.links))
		switch in.Sort {
		case TagSortOldest:
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.Num, b.Num))
		case TagSortNameAZ:
			return cmp.Or(byName, cmp.Compare(a.Num, b.Num))
		case TagSortNameZA:
			return cmp.Or(-byName, cmp.Compare(b.Num, a.Num))
		case TagSortMostLinks:
			return cmp.Or(-byCount, cmp.Compare(b.Num, a.Num))
		case TagSortFewLinks:
			return cmp.Or(byCount, cmp.Compare(a.Num, b.Num))
		default:
			return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), cmp.Compare(b.Num, a.Num))
		}
	})
	infos, next := page(infos, in.Cursor, func(i *tagInfo) int64 { return i.Num })
	out := TagPage{Tags: make([]Tag, len(infos)), NextCursor: next}
	for i, info := range infos {
		out.Tags[i] = info.tag()
	}
	return out, nil
}

// Tag is GET /api/v1/tags/{id}.
func (s *Service) Tag(ctx context.Context, sc favourites.Scope, num int64) (Tag, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Tag{}, err
	}
	info, ok := t.tags()[num]
	if !ok {
		return Tag{}, ErrTagNotFound
	}
	return info.tag(), nil
}

// DeleteTag is DELETE /api/v1/tags/{id}: the tag is removed from the links of
// the subtree; links outside it keep it. It returns the tag as it was.
func (s *Service) DeleteTag(ctx context.Context, sc favourites.Scope, num int64) (Tag, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Tag{}, err
	}
	info, ok := t.tags()[num]
	if !ok {
		return Tag{}, ErrTagNotFound
	}
	if err := s.Favourites.DetachTag(ctx, sc.UserID, info.ID, info.links); err != nil {
		return Tag{}, err
	}
	return info.tag(), nil
}

// CreateTag is POST /api/v1/tags, which Boardly refuses: a tag only exists
// while a link has it.
func (s *Service) CreateTag() error { return ErrTagCreate }
