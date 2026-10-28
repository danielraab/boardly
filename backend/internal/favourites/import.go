package favourites

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"boardly/internal/db"
)

// ImportTree is the content of an import file, as a parser of the importer
// package reads it, with the counts of what the parser left out.
type ImportTree struct {
	Root   ImportFolder
	Report ImportReport
}

// ImportFolder is a folder of an import file. The name of the tree's root is
// ignored; its items go into the import folder.
type ImportFolder struct {
	Name  string
	Items []ImportItem
}

// ImportItem is either a folder or a link, in the order of the file.
type ImportItem struct {
	Folder *ImportFolder
	Link   *ImportLink
}

// ImportLink is a link of an import file. A zero CreatedAt means the import time.
type ImportLink struct {
	URL         string
	Title       *string
	Tags        []string
	Description *string
	CreatedAt   time.Time
}

// ImportReport counts what an import added and left out.
type ImportReport struct {
	Links                 int           `json:"links"`
	Folders               int           `json:"folders"`
	Skipped               ImportSkipped `json:"skipped"`
	ShortenedDescriptions int           `json:"shortenedDescriptions"`
}

// ImportSkipped counts the entries an import left out.
type ImportSkipped struct {
	Duplicates  int `json:"duplicates"`
	InvalidURLs int `json:"invalidUrls"`
	NotLinks    int `json:"notLinks"`
	SmartLists  int `json:"smartLists"`
}

// importer is the state of one Import call.
type importer struct {
	s      *Store
	q      db.Querier
	userID string
	now    time.Time
	// urls holds the comparable URLs of the user's links, including those
	// imported so far, or is nil when duplicates are imported.
	urls   map[string]bool
	report ImportReport
}

// Import adds the tree's folders and links below the user's folder folderID
// (the root is allowed) in one transaction; see ImportTx.
func (s *Store) Import(ctx context.Context, userID, folderID string, tree ImportTree, skipDuplicates bool) (ImportReport, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return ImportReport{}, err
	}
	defer func() { _ = tx.Rollback() }()
	report, err := s.ImportTx(ctx, tx, userID, folderID, tree, skipDuplicates)
	if err != nil {
		return ImportReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportReport{}, err
	}
	s.AfterImport(report)
	return report, nil
}

// ImportTx adds the tree's folders and links below the user's folder
// folderID (the root is allowed) in the transaction q, and returns the tree's
// report with the imported links and folders and the skipped duplicates
// added. A folder whose name equals an existing child folder's, ignoring
// case, is reused. With skipDuplicates, links whose URL is already in the
// user's tree are skipped. Imported nodes go after the existing children, in
// the order of the tree. After the commit, call AfterImport.
func (s *Store) ImportTx(ctx context.Context, q db.Querier, userID, folderID string, tree ImportTree, skipDuplicates bool) (ImportReport, error) {
	if folderID == "" {
		return ImportReport{}, ErrNotFound
	}
	folder, err := load(ctx, q, userID, folderID)
	if err != nil {
		return ImportReport{}, err
	}
	if folder.kind != KindFolder {
		return ImportReport{}, ErrInvalidParent
	}

	im := &importer{s: s, q: q, userID: userID, now: s.Now(), report: tree.Report}
	if skipDuplicates {
		if im.urls, err = linkURLs(ctx, q, userID); err != nil {
			return ImportReport{}, err
		}
	}
	if err := im.add(ctx, folderID, tree.Root.Items); err != nil {
		return ImportReport{}, err
	}
	return im.report, nil
}

// AfterImport starts the fetches of the links of a committed import.
func (s *Store) AfterImport(report ImportReport) {
	s.notifyFetches(report.Links > 0)
}

// linkURLs returns the comparable URLs of the user's links.
func linkURLs(ctx context.Context, q db.Querier, userID string) (map[string]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT url FROM favourite_nodes WHERE user_id = ? AND kind = 'link'`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	urls := map[string]bool{}
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		urls[comparableURL(u)] = true
	}
	return urls, rows.Err()
}

// add appends the items to the folder parentID, reusing its child folders.
func (im *importer) add(ctx context.Context, parentID string, items []ImportItem) error {
	folders := map[string]string{}
	rows, err := im.q.QueryContext(ctx,
		`SELECT id, title FROM favourite_nodes WHERE parent_id = ? AND kind = 'folder'`, parentID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			_ = rows.Close()
			return err
		}
		if _, ok := folders[folderKey(title)]; !ok {
			folders[folderKey(title)] = id
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var position int
	if err := im.q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM favourite_nodes WHERE parent_id = ?`, parentID,
	).Scan(&position); err != nil {
		return err
	}

	now := db.FormatTime(im.now)
	for _, item := range items {
		switch {
		case item.Folder != nil:
			name, err := NormalizeFolderName(item.Folder.Name)
			if err != nil {
				return err
			}
			id, ok := folders[folderKey(name)]
			if !ok {
				id = uuid.NewString()
				if _, err := im.q.ExecContext(ctx,
					`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, created_at, updated_at)
					 VALUES (?, ?, ?, 'folder', ?, ?, ?, ?)`,
					id, im.userID, parentID, position, name, now, now,
				); err != nil {
					return err
				}
				position++
				folders[folderKey(name)] = id
				im.report.Folders++
			}
			if err := im.add(ctx, id, item.Folder.Items); err != nil {
				return err
			}
		case item.Link != nil:
			inserted, err := im.addLink(ctx, parentID, position, item.Link)
			if err != nil {
				return err
			}
			if inserted {
				position++
			}
		}
	}
	return nil
}

// addLink inserts the link at the position, unless its URL is invalid or a
// skipped duplicate, and reports whether it did.
func (im *importer) addLink(ctx context.Context, parentID string, position int, l *ImportLink) (bool, error) {
	link, err := NormalizeURL(l.URL)
	if err != nil {
		im.report.Skipped.InvalidURLs++
		return false, nil
	}
	key := comparableURL(link)
	if im.urls[key] {
		im.report.Skipped.Duplicates++
		return false, nil
	}
	title, err := NormalizeLinkTitle(l.Title)
	if err != nil {
		return false, err
	}
	description, err := NormalizeDescription(l.Description)
	if err != nil {
		return false, err
	}
	tags, err := NormalizeTags(l.Tags)
	if err != nil {
		return false, err
	}
	created := l.CreatedAt
	if created.IsZero() {
		created = im.now
	}

	id := uuid.NewString()
	if _, err := im.q.ExecContext(ctx,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, description, created_at, updated_at)
		 VALUES (?, ?, ?, 'link', ?, ?, ?, ?, ?, ?)`,
		id, im.userID, parentID, position, title, link, description, db.FormatTime(created), db.FormatTime(im.now),
	); err != nil {
		return false, err
	}
	if err := im.s.setTags(ctx, im.q, im.userID, id, tags); err != nil {
		return false, err
	}
	if err := im.s.queueFetches(ctx, im.q, im.userID, id, link); err != nil {
		return false, err
	}
	if im.urls != nil {
		im.urls[key] = true
	}
	im.report.Links++
	return true, nil
}

// folderKey is the case-insensitive key of an already trimmed folder name.
func folderKey(name string) string {
	return strings.ToLower(name)
}
