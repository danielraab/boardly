package dataimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"boardly/internal/attachments"
	"boardly/internal/blobs"
	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/export"
	"boardly/internal/favourites"
	"boardly/internal/feeds"
	"boardly/internal/images"
	"boardly/internal/media"
	"boardly/internal/notes"
	"boardly/internal/offline"
)

// Errors of an import besides ErrInvalidFile and those of the database.
var (
	ErrInvalidFolder   = errors.New("invalid import folder")
	ErrInvalidCategory = errors.New("invalid import category")
)

// maxPreviewBytes is the largest link preview image read from an archive.
// Previews are stored converted, so they are usually far smaller.
const maxPreviewBytes = 5 << 20

// Parts are the parts of an export to import.
type Parts struct {
	Favourites, Dashboards, Notes, Media, Feeds bool
}

// Options are the choices of an import.
type Options struct {
	Parts Parts
	// FolderID is the import folder; required with Parts.Favourites.
	FolderID string
	// CategoryID is the import category for notes; nil is the top level.
	CategoryID     *string
	SkipDuplicates bool
}

// Report is what an import added and left out, per part; a part that wasn't
// imported is nil.
type Report struct {
	Favourites *FavouritesReport `json:"favourites"`
	Dashboards *DashboardsReport `json:"dashboards"`
	Notes      *NotesReport      `json:"notes"`
	Media      *MediaReport      `json:"media"`
	Feeds      *FeedsReport      `json:"feeds"`
	Files      FilesReport       `json:"files"`
}

// FavouritesReport extends the bookmark import's report by the created tags.
type FavouritesReport struct {
	favourites.ImportReport
	Tags int `json:"tags"`
}

// DashboardsReport counts the added dashboards and the widgets left out.
type DashboardsReport struct {
	Dashboards     int `json:"dashboards"`
	SkippedWidgets int `json:"skippedWidgets"`
}

// NotesReport counts the imported notes and categories and the duplicates.
type NotesReport struct {
	Notes          int `json:"notes"`
	Categories     int `json:"categories"`
	DuplicateNotes int `json:"duplicateNotes"`
}

// FeedsReport counts the added subscriptions, duplicates and invalid ones,
// and the flagged items that were restored or skipped.
type FeedsReport struct {
	Subscriptions int `json:"subscriptions"`
	Duplicates    int `json:"duplicates"`
	Invalid       int `json:"invalid"`
	RestoredItems int `json:"restoredItems"`
	SkippedItems  int `json:"skippedItems"`
}

// MediaReport counts the added media sources, duplicates and invalid ones.
type MediaReport struct {
	Sources    int `json:"sources"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}

// FilesReport counts the imported and skipped files: images, attachments
// and the parts (articles and screenshots) of offline copies.
type FilesReport struct {
	Images        int `json:"images"`
	Attachments   int `json:"attachments"`
	OfflineCopies int `json:"offlineCopies"`
	Skipped       int `json:"skipped"`
}

// Service imports Boardly exports.
type Service struct {
	DB          *sql.DB
	Favourites  *favourites.Store
	Notes       *notes.Store
	Dashboards  *dashboards.Store
	Media       *media.Store
	Feeds       *feeds.Store
	Images      *images.Store
	Attachments *attachments.Store
	// OfflineFiles is where the parts of offline copies are stored; empty
	// imports no offline copies.
	OfflineFiles blobs.Dir
	// OfflineMaxBytes is the largest article read from an archive.
	OfflineMaxBytes int64
	// Screenshots converts imported screenshots like captured ones.
	Screenshots images.Converter
}

// run is the state of one import.
type run struct {
	s      *Service
	ctx    context.Context
	userID string
	file   *File
	doc    *export.Document
	opts   Options
	report Report
	// ids maps IDs of the file to IDs in the account: of imported items, of
	// existing items they were matched to, and of the user's own files.
	ids map[string]string
	// images and attachments are the files written for this import, whose
	// records are stored in the transaction.
	images      []images.Image
	attachments []attachments.Attachment
	// available holds the account IDs of the images and attachments the
	// imported items may use.
	available map[string]bool
	// offline maps the file's link IDs to the offline copies written for
	// them; offlineFiles are the files written.
	offline      map[string]*favourites.ImportOfflineCopy
	offlineFiles []string
}

// Import adds the chosen parts of the export in data to the user's account.
func (s *Service) Import(ctx context.Context, userID string, data []byte, opts Options) (Report, error) {
	file, err := Parse(data)
	if err != nil {
		return Report{}, err
	}
	r := &run{s: s, ctx: ctx, userID: userID, file: file, doc: file.Doc, opts: opts,
		ids: map[string]string{}, available: map[string]bool{}, offline: map[string]*favourites.ImportOfflineCopy{}}
	if err := r.checkTargets(); err != nil {
		return Report{}, err
	}
	if err := r.writeFiles(); err != nil {
		return Report{}, err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, step := range []func(db.Querier) error{
		r.insertFiles, r.importFavourites, r.importNotes, r.importMedia, r.importFeeds, r.importDashboards, r.setDefaults,
	} {
		if err := step(tx); err != nil {
			return Report{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Report{}, err
	}
	r.dropUnusedOffline()
	if r.report.Favourites != nil {
		s.Favourites.AfterImport(r.report.Favourites.ImportReport)
	}
	return r.report, nil
}

// checkTargets checks the import folder and category before anything is written.
func (r *run) checkTargets() error {
	if r.opts.Parts.Favourites {
		var kind string
		err := r.s.DB.QueryRowContext(r.ctx,
			`SELECT kind FROM favourite_nodes WHERE id = ? AND user_id = ?`, r.opts.FolderID, r.userID).Scan(&kind)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && kind != string(favourites.KindFolder)) {
			return ErrInvalidFolder
		}
		if err != nil {
			return err
		}
	}
	if r.opts.Parts.Notes && r.opts.CategoryID != nil {
		var one int
		err := r.s.DB.QueryRowContext(r.ctx,
			`SELECT 1 FROM note_categories WHERE id = ? AND user_id = ?`, *r.opts.CategoryID, r.userID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidCategory
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// usedImages returns the IDs of the file's images that the chosen parts use.
func (r *run) usedImages() map[string]bool {
	known := map[string]bool{}
	for _, img := range r.doc.Images {
		known[img.ID] = true
	}
	used := map[string]bool{}
	use := func(id *string) {
		if id != nil && known[*id] {
			used[*id] = true
		}
	}
	useIcon := func(icon *notes.Icon) {
		if icon != nil && icon.Type == notes.IconImage {
			use(&icon.ImageID)
		}
	}
	if r.opts.Parts.Favourites && r.doc.Favourites != nil {
		var walk func(n *export.Node)
		walk = func(n *export.Node) {
			use(n.PreviewImageID)
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(r.doc.Favourites)
	}
	if r.opts.Parts.Notes {
		for _, c := range r.doc.NoteCategories {
			useIcon(c.Icon)
		}
		for _, n := range r.doc.Notes {
			useIcon(n.Icon)
		}
	}
	if r.opts.Parts.Media {
		for _, m := range r.doc.MediaSources {
			useIcon(m.Icon)
		}
	}
	if r.opts.Parts.Feeds {
		for _, f := range r.doc.Feeds {
			useIcon(f.Icon)
		}
	}
	if r.opts.Parts.Dashboards {
		var walk func(ws []*export.Widget)
		walk = func(ws []*export.Widget) {
			for _, w := range ws {
				stringsIn(w.Config, func(s string) { use(&s) })
				walk(w.Children)
			}
		}
		for _, d := range r.doc.Dashboards {
			walk(d.Widgets)
		}
	}
	return used
}

// usedAttachments returns the IDs of the file's attachments that the
// imported notes use.
func (r *run) usedAttachments() map[string]bool {
	used := map[string]bool{}
	if !r.opts.Parts.Notes {
		return used
	}
	for _, n := range r.doc.Notes {
		for _, id := range n.AttachmentIDs {
			used[id] = true
		}
		var d notes.Doc
		if json.Unmarshal(n.Content, &d) == nil {
			for _, id := range notes.AttachmentIDs(d) {
				used[id] = true
			}
		}
	}
	return used
}

// ownIDs returns which of the IDs are rows of the user in table.
func (r *run) ownIDs(table string, ids map[string]bool) (map[string]bool, error) {
	own := map[string]bool{}
	for id := range ids {
		var one int
		err := r.s.DB.QueryRowContext(r.ctx,
			`SELECT 1 FROM `+table+` WHERE id = ? AND user_id = ?`, id, r.userID).Scan(&one)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		own[id] = true
	}
	return own, nil
}

// writeFiles writes the files of the used images and attachments that the
// user doesn't have yet. A file that is missing or not accepted is skipped.
func (r *run) writeFiles() error {
	usedImages := r.usedImages()
	ownImages, err := r.ownIDs("images", usedImages)
	if err != nil {
		return err
	}
	for _, img := range r.doc.Images {
		if !usedImages[img.ID] {
			continue
		}
		if ownImages[img.ID] {
			r.ids[img.ID], r.available[img.ID] = img.ID, true
			continue
		}
		limit := int64(images.MaxSize)
		switch img.Source {
		case images.SourceUpload:
		case images.SourcePreview:
			limit = maxPreviewBytes
		default:
			r.report.Files.Skipped++
			continue
		}
		data, err := r.file.ReadFile(img.File, limit)
		if err != nil {
			r.report.Files.Skipped++
			continue
		}
		written, err := r.s.Images.Write(img.Source, data)
		if err != nil {
			r.report.Files.Skipped++
			continue
		}
		r.images = append(r.images, written)
		r.ids[img.ID], r.available[written.ID] = written.ID, true
	}

	usedAttachments := r.usedAttachments()
	ownAttachments, err := r.ownIDs("attachments", usedAttachments)
	if err != nil {
		return err
	}
	for _, a := range r.doc.Attachments {
		if !usedAttachments[a.ID] {
			continue
		}
		if ownAttachments[a.ID] {
			r.ids[a.ID], r.available[a.ID] = a.ID, true
			continue
		}
		data, err := r.file.ReadFile(a.File, r.s.Attachments.MaxBytes)
		if err != nil {
			r.report.Files.Skipped++
			continue
		}
		written, err := r.s.Attachments.Write(a.Name, data)
		if err != nil {
			r.report.Files.Skipped++
			continue
		}
		r.attachments = append(r.attachments, written)
		r.ids[a.ID], r.available[written.ID] = written.ID, true
	}
	return r.writeOffline()
}

// writeOffline stores the articles and screenshots of the imported links'
// offline copies: articles are sanitized again and screenshots converted
// again, because the file can't be trusted. A part that can't be read or
// isn't valid is skipped and counted; a missing path means the copy doesn't
// have the part, or the export has no files.
func (r *run) writeOffline() error {
	if !r.opts.Parts.Favourites || r.doc.Favourites == nil || r.s.OfflineFiles == "" {
		return nil
	}
	var firstErr error
	var walk func(n *export.Node)
	walk = func(n *export.Node) {
		for _, c := range n.Children {
			walk(c)
		}
		c := n.OfflineCopy
		if n.Kind != string(favourites.KindLink) || c == nil || firstErr != nil {
			return
		}
		imported := &favourites.ImportOfflineCopy{CapturedAt: parseTime(c.CapturedAt), Title: c.Title}
		if c.Article != nil {
			data, err := r.file.ReadFile(c.Article, r.s.OfflineMaxBytes)
			title := ""
			if c.Title != nil {
				title = *c.Title
			}
			var clean []byte
			ok := err == nil
			if ok {
				clean, ok = offline.SanitizeDocument(data, title, n.URL)
			}
			if !ok {
				r.report.Files.Skipped++
			} else if imported.ArticleID, err = r.writeOfflineFile(clean); err != nil {
				firstErr = err
				return
			}
		}
		if c.Screenshot != nil {
			data, err := r.file.ReadFile(c.Screenshot, maxPreviewBytes)
			var out []byte
			if err == nil && screenshotType(data) {
				out, err = r.s.Screenshots.Convert(data)
			} else if err == nil {
				err = images.ErrUnsupported
			}
			if err != nil {
				r.report.Files.Skipped++
			} else if imported.ScreenshotID, err = r.writeOfflineFile(out); err != nil {
				firstErr = err
				return
			}
		}
		if imported.ArticleID != nil || imported.ScreenshotID != nil {
			r.offline[n.ID] = imported
		}
	}
	walk(r.doc.Favourites)
	return firstErr
}

func (r *run) writeOfflineFile(data []byte) (*string, error) {
	id := blobs.NewID()
	if err := r.s.OfflineFiles.Write(id, data); err != nil {
		return nil, err
	}
	r.offlineFiles = append(r.offlineFiles, id)
	return &id, nil
}

// screenshotType reports whether data is a JPEG, PNG or WebP image.
func screenshotType(data []byte) bool {
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}

// dropUnusedOffline counts the offline copy parts that imported links got
// and removes those of links that were skipped, such as duplicates.
func (r *run) dropUnusedOffline() {
	for _, c := range r.offline {
		for _, id := range []*string{c.ArticleID, c.ScreenshotID} {
			if id == nil {
				continue
			}
			if c.Imported {
				r.report.Files.OfflineCopies++
			} else {
				_ = r.s.OfflineFiles.Remove(*id)
			}
		}
	}
}

func (r *run) insertFiles(q db.Querier) error {
	for _, img := range r.images {
		if err := r.s.Images.Insert(r.ctx, q, img, r.userID); err != nil {
			return err
		}
	}
	for _, a := range r.attachments {
		if err := r.s.Attachments.Insert(r.ctx, q, r.userID, a); err != nil {
			return err
		}
	}
	r.report.Files.Images, r.report.Files.Attachments = len(r.images), len(r.attachments)
	return nil
}

// mappedImage returns the account ID of the file's image id, if it is available.
func (r *run) mappedImage(id *string) *string {
	if id == nil {
		return nil
	}
	if mapped, ok := r.ids[*id]; ok && r.available[mapped] {
		return &mapped
	}
	return nil
}

// icon returns the icon with its image mapped, or nil if its image isn't available.
func (r *run) icon(icon *notes.Icon) *notes.Icon {
	if icon == nil || icon.Type != notes.IconImage {
		return icon
	}
	if id := r.mappedImage(&icon.ImageID); id != nil {
		return &notes.Icon{Type: notes.IconImage, ImageID: *id}
	}
	return nil
}

// parseTime reads a time of the file: as the export writes it, or else in
// any RFC 3339 form. An invalid time is zero.
func parseTime(s string) time.Time {
	t, err := db.ParseTime(s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339Nano, s)
	}
	return t
}

// invalid turns the validation errors of the stores into ErrInvalidFile.
func invalid(err error) error {
	for _, target := range []error{
		favourites.ErrInvalidTitle, favourites.ErrInvalidTag, favourites.ErrInvalidDescription, favourites.ErrInvalidURL,
		notes.ErrInvalidTitle, notes.ErrInvalidIcon, notes.ErrInvalidColor, notes.ErrInvalidDocument,
		dashboards.ErrInvalidTitle,
	} {
		if errors.Is(err, target) {
			return fmt.Errorf("%w: %v", ErrInvalidFile, err)
		}
	}
	return err
}

func (r *run) importFavourites(q db.Querier) error {
	if !r.opts.Parts.Favourites {
		return nil
	}
	report := &FavouritesReport{}
	r.report.Favourites = report

	names := make([]string, 0, len(r.doc.Tags))
	for _, t := range r.doc.Tags {
		names = append(names, t.Name)
	}
	tagIDs, created, err := r.s.Favourites.EnsureTagsTx(r.ctx, q, r.userID, names)
	if err != nil {
		return invalid(err)
	}
	report.Tags = created
	for _, t := range r.doc.Tags {
		if id, ok := tagIDs[favourites.TagKey(t.Name)]; ok {
			r.ids[t.ID] = id
		}
	}

	var items func(nodes []*export.Node) []favourites.ImportItem
	items = func(nodes []*export.Node) []favourites.ImportItem {
		out := make([]favourites.ImportItem, 0, len(nodes))
		for _, n := range nodes {
			if n.Kind == string(favourites.KindFolder) {
				name := ""
				if n.Title != nil {
					name = *n.Title
				}
				out = append(out, favourites.ImportItem{Folder: &favourites.ImportFolder{
					Name: name, SourceID: n.ID, Items: items(n.Children),
				}})
				continue
			}
			out = append(out, favourites.ImportItem{Link: &favourites.ImportLink{
				URL: n.URL, Title: n.Title, Tags: n.Tags, Description: n.Description,
				CreatedAt: parseTime(n.CreatedAt), SourceID: n.ID, PreviewImageID: r.mappedImage(n.PreviewImageID),
				OfflineCopy: r.offline[n.ID],
			}})
		}
		return out
	}
	var root []favourites.ImportItem
	if r.doc.Favourites != nil {
		root = items(r.doc.Favourites.Children)
		// The file's root is the import folder.
		r.ids[r.doc.Favourites.ID] = r.opts.FolderID
	}
	tree := favourites.ImportTree{Root: favourites.ImportFolder{Items: root}, IDs: r.ids}
	imported, err := r.s.Favourites.ImportTx(r.ctx, q, r.userID, r.opts.FolderID, tree, r.opts.SkipDuplicates)
	if err != nil {
		return invalid(err)
	}
	report.ImportReport = imported
	return nil
}

func (r *run) importNotes(q db.Querier) error {
	if !r.opts.Parts.Notes {
		return nil
	}
	known := map[string]bool{}
	for _, c := range r.doc.NoteCategories {
		known[c.ID] = true
	}
	children := map[string][]export.Category{} // by parent ID, "" for the top level
	for _, c := range r.doc.NoteCategories {
		parent := ""
		if c.ParentID != nil && known[*c.ParentID] {
			parent = *c.ParentID
		}
		children[parent] = append(children[parent], c)
	}
	notesIn := map[string][]notes.ImportNote{}
	for _, n := range r.doc.Notes {
		content, err := r.noteContent(n.Content)
		if err != nil {
			return err
		}
		category := ""
		if n.CategoryID != nil && known[*n.CategoryID] {
			category = *n.CategoryID
		}
		notesIn[category] = append(notesIn[category], notes.ImportNote{
			Title: n.Title, Content: content, CreatedAt: parseTime(n.CreatedAt), UpdatedAt: parseTime(n.UpdatedAt),
			Archived: n.Archived, Pinned: n.Pinned, SourceID: n.ID, Icon: r.icon(n.Icon), Color: n.Color,
		})
	}
	var build func(c export.Category) notes.ImportCategory
	build = func(c export.Category) notes.ImportCategory {
		ic := notes.ImportCategory{Name: c.Title, SourceID: c.ID, Icon: r.icon(c.Icon), Color: c.Color, Notes: notesIn[c.ID]}
		for _, sub := range children[c.ID] {
			ic.Categories = append(ic.Categories, build(sub))
		}
		return ic
	}
	root := notes.ImportCategory{Notes: notesIn[""]}
	for _, c := range children[""] {
		root.Categories = append(root.Categories, build(c))
	}
	imported, err := r.s.Notes.ImportTx(r.ctx, q, r.userID, r.opts.CategoryID, notes.ImportTree{Root: root, IDs: r.ids}, r.opts.SkipDuplicates)
	if errors.Is(err, notes.ErrInvalidCategory) {
		return ErrInvalidCategory
	}
	if err != nil {
		return invalid(err)
	}
	r.report.Notes = &NotesReport{Notes: imported.Notes, Categories: imported.Categories, DuplicateNotes: imported.DuplicateNotes}
	return nil
}

// noteContent rewrites the IDs in a note's content and drops the attachment
// blocks whose attachment isn't available.
func (r *run) noteContent(raw json.RawMessage) (notes.Doc, error) {
	rewritten, err := rewriteIDs(raw, r.ids)
	if err != nil {
		return notes.Doc{}, fmt.Errorf("%w: note content: %v", ErrInvalidFile, err)
	}
	var d notes.Doc
	if err := json.Unmarshal(rewritten, &d); err != nil {
		return notes.Doc{}, fmt.Errorf("%w: note content: %v", ErrInvalidFile, err)
	}
	blocks := d.Blocks[:0]
	for _, b := range d.Blocks {
		if b.Type == notes.BlockAttachment && !r.available[b.ID] {
			continue
		}
		blocks = append(blocks, b)
	}
	d.Blocks = blocks
	return d, nil
}

func (r *run) importMedia(q db.Querier) error {
	if !r.opts.Parts.Media {
		return nil
	}
	report := &MediaReport{}
	r.report.Media = report
	urls := map[string]bool{}
	rows, err := q.QueryContext(r.ctx, `SELECT url FROM media_sources WHERE user_id = ?`, r.userID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			_ = rows.Close()
			return err
		}
		urls[u] = true
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range r.doc.MediaSources {
		if urls[m.URL] {
			report.Duplicates++
			continue
		}
		id, err := r.s.Media.InsertTx(r.ctx, q, r.userID, media.Imported{Kind: m.Kind, Title: m.Title, URL: m.URL, Icon: r.icon(m.Icon)})
		if errors.Is(err, media.ErrInvalidKind) || errors.Is(err, media.ErrInvalidTitle) ||
			errors.Is(err, media.ErrInvalidURL) || errors.Is(err, media.ErrInvalidIcon) ||
			errors.Is(err, media.ErrEmbedUnsupported) {
			report.Invalid++
			continue
		}
		if err != nil {
			return err
		}
		urls[m.URL] = true
		r.ids[m.ID] = id
		report.Sources++
	}
	return nil
}

// importFeeds adds subscriptions and restores flags and listen records.
// Feeds and items are shared with other users, so nothing in the file can
// change them: see feeds.EnsureFeedTx and feeds.RestoreItemTx.
func (r *run) importFeeds(q db.Querier) error {
	if !r.opts.Parts.Feeds {
		return nil
	}
	report := &FeedsReport{}
	r.report.Feeds = report
	stored := make(map[string]string, len(r.doc.Feeds))
	for _, f := range r.doc.Feeds {
		feedID, err := r.s.Feeds.EnsureFeedTx(r.ctx, q, f.URL)
		if errors.Is(err, feeds.ErrInvalidURL) {
			report.Invalid++
			continue
		}
		if err != nil {
			return err
		}
		stored[f.ID] = feedID
		if !f.Subscribed {
			continue
		}
		id, added, err := r.s.Feeds.SubscribeTx(r.ctx, q, r.userID, feedID, f.CustomTitle, r.icon(f.Icon))
		if errors.Is(err, feeds.ErrInvalidIcon) {
			report.Invalid++
			continue
		}
		if err != nil {
			return err
		}
		r.ids[f.ID] = id
		if added {
			report.Subscriptions++
		} else {
			report.Duplicates++
		}
	}
	for _, it := range r.doc.FeedItems {
		feedID, ok := stored[it.FeedID]
		if !ok {
			report.SkippedItems++
			continue
		}
		item := feeds.Imported{Identity: it.GUID, Title: it.Title, Link: it.Link, Summary: it.Summary,
			PublishedAt: it.PublishedAt, ReadLaterAt: it.ReadLaterAt, LikedAt: it.LikedAt}
		if it.Audio != nil {
			item.Audio = &feeds.Audio{URL: it.Audio.URL, Type: it.Audio.Type, DurationSeconds: it.Audio.DurationSeconds}
		}
		if l := it.Listen; l != nil {
			item.Listen = &feeds.ImportedListen{PositionSeconds: l.PositionSeconds, DurationSeconds: l.DurationSeconds,
				StartedAt: l.StartedAt, UpdatedAt: l.UpdatedAt, FinishedAt: l.FinishedAt}
		}
		restored, err := r.s.Feeds.RestoreItemTx(r.ctx, q, r.userID, feedID, item)
		if err != nil {
			return err
		}
		if restored {
			report.RestoredItems++
		} else {
			report.SkippedItems++
		}
	}
	return nil
}

func (r *run) importDashboards(q db.Querier) error {
	if !r.opts.Parts.Dashboards {
		return nil
	}
	report := &DashboardsReport{}
	r.report.Dashboards = report
	var widgets func(list []*export.Widget) ([]dashboards.Widget, error)
	widgets = func(list []*export.Widget) ([]dashboards.Widget, error) {
		out := make([]dashboards.Widget, 0, len(list))
		for _, w := range list {
			children, err := widgets(w.Children)
			if err != nil {
				return nil, err
			}
			config, err := rewriteIDs(w.Config, r.ids)
			if err != nil {
				report.SkippedWidgets += 1 + len(w.Children)
				continue
			}
			converted, err := dashboards.StoredWidget(w.Kind, w.Width, config, children)
			if err != nil {
				report.SkippedWidgets += 1 + len(children)
				continue
			}
			out = append(out, converted)
		}
		return out, nil
	}
	for _, d := range r.doc.Dashboards {
		sections, err := widgets(d.Widgets)
		if err != nil {
			return err
		}
		cols := dashboards.Columns{DesktopColumns: d.DesktopColumns, MobileColumns: d.MobileColumns, MobileBreakpoint: d.MobileBreakpoint}
		created, skipped, err := r.s.Dashboards.ImportTx(r.ctx, q, r.userID, d.Title, d.Slug, cols, sections)
		if err != nil {
			return invalid(err)
		}
		r.ids[d.ID] = created.ID
		report.Dashboards++
		report.SkippedWidgets += skipped
	}
	return nil
}

// setDefaults sets the default folder and category from the file when the
// user has none and the file's default was imported or matched.
func (r *run) setDefaults(q db.Querier) error {
	set := func(column string, part bool, fileID *string) error {
		if !part || fileID == nil {
			return nil
		}
		id, ok := r.ids[*fileID]
		if !ok {
			return nil
		}
		_, err := q.ExecContext(r.ctx,
			`UPDATE users SET `+column+` = ? WHERE id = ? AND `+column+` IS NULL`, id, r.userID)
		return err
	}
	if err := set("share_folder_id", r.opts.Parts.Favourites, r.doc.Profile.DefaultFolderID); err != nil {
		return err
	}
	return set("note_category_id", r.opts.Parts.Notes, r.doc.Profile.DefaultNoteCategoryID)
}

// ParseParts reads a comma-separated list of parts; it reports false for an
// empty list or an unknown part.
func ParseParts(s string) (Parts, bool) {
	var p Parts
	if strings.TrimSpace(s) == "" {
		return p, false
	}
	for _, name := range strings.Split(s, ",") {
		switch strings.TrimSpace(name) {
		case "favourites":
			p.Favourites = true
		case "dashboards":
			p.Dashboards = true
		case "notes":
			p.Notes = true
		case "media":
			p.Media = true
		case "feeds":
			p.Feeds = true
		default:
			return Parts{}, false
		}
	}
	return p, true
}
