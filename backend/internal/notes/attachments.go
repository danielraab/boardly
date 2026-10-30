package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"boardly/internal/attachments"
	"boardly/internal/db"
)

// NoteAttachment is an attachment as a note returns it.
type NoteAttachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
}

// noteAttachmentsColumn selects a note's attachments as a JSON array, in
// content order. It is part of noteColumns, so every note that is read has
// them.
const noteAttachmentsColumn = `COALESCE((
	SELECT json_group_array(json_object('id', a.id, 'name', a.name, 'contentType', a.content_type, 'size', a.size) ORDER BY na.position)
	FROM note_attachments AS na JOIN attachments AS a ON a.id = na.attachment_id
	WHERE na.note_id = notes.id), '[]')`

// scanNoteAttachments decodes noteAttachmentsColumn into n.Attachments.
func scanNoteAttachments(raw string, n *Note) error {
	n.Attachments = []NoteAttachment{}
	if err := json.Unmarshal([]byte(raw), &n.Attachments); err != nil {
		return err
	}
	for i := range n.Attachments {
		a := &n.Attachments[i]
		a.URL = attachments.URL(a.ID, a.Name)
	}
	return nil
}

// setAttachments makes the note refer to the attachments its content
// contains, which must all be the user's, and removes the deletion mark of
// those that are marked. Other IDs return ErrInvalidDocument.
func setAttachments(ctx context.Context, q db.Querier, userID, noteID string, d Doc) error {
	ids := AttachmentIDs(d)
	if len(ids) > 0 {
		placeholders := "?" + strings.Repeat(", ?", len(ids)-1)
		args := make([]any, 0, len(ids)+1)
		args = append(args, userID)
		for _, id := range ids {
			args = append(args, id)
		}
		var found int
		if err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM attachments WHERE user_id = ? AND id IN (`+placeholders+`)`, args...,
		).Scan(&found); err != nil {
			return err
		}
		if found != len(ids) {
			return fmt.Errorf("%w: unknown attachment", ErrInvalidDocument)
		}
		if _, err := q.ExecContext(ctx,
			`UPDATE attachments SET deleted_at = NULL
			 WHERE user_id = ? AND id IN (`+placeholders+`) AND deleted_at IS NOT NULL`, args...,
		); err != nil {
			return err
		}
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM note_attachments WHERE note_id = ?`, noteID); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO note_attachments (note_id, attachment_id, position) VALUES (?, ?, ?)`, noteID, id, i,
		); err != nil {
			return err
		}
	}
	return nil
}

// MarkdownAttachments lets Markdown conversion handle attachments: Origin is
// what their URLs start with (Boardly's origin), and ByID holds the
// attachments that may appear.
type MarkdownAttachments struct {
	Origin string
	ByID   map[string]NoteAttachment
}

// NoteMarkdownAttachments returns the attachments of a note, for writing its
// content as Markdown.
func NoteMarkdownAttachments(origin string, n Note) *MarkdownAttachments {
	m := &MarkdownAttachments{Origin: origin, ByID: map[string]NoteAttachment{}}
	for _, a := range n.Attachments {
		m.ByID[a.ID] = a
	}
	return m
}

// UserMarkdownAttachments returns all of the user's attachments, for reading
// Markdown that may refer to any of them.
func (s *Store) UserMarkdownAttachments(ctx context.Context, userID, origin string) (*MarkdownAttachments, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, name, content_type, size FROM attachments WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := &MarkdownAttachments{Origin: origin, ByID: map[string]NoteAttachment{}}
	for rows.Next() {
		var a NoteAttachment
		if err := rows.Scan(&a.ID, &a.Name, &a.ContentType, &a.Size); err != nil {
			return nil, err
		}
		a.URL = attachments.URL(a.ID, a.Name)
		m.ByID[a.ID] = a
	}
	return m, rows.Err()
}

// attachmentPathRe matches the path of an attachment URL.
var attachmentPathRe = regexp.MustCompile(`^/api/attachments/([A-Za-z0-9-]+)(?:/[^?#]*)?$`)

// id returns the ID of the attachment that u is the URL of, as
// <origin>/api/attachments/<id>/<name> or as the path alone, or "" when u
// isn't the URL of one of m's attachments.
func (m *MarkdownAttachments) id(u string) string {
	if m == nil {
		return ""
	}
	if m.Origin != "" && strings.HasPrefix(u, m.Origin+"/") {
		u = u[len(m.Origin):]
	}
	match := attachmentPathRe.FindStringSubmatch(u)
	if match == nil {
		return ""
	}
	if _, ok := m.ByID[match[1]]; !ok {
		return ""
	}
	return match[1]
}

// markdown writes an attachment: an image as an image, any other file as a
// link. An attachment m doesn't know is written as a link named "Attachment".
func (m *MarkdownAttachments) markdown(id string) string {
	var (
		a      NoteAttachment
		ok     bool
		origin string
	)
	if m != nil {
		origin = m.Origin
		a, ok = m.ByID[id]
	}
	if !ok {
		a = NoteAttachment{ID: id, Name: "Attachment"}
	}
	// Parentheses would end the link's address.
	u := origin + strings.NewReplacer("(", "%28", ")", "%29").Replace(attachments.URL(a.ID, a.Name))
	text := writeInlines([]Inline{textRun(a.Name, nil)})
	if t, known := attachments.TypeOf(a.ContentType); ok && known && t.Group() == "image" {
		return "![" + text + "](" + u + ")"
	}
	return "[" + text + "](" + u + ")"
}
