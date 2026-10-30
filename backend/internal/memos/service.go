package memos

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"boardly/internal/attachments"
	"boardly/internal/notes"
)

// Page sizes of ListMemos, as in Memos.
const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// Service serves the memos of a token's note scope.
type Service struct {
	Notes       *notes.Store
	Attachments *attachments.Store
	// Origin is Boardly's origin, which attachment URLs in memos start with.
	Origin string
}

// ListInput is the query of GET /api/v1/memos.
type ListInput struct {
	State     string
	PageSize  int
	PageToken string
	Filter    string
}

// ListResponse is a page of memos.
type ListResponse struct {
	Memos         []Memo `json:"memos"`
	NextPageToken string `json:"nextPageToken"`
}

// List returns a page of the memos in scope with the given state, newest
// first by creation time, then by ID.
func (s *Service) List(ctx context.Context, sc notes.Scope, in ListInput) (ListResponse, error) {
	archived := false
	switch in.State {
	case "", StateNormal:
	case StateArchived:
		archived = true
	default:
		return ListResponse{}, errState
	}
	offset, err := pageOffset(in.PageToken)
	if err != nil {
		return ListResponse{}, err
	}
	size := in.PageSize
	switch {
	case size <= 0:
		size = defaultPageSize
	case size > maxPageSize:
		size = maxPageSize
	}
	out := ListResponse{Memos: []Memo{}}
	if !matchesFilter(in.Filter, sc.UserID) {
		return out, nil
	}
	ns, err := s.Notes.ScopedNotes(ctx, sc)
	if err != nil {
		return ListResponse{}, err
	}
	ns = slices.DeleteFunc(ns, func(n notes.Note) bool { return n.Archived != archived })
	slices.SortFunc(ns, func(a, b notes.Note) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), strings.Compare(b.ID, a.ID))
	})
	if offset > len(ns) {
		offset = len(ns)
	}
	end := min(offset+size, len(ns))
	for _, n := range ns[offset:end] {
		out.Memos = append(out.Memos, MemoOf(sc.UserID, s.Origin, n))
	}
	if end < len(ns) {
		out.NextPageToken = pageToken(end)
	}
	return out, nil
}

var (
	creatorFilter    = regexp.MustCompile(`^creator\s*==\s*"([^"]*)"$`)
	visibilityIn     = regexp.MustCompile(`^visibility\s+in\s*\[(.*)\]$`)
	visibilityEquals = regexp.MustCompile(`^visibility\s*==\s*(.+)$`)
)

// matchesFilter reports whether the memos of the user pass a Memos filter.
// Only the conditions MoeMemos sends are understood, joined with &&: the
// creator, which must be the user, and the visibility, which must include
// PRIVATE. Other conditions are ignored.
func matchesFilter(filter, userID string) bool {
	for _, cond := range strings.Split(filter, "&&") {
		cond = strings.TrimSpace(cond)
		if m := creatorFilter.FindStringSubmatch(cond); m != nil && m[1] != UserName(userID) {
			return false
		}
		values := ""
		if m := visibilityIn.FindStringSubmatch(cond); m != nil {
			values = m[1]
		} else if m := visibilityEquals.FindStringSubmatch(cond); m != nil {
			values = m[1]
		} else {
			continue
		}
		private := false
		for _, v := range strings.Split(values, ",") {
			private = private || strings.Trim(strings.TrimSpace(v), `"'`) == "PRIVATE"
		}
		if !private {
			return false
		}
	}
	return true
}

// CreateInput is the body of POST /api/v1/memos. Visibility and location are
// ignored.
type CreateInput struct {
	Content     *string         `json:"content"`
	Attachments []AttachmentRef `json:"attachments"`
	CreateTime  *string         `json:"createTime"`
}

// Create adds a note in the scope's root category, or uncategorized for all
// notes, with the title and content from the memo's content, followed by
// its attachments.
func (s *Service) Create(ctx context.Context, sc notes.Scope, in CreateInput) (Memo, error) {
	if in.Content == nil {
		return Memo{}, errContent
	}
	att, err := s.Notes.UserMarkdownAttachments(ctx, sc.UserID, s.Origin)
	if err != nil {
		return Memo{}, err
	}
	ids, err := attachmentIDs(in.Attachments, att)
	if err != nil {
		return Memo{}, err
	}
	title, doc := notes.TitleFromHeading(notes.FromMarkdown(*in.Content, att))
	content, err := json.Marshal(notes.JoinTrailingAttachments(doc, ids))
	if err != nil {
		return Memo{}, err
	}
	n := notes.NewNote{Title: title, Content: content}
	if sc.Mode == notes.ScopeCategory {
		n.CategoryID = &sc.CategoryID
	}
	if in.CreateTime != nil {
		// A time that can't be read means now, like a missing one.
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, *in.CreateTime)
	}
	created, err := s.Notes.CreateNote(ctx, sc.UserID, n)
	if err != nil {
		return Memo{}, storeError(err)
	}
	return MemoOf(sc.UserID, s.Origin, created), nil
}

// UpdateInput is what PATCH /api/v1/memos/{id} changes; nil fields stay.
type UpdateInput struct {
	Content     *string
	Pinned      *bool
	State       *string
	Attachments *[]AttachmentRef
}

// UpdateFromBody reads the fields of a memo update from its body. With a
// mask (comma-separated snake case names), only the listed fields are read;
// without one, every field in the body. Visibility, times and location are
// ignored.
func UpdateFromBody(body map[string]json.RawMessage, mask string) (UpdateInput, error) {
	use := func(name, masked string) (json.RawMessage, bool) {
		raw, ok := body[name]
		if !ok || string(raw) == "null" {
			return nil, false
		}
		if mask == "" {
			return raw, true
		}
		for _, m := range strings.Split(mask, ",") {
			if strings.TrimSpace(m) == masked {
				return raw, true
			}
		}
		return nil, false
	}
	var in UpdateInput
	for _, f := range []struct {
		name, masked string
		dst          any
	}{
		{"content", "content", &in.Content},
		{"pinned", "pinned", &in.Pinned},
		{"state", "state", &in.State},
		{"attachments", "attachments", &in.Attachments},
	} {
		if raw, ok := use(f.name, f.masked); ok {
			if err := json.Unmarshal(raw, f.dst); err != nil {
				return UpdateInput{}, invalid("The field " + f.name + " has the wrong type.")
			}
		}
	}
	return in, nil
}

// Update changes the note of a memo in scope. A change from the app always
// wins: it is applied to the note's current version.
func (s *Service) Update(ctx context.Context, sc notes.Scope, id string, in UpdateInput) (Memo, error) {
	if in.State != nil && *in.State != StateNormal && *in.State != StateArchived {
		return Memo{}, errState
	}
	// A conflict means the note changed between reading and writing it;
	// the change is then applied to the newer version once more.
	for attempt := 0; ; attempt++ {
		n, err := s.Notes.ScopedNote(ctx, sc, id)
		if err != nil {
			return Memo{}, storeError(err)
		}
		patch, err := s.patchOf(ctx, sc.UserID, n, in)
		if err != nil {
			return Memo{}, err
		}
		updated, err := s.Notes.UpdateNote(ctx, sc.UserID, n.ID, n.Version, patch)
		if errors.Is(err, notes.ErrVersionConflict) && attempt == 0 {
			continue
		}
		if err != nil {
			return Memo{}, storeError(err)
		}
		return MemoOf(sc.UserID, s.Origin, updated), nil
	}
}

// patchOf is the change of the note n that in asks for; fields that are
// already as asked are left out, so an unchanged memo isn't a change. The
// content replaces the note's content before its trailing attachments, and
// the attachments replace the trailing attachments.
func (s *Service) patchOf(ctx context.Context, userID string, n notes.Note, in UpdateInput) (notes.NotePatch, error) {
	var p notes.NotePatch
	var att *notes.MarkdownAttachments
	if in.Content != nil || in.Attachments != nil {
		var err error
		if att, err = s.Notes.UserMarkdownAttachments(ctx, userID, s.Origin); err != nil {
			return p, err
		}
	}
	body, trailing := notes.SplitTrailingAttachments(n.Content)
	changed := false
	if in.Content != nil && *in.Content != Content(n, s.Origin) {
		var title *string
		title, body = notes.TitleFromHeading(notes.ReplaceContentFromMarkdown(body, *in.Content, att))
		p.Title, changed = &title, true
	}
	if in.Attachments != nil {
		ids, err := attachmentIDs(*in.Attachments, att)
		if err != nil {
			return p, err
		}
		if !slices.Equal(ids, trailing) {
			trailing, changed = ids, true
		}
	}
	if changed {
		content, err := json.Marshal(notes.JoinTrailingAttachments(body, trailing))
		if err != nil {
			return p, err
		}
		p.Content = content
	}
	if in.Pinned != nil && *in.Pinned != n.Pinned {
		p.Pinned = in.Pinned
	}
	if in.State != nil {
		if archived := *in.State == StateArchived; archived != n.Archived {
			p.Archived = &archived
		}
	}
	return p, nil
}

// Delete deletes the note of a memo in scope.
func (s *Service) Delete(ctx context.Context, sc notes.Scope, id string) error {
	if _, err := s.Notes.ScopedNote(ctx, sc, id); err != nil {
		return storeError(err)
	}
	return storeError(s.Notes.DeleteNote(ctx, sc.UserID, id))
}
