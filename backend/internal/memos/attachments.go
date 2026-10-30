package memos

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"boardly/internal/attachments"
	"boardly/internal/notes"
)

// Attachment is a Boardly attachment as a Memos attachment. ExternalLink is
// its Boardly URL, which MoeMemos shows and downloads it from.
type Attachment struct {
	Name         string `json:"name"`
	CreateTime   Time   `json:"createTime"`
	Filename     string `json:"filename"`
	Type         string `json:"type"`
	Size         string `json:"size"`
	ExternalLink string `json:"externalLink"`
	Memo         string `json:"memo,omitempty"`
}

// AttachmentRef is an attachment in a request, by its name.
type AttachmentRef struct {
	Name string `json:"name"`
}

// attachmentPrefix starts the Memos name of an attachment,
// "attachments/<id>".
const attachmentPrefix = "attachments/"

// attachmentOf is the Memos attachment of a note's attachment; memo is the
// name of the memo it is listed in, or "".
func attachmentOf(origin string, a notes.NoteAttachment, memo string) Attachment {
	return Attachment{
		Name: attachmentPrefix + a.ID, CreateTime: Time(a.CreatedAt), Filename: a.Name, Type: a.ContentType,
		Size: strconv.FormatInt(a.Size, 10), ExternalLink: origin + attachments.URL(a.ID, a.Name), Memo: memo,
	}
}

// memoAttachments are the note's trailing attachments as Memos attachments.
func memoAttachments(origin string, n notes.Note) []Attachment {
	_, ids := notes.SplitTrailingAttachments(n.Content)
	out := []Attachment{}
	for _, id := range ids {
		for _, a := range n.Attachments {
			if a.ID == id {
				out = append(out, attachmentOf(origin, a, memoName(n.ID)))
				break
			}
		}
	}
	return out
}

// attachmentIDs resolves attachments of a request against the user's
// attachments in att.
func attachmentIDs(refs []AttachmentRef, att *notes.MarkdownAttachments) ([]string, error) {
	ids := make([]string, 0, len(refs))
	for _, r := range refs {
		id, ok := strings.CutPrefix(r.Name, attachmentPrefix)
		if _, known := att.ByID[id]; !ok || !known {
			return nil, errAttachment
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UploadInput is the body of POST /api/v1/attachments. The type is found
// from the file, and the memo is set by the memo update that follows.
type UploadInput struct {
	Filename string  `json:"filename"`
	Content  *string `json:"content"`
}

// Upload stores a file as an attachment of the user.
func (s *Service) Upload(ctx context.Context, userID string, in UploadInput) (Attachment, error) {
	if in.Content == nil || *in.Content == "" {
		return Attachment{}, invalid("An attachment needs a content.")
	}
	data, err := decodeBase64(*in.Content)
	if err != nil {
		return Attachment{}, invalid("The content is not base64.")
	}
	a, err := s.Attachments.Create(ctx, userID, in.Filename, data)
	if err != nil {
		return Attachment{}, s.attachmentError(err)
	}
	return attachmentOf(s.Origin, notes.NoteAttachment{
		ID: a.ID, Name: a.Name, ContentType: a.ContentType, Size: a.Size, CreatedAt: a.CreatedAt,
	}, ""), nil
}

// decodeBase64 reads standard base64, which protobuf JSON writes, or
// URL-safe base64, which it also accepts, with or without padding.
func decodeBase64(s string) ([]byte, error) {
	var err error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		var data []byte
		if data, err = enc.DecodeString(s); err == nil {
			return data, nil
		}
	}
	return nil, err
}

// ErrUploadTooLarge is the error for a file or upload body above the
// maximum size.
func (s *Service) ErrUploadTooLarge() error {
	return invalid(fmt.Sprintf("Attachments can be at most %d MB.", s.Attachments.MaxBytes>>20))
}

// attachmentError maps the errors of the attachment store to Memos errors;
// other errors pass through.
func (s *Service) attachmentError(err error) error {
	switch {
	case errors.Is(err, attachments.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, attachments.ErrInvalidName):
		return invalid("Invalid file name.")
	case errors.Is(err, attachments.ErrUnsupported):
		return invalid("This type of file can't be attached.")
	case errors.Is(err, attachments.ErrTooLarge):
		return s.ErrUploadTooLarge()
	}
	return err
}

// AttachmentList is the response of GET /api/v1/attachments.
type AttachmentList struct {
	Attachments   []Attachment `json:"attachments"`
	NextPageToken string       `json:"nextPageToken"`
}

// ListAttachments returns the attachments that the notes in scope refer to,
// each once, newest first, then by ID.
func (s *Service) ListAttachments(ctx context.Context, sc notes.Scope) (AttachmentList, error) {
	ns, err := s.Notes.ScopedNotes(ctx, sc)
	if err != nil {
		return AttachmentList{}, err
	}
	seen := map[string]bool{}
	var list []notes.NoteAttachment
	for _, n := range ns {
		for _, a := range n.Attachments {
			if !seen[a.ID] {
				seen[a.ID] = true
				list = append(list, a)
			}
		}
	}
	slices.SortFunc(list, func(a, b notes.NoteAttachment) int {
		return cmp.Or(b.CreatedAt.Compare(a.CreatedAt), strings.Compare(a.ID, b.ID))
	})
	out := AttachmentList{Attachments: []Attachment{}}
	for _, a := range list {
		out.Attachments = append(out.Attachments, attachmentOf(s.Origin, a, ""))
	}
	return out, nil
}

// DeleteAttachment accepts the deletion of one of the user's attachments
// that a note in scope or no note refers to, without deleting it: MoeMemos
// deletes a file after removing it from a memo, and other notes may still
// show it. Unused files are deleted by the cleanup.
func (s *Service) DeleteAttachment(ctx context.Context, sc notes.Scope, id string) error {
	a, err := s.Attachments.Get(ctx, sc.UserID, id)
	if err != nil {
		return s.attachmentError(err)
	}
	if len(a.NoteIDs) == 0 {
		return nil
	}
	ns, err := s.Notes.ScopedNotes(ctx, sc)
	if err != nil {
		return err
	}
	for _, n := range ns {
		if slices.Contains(a.NoteIDs, n.ID) {
			return nil
		}
	}
	return ErrNotFound
}
