package export

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"boardly/internal/blobs"
	"boardly/internal/db"
	"boardly/internal/notes"
)

// ErrNoUser is returned for a user that doesn't exist.
var ErrNoUser = errors.New("user not found")

// Service builds exports. ImageFiles and AttachmentFiles are where the files
// of images and attachments are kept.
type Service struct {
	DB              *sql.DB
	ImageFiles      blobs.Dir
	AttachmentFiles blobs.Dir
	Now             func() time.Time
}

// Document reads all data of the user in one read-only transaction, so the
// document is a consistent snapshot. The File of every image and attachment
// is nil; WriteArchive fills them in.
func (s *Service) Document(ctx context.Context, userID string) (*Document, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	doc := &Document{
		Format:     Format,
		Version:    Version,
		ExportedAt: db.FormatTime(s.Now()),
	}
	r := reader{ctx: ctx, q: tx, userID: userID}
	for _, step := range []struct {
		name string
		read func(*Document) error
	}{
		{"profile", r.profile},
		{"favourites", r.favourites},
		{"tags", r.tags},
		{"dashboards", r.dashboards},
		{"note categories", r.categories},
		{"notes", r.notes},
		{"media sources", r.mediaSources},
		{"images", r.images},
		{"attachments", r.attachments},
	} {
		if err := step.read(doc); err != nil {
			if errors.Is(err, ErrNoUser) {
				return nil, err
			}
			return nil, fmt.Errorf("reading %s: %w", step.name, err)
		}
	}
	return doc, nil
}

type reader struct {
	ctx    context.Context
	q      db.Querier
	userID string
}

// each runs query for the user and calls scan for every row.
func (r reader) each(query string, scan func(*sql.Rows) error) error {
	rows, err := r.q.QueryContext(r.ctx, query, r.userID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r reader) profile(doc *Document) error {
	p := &doc.Profile
	err := r.q.QueryRowContext(r.ctx,
		`SELECT email, name, slug, created_at, share_folder_id, note_category_id FROM users WHERE id = ?`, r.userID,
	).Scan(&p.Email, &p.Name, &p.Slug, &p.CreatedAt, &p.DefaultFolderID, &p.DefaultNoteCategoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoUser
	}
	return err
}

func (r reader) favourites(doc *Document) error {
	tags := map[string][]string{}
	err := r.each(
		`SELECT nt.node_id, t.name FROM favourite_node_tags nt
		 JOIN tags t ON t.id = nt.tag_id
		 WHERE t.user_id = ? ORDER BY t.name_key, t.id`,
		func(rows *sql.Rows) error {
			var nodeID, name string
			if err := rows.Scan(&nodeID, &name); err != nil {
				return err
			}
			tags[nodeID] = append(tags[nodeID], name)
			return nil
		})
	if err != nil {
		return err
	}

	byID := map[string]*Node{}
	var order []*Node
	parents := map[*Node]string{}
	err = r.each(
		`SELECT id, parent_id, kind, title, url, description, preview_image_id, created_at, updated_at
		 FROM favourite_nodes WHERE user_id = ? ORDER BY position, id`,
		func(rows *sql.Rows) error {
			n := &Node{}
			var parentID, url sql.NullString
			if err := rows.Scan(&n.ID, &parentID, &n.Kind, &n.Title, &url, &n.Description, &n.PreviewImageID, &n.CreatedAt, &n.UpdatedAt); err != nil {
				return err
			}
			n.URL = url.String
			n.Tags = tags[n.ID]
			byID[n.ID] = n
			order = append(order, n)
			if parentID.Valid {
				parents[n] = parentID.String
			}
			return nil
		})
	if err != nil {
		return err
	}
	// Rows come ordered by position, so appending keeps each folder's order.
	for _, n := range order {
		parentID, ok := parents[n]
		if !ok {
			doc.Favourites = n
			continue
		}
		if parent := byID[parentID]; parent != nil {
			parent.Children = append(parent.Children, n)
		}
	}
	return nil
}

func (r reader) tags(doc *Document) error {
	doc.Tags = []Tag{}
	return r.each(
		`SELECT id, name, created_at FROM tags WHERE user_id = ? ORDER BY name_key, id`,
		func(rows *sql.Rows) error {
			var t Tag
			if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
				return err
			}
			doc.Tags = append(doc.Tags, t)
			return nil
		})
}

func (r reader) dashboards(doc *Document) error {
	doc.Dashboards = []Dashboard{}
	index := map[string]int{}
	err := r.each(
		`SELECT id, slug, title, is_default, is_public, desktop_columns, mobile_columns, mobile_breakpoint, created_at, updated_at
		 FROM dashboards WHERE user_id = ? ORDER BY created_at, id`,
		func(rows *sql.Rows) error {
			d := Dashboard{Widgets: []*Widget{}}
			if err := rows.Scan(&d.ID, &d.Slug, &d.Title, &d.IsDefault, &d.IsPublic,
				&d.DesktopColumns, &d.MobileColumns, &d.MobileBreakpoint, &d.CreatedAt, &d.UpdatedAt); err != nil {
				return err
			}
			index[d.ID] = len(doc.Dashboards)
			doc.Dashboards = append(doc.Dashboards, d)
			return nil
		})
	if err != nil {
		return err
	}

	type widgetRow struct {
		w           *Widget
		dashboardID string
		parentID    sql.NullString
	}
	var widgets []widgetRow
	byID := map[string]*Widget{}
	err = r.each(
		`SELECT w.id, w.dashboard_id, w.parent_id, w.kind, w.width, w.config
		 FROM dashboard_widgets w JOIN dashboards d ON d.id = w.dashboard_id
		 WHERE d.user_id = ? ORDER BY w.position, w.id`,
		func(rows *sql.Rows) error {
			row := widgetRow{w: &Widget{}}
			var config string
			if err := rows.Scan(&row.w.ID, &row.dashboardID, &row.parentID, &row.w.Kind, &row.w.Width, &config); err != nil {
				return err
			}
			row.w.Config = json.RawMessage(config)
			byID[row.w.ID] = row.w
			widgets = append(widgets, row)
			return nil
		})
	if err != nil {
		return err
	}
	for _, row := range widgets {
		if row.parentID.Valid {
			if parent := byID[row.parentID.String]; parent != nil {
				parent.Children = append(parent.Children, row.w)
			}
			continue
		}
		if i, ok := index[row.dashboardID]; ok {
			doc.Dashboards[i].Widgets = append(doc.Dashboards[i].Widgets, row.w)
		}
	}
	return nil
}

func (r reader) categories(doc *Document) error {
	doc.NoteCategories = []Category{}
	return r.each(
		`SELECT id, parent_id, title, icon_symbol, icon_image_id, color, created_at, updated_at
		 FROM note_categories WHERE user_id = ? ORDER BY created_at, id`,
		func(rows *sql.Rows) error {
			var c Category
			var symbol, imageID *string
			if err := rows.Scan(&c.ID, &c.ParentID, &c.Title, &symbol, &imageID, &c.Color, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return err
			}
			c.Icon = notes.IconOf(symbol, imageID)
			doc.NoteCategories = append(doc.NoteCategories, c)
			return nil
		})
}

func (r reader) notes(doc *Document) error {
	attachmentIDs := map[string][]string{}
	err := r.each(
		`SELECT na.note_id, na.attachment_id FROM note_attachments na
		 JOIN notes n ON n.id = na.note_id
		 WHERE n.user_id = ? ORDER BY na.position, na.attachment_id`,
		func(rows *sql.Rows) error {
			var noteID, attachmentID string
			if err := rows.Scan(&noteID, &attachmentID); err != nil {
				return err
			}
			attachmentIDs[noteID] = append(attachmentIDs[noteID], attachmentID)
			return nil
		})
	if err != nil {
		return err
	}

	doc.Notes = []Note{}
	return r.each(
		`SELECT id, category_id, title, icon_symbol, icon_image_id, color, content, pinned, archived, created_at, updated_at
		 FROM notes WHERE user_id = ? ORDER BY created_at, id`,
		func(rows *sql.Rows) error {
			var n Note
			var symbol, imageID *string
			var content string
			if err := rows.Scan(&n.ID, &n.CategoryID, &n.Title, &symbol, &imageID, &n.Color, &content,
				&n.Pinned, &n.Archived, &n.CreatedAt, &n.UpdatedAt); err != nil {
				return err
			}
			n.Icon = notes.IconOf(symbol, imageID)
			n.Content = json.RawMessage(content)
			n.AttachmentIDs = attachmentIDs[n.ID]
			if n.AttachmentIDs == nil {
				n.AttachmentIDs = []string{}
			}
			doc.Notes = append(doc.Notes, n)
			return nil
		})
}

func (r reader) mediaSources(doc *Document) error {
	doc.MediaSources = []MediaSource{}
	return r.each(
		`SELECT id, kind, title, url, icon_symbol, icon_image_id, position, created_at, updated_at
		 FROM media_sources WHERE user_id = ? ORDER BY position, id`,
		func(rows *sql.Rows) error {
			var m MediaSource
			var symbol, imageID *string
			if err := rows.Scan(&m.ID, &m.Kind, &m.Title, &m.URL, &symbol, &imageID, &m.Position, &m.CreatedAt, &m.UpdatedAt); err != nil {
				return err
			}
			m.Icon = notes.IconOf(symbol, imageID)
			doc.MediaSources = append(doc.MediaSources, m)
			return nil
		})
}

func (r reader) images(doc *Document) error {
	doc.Images = []Image{}
	return r.each(
		`SELECT id, source, content_type, size, created_at FROM images
		 WHERE user_id = ? AND source IN ('upload', 'preview') ORDER BY created_at, id`,
		func(rows *sql.Rows) error {
			var img Image
			if err := rows.Scan(&img.ID, &img.Source, &img.ContentType, &img.Size, &img.CreatedAt); err != nil {
				return err
			}
			doc.Images = append(doc.Images, img)
			return nil
		})
}

func (r reader) attachments(doc *Document) error {
	doc.Attachments = []Attachment{}
	return r.each(
		`SELECT id, name, content_type, size, created_at FROM attachments
		 WHERE user_id = ? AND deleted_at IS NULL ORDER BY created_at, id`,
		func(rows *sql.Rows) error {
			var a Attachment
			if err := rows.Scan(&a.ID, &a.Name, &a.ContentType, &a.Size, &a.CreatedAt); err != nil {
				return err
			}
			doc.Attachments = append(doc.Attachments, a)
			return nil
		})
}
