// Package favourites stores each user's favourites tree (folders and links)
// and the tags on its links.
package favourites

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/jobs"
)

// Kind is the type of a node.
type Kind string

// Node kinds.
const (
	KindFolder Kind = "folder"
	KindLink   Kind = "link"
)

// Node is a folder or link as the API exposes it. The root folder has no title.
type Node struct {
	ID       string   `json:"id"`
	Kind     Kind     `json:"kind"`
	Title    *string  `json:"title"`
	URL      *string  `json:"url,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Children []*Node  `json:"children,omitempty"`
	// IconImageID is the favicon of a link's origin, when one is known. The
	// store doesn't set it; favicons are resolved by origin when serving.
	IconImageID *string `json:"iconImageId,omitempty"`
	// Description and PreviewImageID are only set for the tree's owner (Tree,
	// Create and Update), never by the loaders that widgets use.
	Description    *string `json:"description,omitempty"`
	PreviewImageID *string `json:"previewImageId,omitempty"`
	// CreatedAt is only set by Tree.
	CreatedAt *time.Time `json:"createdAt,omitempty"`
}

// Tag is a tag of a user.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Num is the integer ID for apps; CreatedAt is set by SubtreeNodes only.
	Num       int64     `json:"-"`
	CreatedAt time.Time `json:"-"`
}

// DeleteMode says what happens to the contents of a folder that is deleted.
type DeleteMode string

// Delete modes. DeleteKeepNone is only valid for links and empty folders.
const (
	DeleteKeepNone     DeleteMode = ""
	DeleteWithChildren DeleteMode = "delete"
	DeleteMoveChildren DeleteMode = "move"
)

// Errors returned by the store.
var (
	ErrNotFound           = errors.New("node not found")
	ErrInvalidTitle       = errors.New("invalid title")
	ErrInvalidURL         = errors.New("invalid url")
	ErrInvalidTag         = errors.New("invalid tag")
	ErrInvalidDescription = errors.New("invalid description")
	ErrInvalidKind        = errors.New("invalid kind")
	ErrInvalidField       = errors.New("field not allowed for this kind")
	ErrInvalidParent      = errors.New("parent is not a folder")
	ErrInvalidMove        = errors.New("cannot move a folder into itself")
	ErrFolderNotEmpty     = errors.New("folder is not empty")
	ErrRootImmutable      = errors.New("the root folder cannot be changed")
)

// MaxTagSuggestions is the most tags SuggestTags returns.
const MaxTagSuggestions = 10

// Store reads and writes favourites.
type Store struct {
	DB  *sql.DB
	Now func() time.Time
	// Jobs, when set, gets a favicon and a preview fetch for every new link
	// and every URL change, in the same transaction.
	Jobs *jobs.Queue
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB) *Store {
	return &Store{DB: database, Now: time.Now}
}

// NewNode is a node to create. An empty ParentID means the root folder.
// Folders need a Title and may not have a URL, tags or a description; links
// need a URL. The node is added at the end of its folder, or first when First
// is set.
type NewNode struct {
	ParentID    string
	Kind        Kind
	Title       *string
	URL         *string
	Tags        []string
	Description *string
	First       bool
}

// Patch changes a node. Nil fields stay unchanged. An empty link title or
// description clears it. URL, Tags, Description and RemovePreview are only
// allowed on links. Changing the URL or RemovePreview removes the link's
// preview image.
type Patch struct {
	Title         *string
	URL           *string
	Tags          *[]string
	Description   *string
	RemovePreview bool
}

// row is a stored node without its tags and children.
type row struct {
	id       string
	parentID sql.NullString
	kind     Kind
	position int
	title    sql.NullString
	url      sql.NullString
	// description and previewImageID are only set by load and Tree.
	description    sql.NullString
	previewImageID sql.NullString
}

// EnsureRoot returns the ID of the user's root folder, creating it first if
// the user has none. q may be a transaction.
func (s *Store) EnsureRoot(ctx context.Context, q db.Querier, userID string) (string, error) {
	now := db.FormatTime(s.Now())
	if _, err := q.ExecContext(ctx,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, created_at, updated_at)
		 VALUES (?, ?, NULL, 'folder', 0, NULL, NULL, ?, ?)
		 ON CONFLICT (user_id) WHERE parent_id IS NULL DO NOTHING`,
		uuid.NewString(), userID, now, now,
	); err != nil {
		return "", err
	}
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT id FROM favourite_nodes WHERE user_id = ? AND parent_id IS NULL`, userID,
	).Scan(&id)
	return id, err
}

// Tree returns the user's whole tree, starting at the root folder, with every
// folder's children in their order and the tags of every link.
func (s *Store) Tree(ctx context.Context, userID string) (*Node, error) {
	rootID, err := s.EnsureRoot(ctx, s.DB, userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, parent_id, kind, position, title, url, description, preview_image_id, created_at FROM favourite_nodes
		 WHERE user_id = ? ORDER BY parent_id, position`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodes := map[string]*Node{}
	var order []row
	for rows.Next() {
		var r row
		var created string
		if err := rows.Scan(&r.id, &r.parentID, &r.kind, &r.position, &r.title, &r.url, &r.description, &r.previewImageID, &created); err != nil {
			return nil, err
		}
		n := r.node()
		n.Description, n.PreviewImageID = nullable(r.description), nullable(r.previewImageID)
		createdAt, err := db.ParseTime(created)
		if err != nil {
			return nil, err
		}
		n.CreatedAt = &createdAt
		nodes[r.id] = n
		order = append(order, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, r := range order {
		if parent := nodes[r.parentID.String]; r.parentID.Valid && parent != nil {
			parent.Children = append(parent.Children, nodes[r.id])
		}
	}

	tagRows, err := s.DB.QueryContext(ctx,
		`SELECT nt.node_id, t.name FROM favourite_node_tags nt JOIN tags t ON t.id = nt.tag_id
		 WHERE t.user_id = ? ORDER BY t.name_key`, userID)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var nodeID, name string
		if err := tagRows.Scan(&nodeID, &name); err != nil {
			return nil, err
		}
		if n := nodes[nodeID]; n != nil {
			n.Tags = append(n.Tags, name)
		}
	}
	if err := tagRows.Err(); err != nil {
		return nil, err
	}
	return nodes[rootID], nil
}

// Subtree returns the user's folder folderID with everything inside it, in
// their order, without tags. It returns ErrNotFound when the user has no
// folder with that ID.
func (s *Store) Subtree(ctx context.Context, userID, folderID string) (*Node, error) {
	rows, err := s.DB.QueryContext(ctx,
		`WITH RECURSIVE sub (id, parent_id, kind, position, title, url, depth) AS (
			SELECT id, parent_id, kind, position, title, url, 0 FROM favourite_nodes
			WHERE id = ? AND user_id = ? AND kind = 'folder'
			UNION ALL
			SELECT n.id, n.parent_id, n.kind, n.position, n.title, n.url, sub.depth + 1
			FROM favourite_nodes n JOIN sub ON n.parent_id = sub.id
		 )
		 SELECT id, parent_id, kind, position, title, url FROM sub ORDER BY depth, position`,
		folderID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var root *Node
	nodes := map[string]*Node{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.parentID, &r.kind, &r.position, &r.title, &r.url); err != nil {
			return nil, err
		}
		n := r.node()
		nodes[r.id] = n
		if root == nil {
			root = n
		} else if parent := nodes[r.parentID.String]; parent != nil {
			parent.Children = append(parent.Children, n)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, ErrNotFound
	}
	return root, nil
}

// Link returns the user's link with the given ID, without tags. It returns
// ErrNotFound when the user has no link with that ID.
func (s *Store) Link(ctx context.Context, userID, id string) (*Node, error) {
	r, err := load(ctx, s.DB, userID, id)
	if err != nil {
		return nil, err
	}
	if r.kind != KindLink {
		return nil, ErrNotFound
	}
	return r.node(), nil
}

// Create adds a node at the end of its parent folder and returns it.
func (s *Store) Create(ctx context.Context, userID string, n NewNode) (*Node, error) {
	var title, link, description *string
	var tags []string
	switch n.Kind {
	case KindFolder:
		if n.URL != nil || len(n.Tags) > 0 || n.Description != nil {
			return nil, ErrInvalidField
		}
		if n.Title == nil {
			return nil, ErrInvalidTitle
		}
		name, err := NormalizeFolderName(*n.Title)
		if err != nil {
			return nil, err
		}
		title = &name
	case KindLink:
		if n.URL == nil {
			return nil, ErrInvalidURL
		}
		u, err := NormalizeURL(*n.URL)
		if err != nil {
			return nil, err
		}
		link = &u
		if title, err = NormalizeLinkTitle(n.Title); err != nil {
			return nil, err
		}
		if tags, err = NormalizeTags(n.Tags); err != nil {
			return nil, err
		}
		if description, err = NormalizeDescription(n.Description); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidKind
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	parentID := n.ParentID
	if parentID == "" {
		if parentID, err = s.EnsureRoot(ctx, tx, userID); err != nil {
			return nil, err
		}
	}
	parent, err := load(ctx, tx, userID, parentID)
	if err != nil {
		return nil, err
	}
	if parent.kind != KindFolder {
		return nil, ErrInvalidParent
	}
	var position int
	if n.First {
		if _, err := tx.ExecContext(ctx,
			`UPDATE favourite_nodes SET position = position + 1 WHERE parent_id = ?`, parentID,
		); err != nil {
			return nil, err
		}
	} else if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM favourite_nodes WHERE parent_id = ?`, parentID,
	).Scan(&position); err != nil {
		return nil, err
	}

	id := uuid.NewString()
	now := db.FormatTime(s.Now())
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO favourite_nodes (id, user_id, parent_id, kind, position, title, url, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, parentID, n.Kind, position, title, link, description, now, now,
	); err != nil {
		return nil, err
	}
	node := &Node{ID: id, Kind: n.Kind, Title: title, URL: link, Description: description}
	if n.Kind == KindLink {
		if err := s.setTags(ctx, tx, userID, id, tags); err != nil {
			return nil, err
		}
		if node.Tags, err = nodeTags(ctx, tx, id); err != nil {
			return nil, err
		}
		if err := s.queueFetches(ctx, tx, userID, id, *link); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.notifyFetches(n.Kind == KindLink)
	return node, nil
}

// queueFetches queues the favicon of the link's origin and the metadata of
// its page. It does nothing without a job queue.
func (s *Store) queueFetches(ctx context.Context, q db.Querier, userID, nodeID, rawURL string) error {
	if s.Jobs == nil {
		return nil
	}
	if origin, err := Origin(rawURL); err == nil {
		if err := s.Jobs.Enqueue(ctx, q, jobs.KindFavicon, origin, jobs.FaviconPayload{URL: rawURL}); err != nil {
			return err
		}
	}
	return s.Jobs.Enqueue(ctx, q, jobs.KindPreviewFetch, nodeID,
		jobs.PreviewFetchPayload{UserID: userID, NodeID: nodeID, URL: rawURL})
}

// notifyFetches wakes the fetch workers after a commit that queued jobs.
func (s *Store) notifyFetches(queued bool) {
	if queued && s.Jobs != nil {
		s.Jobs.Notify(jobs.KindFavicon, jobs.KindPreviewFetch)
	}
}

// Update changes the title, URL, tags, description or preview of a node and returns it without children.
func (s *Store) Update(ctx context.Context, userID, id string, p Patch) (*Node, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	r, err := load(ctx, tx, userID, id)
	if err != nil {
		return nil, err
	}
	if !r.parentID.Valid {
		return nil, ErrRootImmutable
	}
	if r.kind == KindFolder && (p.URL != nil || p.Tags != nil || p.Description != nil || p.RemovePreview) {
		return nil, ErrInvalidField
	}

	title := nullable(r.title)
	if p.Title != nil {
		if r.kind == KindFolder {
			name, err := NormalizeFolderName(*p.Title)
			if err != nil {
				return nil, err
			}
			title = &name
		} else if title, err = NormalizeLinkTitle(p.Title); err != nil {
			return nil, err
		}
	}
	link := nullable(r.url)
	if p.URL != nil {
		u, err := NormalizeURL(*p.URL)
		if err != nil {
			return nil, err
		}
		link = &u
	}
	var tags []string
	if p.Tags != nil {
		if tags, err = NormalizeTags(*p.Tags); err != nil {
			return nil, err
		}
	}
	description := nullable(r.description)
	if p.Description != nil {
		if description, err = NormalizeDescription(p.Description); err != nil {
			return nil, err
		}
	}
	urlChanged := link != nil && (!r.url.Valid || r.url.String != *link)
	preview := nullable(r.previewImageID)
	if urlChanged || p.RemovePreview {
		preview = nil
	}

	// A new URL is a new page: its preview is fetched again. A removed
	// preview isn't.
	if _, err := tx.ExecContext(ctx,
		`UPDATE favourite_nodes SET title = ?, url = ?, description = ?, preview_image_id = ?, updated_at = ?
		 WHERE id = ?`,
		title, link, description, preview, db.FormatTime(s.Now()), id,
	); err != nil {
		return nil, err
	}
	node := &Node{ID: id, Kind: r.kind, Title: title, URL: link, Description: description, PreviewImageID: preview}
	if r.kind == KindLink {
		if p.Tags != nil {
			if err := s.setTags(ctx, tx, userID, id, tags); err != nil {
				return nil, err
			}
			if err := removeUnusedTags(ctx, tx, userID); err != nil {
				return nil, err
			}
		}
		if node.Tags, err = nodeTags(ctx, tx, id); err != nil {
			return nil, err
		}
		if urlChanged {
			if err := s.queueFetches(ctx, tx, userID, id, *link); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.notifyFetches(urlChanged)
	return node, nil
}

// Move puts a node into the folder parentID at position, which is the index
// the node has among the folder's children after the move. The position is
// clamped to the valid range. Folders move with everything inside them.
func (s *Store) Move(ctx context.Context, userID, id, parentID string, position int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.move(ctx, tx, userID, id, parentID, position); err != nil {
		return err
	}
	return tx.Commit()
}

// move is Move inside the transaction tx.
func (s *Store) move(ctx context.Context, tx *sql.Tx, userID, id, parentID string, position int) error {
	r, err := load(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	if !r.parentID.Valid {
		return ErrRootImmutable
	}
	target, err := load(ctx, tx, userID, parentID)
	if err != nil {
		return err
	}
	if target.kind != KindFolder {
		return ErrInvalidParent
	}
	var cycle bool
	if err := tx.QueryRowContext(ctx,
		`WITH RECURSIVE ancestors (id, parent_id) AS (
			SELECT id, parent_id FROM favourite_nodes WHERE id = ?
			UNION ALL
			SELECT n.id, n.parent_id FROM favourite_nodes n JOIN ancestors a ON n.id = a.parent_id
		 )
		 SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = ?)`, parentID, id,
	).Scan(&cycle); err != nil {
		return err
	}
	if cycle {
		return ErrInvalidMove
	}

	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM favourite_nodes WHERE parent_id = ? AND id <> ?`, parentID, id,
	).Scan(&count); err != nil {
		return err
	}
	position = min(max(position, 0), count)

	// Close the gap in the old folder, then open one in the new folder.
	if err := shiftAfter(ctx, tx, r.parentID.String, r.position, -1); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE favourite_nodes SET position = position + 1 WHERE parent_id = ? AND id <> ? AND position >= ?`,
		parentID, id, position,
	); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE favourite_nodes SET parent_id = ?, position = ?, updated_at = ? WHERE id = ?`,
		parentID, position, db.FormatTime(s.Now()), id,
	)
	return err
}

// shiftAfter moves the children of folder parentID behind position by `by` places.
func shiftAfter(ctx context.Context, q db.Querier, parentID string, position, by int) error {
	_, err := q.ExecContext(ctx,
		`UPDATE favourite_nodes SET position = position + ? WHERE parent_id = ? AND position > ?`,
		by, parentID, position,
	)
	return err
}

// Delete removes a node. A folder with children needs DeleteWithChildren,
// which deletes everything inside it, or DeleteMoveChildren, which puts its
// children in its place in the parent folder, in their order.
func (s *Store) Delete(ctx context.Context, userID, id string, mode DeleteMode) error {
	if mode != DeleteKeepNone && mode != DeleteWithChildren && mode != DeleteMoveChildren {
		return ErrInvalidField
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	r, err := load(ctx, tx, userID, id)
	if err != nil {
		return err
	}
	if !r.parentID.Valid {
		return ErrRootImmutable
	}
	var children int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM favourite_nodes WHERE parent_id = ?`, id,
	).Scan(&children); err != nil {
		return err
	}
	if children > 0 && mode == DeleteKeepNone {
		return ErrFolderNotEmpty
	}
	if err := s.revokeAppTokens(ctx, tx, id, mode == DeleteWithChildren); err != nil {
		return err
	}

	// The node's slot is replaced by its children when they move up, otherwise it closes.
	shift := -1
	if mode == DeleteMoveChildren {
		shift = children - 1
	}
	if err := shiftAfter(ctx, tx, r.parentID.String, r.position, shift); err != nil {
		return err
	}
	if mode == DeleteMoveChildren {
		if _, err := tx.ExecContext(ctx,
			`UPDATE favourite_nodes SET parent_id = ?, position = position + ?, updated_at = ? WHERE parent_id = ?`,
			r.parentID.String, r.position, db.FormatTime(s.Now()), id,
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM favourite_nodes WHERE id = ?`, id); err != nil {
		return err
	}
	if err := removeUnusedTags(ctx, tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// revokeAppTokens revokes the active app tokens whose root folder is removed
// by deleting id: id itself, and with withChildren every folder below it.
// Their root_id is cleared by the foreign key when the nodes are deleted.
func (s *Store) revokeAppTokens(ctx context.Context, q db.Querier, id string, withChildren bool) error {
	now := db.FormatTime(s.Now())
	removed := `SELECT ?`
	if withChildren {
		removed = `WITH RECURSIVE sub (id) AS (
			SELECT ? UNION ALL SELECT n.id FROM favourite_nodes n JOIN sub ON n.parent_id = sub.id
		 ) SELECT id FROM sub`
	}
	_, err := q.ExecContext(ctx,
		`UPDATE app_tokens SET revoked_at = ?
		 WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?) AND root_id IN (`+removed+`)`,
		now, now, id)
	return err
}

// SuggestTags returns up to MaxTagSuggestions of the user's tags whose name
// starts with prefix, ignoring case, sorted by name.
func (s *Store) SuggestTags(ctx context.Context, userID, prefix string) ([]Tag, error) {
	pattern := likeEscaper.Replace(tagKey(strings.TrimSpace(prefix))) + "%"
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, name FROM tags WHERE user_id = ? AND name_key LIKE ? ESCAPE '\'
		 ORDER BY name_key LIMIT ?`, userID, pattern, MaxTagSuggestions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// setTags replaces the tags of a link with the given normalized names,
// creating tags that don't exist yet.
func (s *Store) setTags(ctx context.Context, q db.Querier, userID, nodeID string, names []string) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM favourite_node_tags WHERE node_id = ?`, nodeID); err != nil {
		return err
	}
	now := db.FormatTime(s.Now())
	for _, name := range names {
		key := tagKey(name)
		if _, err := q.ExecContext(ctx,
			`INSERT INTO tags (id, user_id, name, name_key, created_at) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (user_id, name_key) DO NOTHING`,
			uuid.NewString(), userID, name, key, now,
		); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO favourite_node_tags (node_id, tag_id)
			 SELECT ?, id FROM tags WHERE user_id = ? AND name_key = ?`, nodeID, userID, key,
		); err != nil {
			return err
		}
	}
	return nil
}

// removeUnusedTags deletes the user's tags that are on no link.
func removeUnusedTags(ctx context.Context, q db.Querier, userID string) error {
	_, err := q.ExecContext(ctx,
		`DELETE FROM tags WHERE user_id = ?
		 AND NOT EXISTS (SELECT 1 FROM favourite_node_tags WHERE tag_id = tags.id)`, userID)
	return err
}

// nodeTags returns the tag names of a link, sorted by name as in Tree.
func nodeTags(ctx context.Context, q db.Querier, nodeID string) ([]string, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT t.name FROM favourite_node_tags nt JOIN tags t ON t.id = nt.tag_id
		 WHERE nt.node_id = ? ORDER BY t.name_key`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, rows.Err()
}

// load returns the user's node with the given ID, or ErrNotFound.
func load(ctx context.Context, q db.Querier, userID, id string) (row, error) {
	var r row
	err := q.QueryRowContext(ctx,
		`SELECT id, parent_id, kind, position, title, url, description, preview_image_id FROM favourite_nodes WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&r.id, &r.parentID, &r.kind, &r.position, &r.title, &r.url, &r.description, &r.previewImageID)
	if errors.Is(err, sql.ErrNoRows) {
		return row{}, ErrNotFound
	}
	return r, err
}

func (r row) node() *Node {
	return &Node{ID: r.id, Kind: r.kind, Title: nullable(r.title), URL: nullable(r.url)}
}

func nullable(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}
