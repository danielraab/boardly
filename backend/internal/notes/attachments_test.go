package notes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"boardly/internal/db"
)

// attachment adds an attachment of the user directly, and returns its ID.
func (f *fixture) attachment(userID, id, name string) string {
	f.t.Helper()
	if _, err := f.db.Exec(
		`INSERT INTO attachments (id, user_id, name, content_type, size, created_at) VALUES (?, ?, ?, 'application/pdf', 10, ?)`,
		id, userID, name, db.FormatTime(f.now)); err != nil {
		f.t.Fatal(err)
	}
	return id
}

// attachmentDoc is a document with a paragraph and the attachments.
func attachmentDoc(ids ...string) json.RawMessage {
	d := Doc{V: DocVersion, Blocks: []Block{{Type: BlockParagraph, Content: &[]Inline{}}}}
	for _, id := range ids {
		d.Blocks = append(d.Blocks, Block{Type: BlockAttachment, ID: id})
	}
	b, _ := json.Marshal(d)
	return b
}

func (f *fixture) refs(attachmentID string) []string {
	f.t.Helper()
	rows, err := f.db.Query(`SELECT note_id FROM note_attachments WHERE attachment_id = ? ORDER BY note_id`, attachmentID)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			f.t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func (f *fixture) marked(attachmentID string) bool {
	f.t.Helper()
	var deleted sql.NullString
	if err := f.db.QueryRow(`SELECT deleted_at FROM attachments WHERE id = ?`, attachmentID).Scan(&deleted); err != nil {
		f.t.Fatal(err)
	}
	return deleted.Valid
}

func TestNoteAttachments(t *testing.T) {
	f := newFixture(t)
	roses := f.attachment(f.user, "00000000-0000-4000-8000-000000000001", "roses.jpg")
	plan := f.attachment(f.user, "00000000-0000-4000-8000-000000000002", "my plan.pdf")

	n, err := f.s.CreateNote(f.ctx, f.user, NewNote{Content: attachmentDoc(roses, plan, roses)})
	if err != nil {
		t.Fatal(err)
	}
	want := []NoteAttachment{
		{ID: roses, Name: "roses.jpg", ContentType: "application/pdf", Size: 10, URL: "/api/attachments/" + roses + "/roses.jpg", CreatedAt: f.now},
		{ID: plan, Name: "my plan.pdf", ContentType: "application/pdf", Size: 10, URL: "/api/attachments/" + plan + "/my%20plan.pdf", CreatedAt: f.now},
	}
	if !reflect.DeepEqual(n.Attachments, want) {
		t.Fatalf("Attachments = %+v, want %+v", n.Attachments, want)
	}
	got, err := f.s.Note(f.ctx, f.user, n.ID)
	if err != nil || !reflect.DeepEqual(got.Attachments, want) {
		t.Fatalf("Note().Attachments = %+v, %v", got.Attachments, err)
	}
	all, err := f.s.Notes(f.ctx, f.user)
	if err != nil || len(all) != 1 || len(all[0].Attachments) != 2 {
		t.Fatalf("Notes() = %+v, %v", all, err)
	}

	// Removed from the content: the reference goes, the attachment stays.
	n, err = f.s.UpdateNote(f.ctx, f.user, n.ID, n.Version, NotePatch{Content: attachmentDoc(roses)})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Attachments) != 1 || n.Attachments[0].ID != roses || len(f.refs(plan)) != 0 {
		t.Fatalf("after removing: %+v, refs %v", n.Attachments, f.refs(plan))
	}

	// A change that doesn't touch the content keeps the references.
	n, err = f.s.UpdateNote(f.ctx, f.user, n.ID, n.Version, NotePatch{Archived: ptr(true)})
	if err != nil || !reflect.DeepEqual(f.refs(roses), []string{n.ID}) {
		t.Fatalf("archived note refs = %v, %v", f.refs(roses), err)
	}

	// Deleting the note removes the references, not the attachment.
	if err := f.s.DeleteNote(f.ctx, f.user, n.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM attachments`).Scan(&count); err != nil || count != 2 || len(f.refs(roses)) != 0 {
		t.Fatalf("after delete: %d attachments, refs %v, %v", count, f.refs(roses), err)
	}
}

func TestNoteAttachmentsOwnership(t *testing.T) {
	f := newFixture(t)
	bobs := f.attachment(f.other, "00000000-0000-4000-8000-000000000003", "bob.pdf")
	n := f.note("Plan", nil)

	if _, err := f.s.CreateNote(f.ctx, f.user, NewNote{Content: attachmentDoc(bobs)}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("create with Bob's attachment: %v", err)
	}
	if _, err := f.s.UpdateNote(f.ctx, f.user, n.ID, n.Version, NotePatch{Content: attachmentDoc(bobs)}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("update with Bob's attachment: %v", err)
	}
	if _, err := f.s.CreateNote(f.ctx, f.user, NewNote{Content: attachmentDoc("00000000-0000-4000-8000-00000000dead")}); !errors.Is(err, ErrInvalidDocument) {
		t.Fatalf("create with an unknown attachment: %v", err)
	}
	if refs := f.refs(bobs); len(refs) != 0 {
		t.Fatalf("Bob's attachment got references %v", refs)
	}
}

func TestNoteAttachmentReferredToAgain(t *testing.T) {
	f := newFixture(t)
	report := f.attachment(f.user, "00000000-0000-4000-8000-000000000004", "report.pdf")
	other := f.attachment(f.user, "00000000-0000-4000-8000-000000000005", "other.pdf")
	if _, err := f.db.Exec(`UPDATE attachments SET deleted_at = ?`, db.FormatTime(f.now)); err != nil {
		t.Fatal(err)
	}
	n := f.note("Taxes 2026", nil)
	if _, err := f.s.UpdateNote(f.ctx, f.user, n.ID, n.Version, NotePatch{Content: attachmentDoc(report)}); err != nil {
		t.Fatal(err)
	}
	if f.marked(report) || !f.marked(other) {
		t.Fatalf("marked: report %v, other %v; want false, true", f.marked(report), f.marked(other))
	}
}
