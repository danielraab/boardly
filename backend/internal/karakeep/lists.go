package karakeep

import (
	"context"
	"encoding/json"
	"math"

	"boardly/internal/favourites"
	"boardly/internal/trpc"
)

// listIcon is the icon of every list; Karakeep requires one and folders have none.
const listIcon = "📁"

// List is Karakeep's list shape for a folder.
type List struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      *string `json:"description"`
	Icon             string  `json:"icon"`
	ParentID         *string `json:"parentId"`
	Type             string  `json:"type"`
	Query            *string `json:"query"`
	Public           bool    `json:"public"`
	HasCollaborators bool    `json:"hasCollaborators"`
	UserRole         string  `json:"userRole"`
}

func (t *tree) asList(n *favourites.ScopedNode) List {
	l := List{ID: n.ID, Icon: listIcon, Type: "manual", UserRole: "owner"}
	if n.Title != nil {
		l.Name = *n.Title
	}
	if n.ParentID != t.scope.RootID {
		l.ParentID = &n.ParentID
	}
	return l
}

// folders returns the folders below the root, depth first in tree order.
func (t *tree) folders() []*favourites.ScopedNode {
	var out []*favourites.ScopedNode
	var visit func(id string)
	visit = func(id string) {
		for _, c := range t.children[id] {
			if c.Kind == favourites.KindFolder {
				out = append(out, c)
				visit(c.ID)
			}
		}
	}
	visit(t.scope.RootID)
	return out
}

// Lists is lists.list.
func (s *Service) Lists(ctx context.Context, sc Scope) ([]List, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return nil, err
	}
	lists := []List{}
	for _, f := range t.folders() {
		lists = append(lists, t.asList(f))
	}
	return lists, nil
}

// List is lists.get.
func (s *Service) List(ctx context.Context, sc Scope, id string) (List, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return List{}, err
	}
	f, err := t.list(id)
	if err != nil {
		return List{}, err
	}
	return t.asList(f), nil
}

// ListStats is lists.stats: the number of links directly in each list.
func (s *Service) ListStats(ctx context.Context, sc Scope) (trpc.Map, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return nil, err
	}
	stats := trpc.Map{}
	for _, f := range t.folders() {
		n := 0
		for _, c := range t.children[f.ID] {
			if c.Kind == favourites.KindLink {
				n++
			}
		}
		stats = append(stats, trpc.MapEntry{Key: f.ID, Value: n})
	}
	return stats, nil
}

// ListsOfBookmark is lists.getListsOfBookmark: the link's folder, unless it
// is the token's root folder.
func (s *Service) ListsOfBookmark(ctx context.Context, sc Scope, bookmarkID string) ([]List, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return nil, err
	}
	link, note, err := s.linkOrNote(ctx, sc, t, bookmarkID)
	if err != nil {
		return nil, err
	}
	if note != nil {
		return []List{}, nil
	}
	if f, err := t.list(link.ParentID); err == nil {
		return []List{t.asList(f)}, nil
	}
	return []List{}, nil
}

// CreateListInput is the input of lists.create.
type CreateListInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Icon        string  `json:"icon"`
	Type        string  `json:"type"`
	Query       *string `json:"query"`
	ParentID    *string `json:"parentId"`
}

// CreateList is lists.create: a folder at the end of the parent list, or of
// the token's root folder.
func (s *Service) CreateList(ctx context.Context, sc Scope, in CreateListInput) (List, error) {
	if err := checkUnsupported(map[string]bool{
		"smart lists":       (in.Type != "" && in.Type != "manual") || in.Query != nil,
		"list descriptions": nonEmpty(in.Description),
	}); err != nil {
		return List{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return List{}, err
	}
	parent, err := t.parent(in.ParentID)
	if err != nil {
		return List{}, err
	}
	node, err := s.Favourites.Create(ctx, sc.UserID, favourites.NewNode{ParentID: parent, Kind: favourites.KindFolder, Title: &in.Name})
	if err != nil {
		return List{}, err
	}
	return s.List(ctx, sc, node.ID)
}

// parent resolves a list's parent ID: nil or empty is the token's root folder.
func (t *tree) parent(id *string) (string, error) {
	if id == nil || *id == "" {
		return t.scope.RootID, nil
	}
	if _, err := t.list(*id); err != nil {
		return "", err
	}
	return *id, nil
}

// EditListInput is the input of lists.edit. ParentID is raw JSON to tell a
// missing parent from null (the token's root folder).
type EditListInput struct {
	ListID      string          `json:"listId"`
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Icon        *string         `json:"icon"`
	ParentID    json.RawMessage `json:"parentId"`
	Query       *string         `json:"query"`
	Public      *bool           `json:"public"`
}

// EditList is lists.edit: rename, and move to the end of another parent.
func (s *Service) EditList(ctx context.Context, sc Scope, in EditListInput) (List, error) {
	if err := checkUnsupported(map[string]bool{
		"smart lists":       in.Query != nil,
		"public lists":      isTrue(in.Public),
		"list descriptions": nonEmpty(in.Description),
	}); err != nil {
		return List{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return List{}, err
	}
	list, err := t.list(in.ListID)
	if err != nil {
		return List{}, err
	}
	parent := list.ParentID
	if in.ParentID != nil {
		var id *string
		if err := json.Unmarshal(in.ParentID, &id); err != nil {
			return List{}, &InputError{"Invalid parent list"}
		}
		if parent, err = t.parent(id); err != nil {
			return List{}, err
		}
	}
	if in.Name != nil {
		if _, err := s.Favourites.Update(ctx, sc.UserID, list.ID, favourites.Patch{Title: in.Name}); err != nil {
			return List{}, err
		}
	}
	if parent != list.ParentID {
		if err := s.Favourites.Move(ctx, sc.UserID, list.ID, parent, math.MaxInt); err != nil {
			return List{}, err
		}
	}
	return s.List(ctx, sc, list.ID)
}

// DeleteList is lists.delete: the folder's contents move to its parent.
func (s *Service) DeleteList(ctx context.Context, sc Scope, id string) error {
	t, err := s.load(ctx, sc)
	if err != nil {
		return err
	}
	if _, err := t.list(id); err != nil {
		return err
	}
	return s.Favourites.Delete(ctx, sc.UserID, id, favourites.DeleteMoveChildren)
}

// AddToList is lists.addToList: the link moves first into the list's folder.
func (s *Service) AddToList(ctx context.Context, sc Scope, listID, bookmarkID string) error {
	t, err := s.load(ctx, sc)
	if err != nil {
		return err
	}
	if _, err := t.list(listID); err != nil {
		return err
	}
	if _, note, err := s.linkOrNote(ctx, sc, t, bookmarkID); err != nil {
		return err
	} else if note != nil {
		return errNoteInList
	}
	return s.Favourites.Move(ctx, sc.UserID, bookmarkID, listID, 0)
}

// ErrNotInList is returned by RemoveFromList for a link that isn't directly
// in the list.
var ErrNotInList = &InputError{"The bookmark isn't in this list"}

// RemoveFromList is lists.removeFromList: the link moves first into the
// token's root folder.
func (s *Service) RemoveFromList(ctx context.Context, sc Scope, listID, bookmarkID string) error {
	t, err := s.load(ctx, sc)
	if err != nil {
		return err
	}
	if _, err := t.list(listID); err != nil {
		return err
	}
	link, note, err := s.linkOrNote(ctx, sc, t, bookmarkID)
	if err != nil {
		return err
	}
	if note != nil {
		return errNoteInList
	}
	if link.ParentID != listID {
		return ErrNotInList
	}
	return s.Favourites.Move(ctx, sc.UserID, bookmarkID, sc.RootID, 0)
}
