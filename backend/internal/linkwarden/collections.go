package linkwarden

import (
	"context"
	"encoding/json"
	"math"

	"boardly/internal/favourites"
)

// Collections is GET /api/v1/collections: every folder of the subtree.
func (s *Service) Collections(ctx context.Context, sc favourites.Scope) ([]Collection, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return nil, err
	}
	folders := t.folders()
	out := make([]Collection, len(folders))
	for i, f := range folders {
		out[i] = t.fullCollection(f)
	}
	return out, nil
}

// collectionByID loads the subtree and returns the collection with the given
// node ID.
func (s *Service) collectionByID(ctx context.Context, sc favourites.Scope, id string) (Collection, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Collection{}, err
	}
	n, ok := t.byID[id]
	if !ok || n.Kind != favourites.KindFolder {
		return Collection{}, ErrCollectionNotFound
	}
	return t.fullCollection(n), nil
}

// Collection is GET /api/v1/collections/{id}.
func (s *Service) Collection(ctx context.Context, sc favourites.Scope, num int64) (Collection, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Collection{}, err
	}
	n, err := t.collection(num)
	if err != nil {
		return Collection{}, err
	}
	return t.fullCollection(n), nil
}

// CreateCollectionInput is the body of POST /api/v1/collections (Linkwarden's
// PostCollectionSchema).
type CreateCollectionInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	IconWeight  *string `json:"iconWeight"`
	ParentID    *int64  `json:"parentId"`
}

// CreateCollection is POST /api/v1/collections: a folder at the end of the
// parent collection, or of the token's root folder.
func (s *Service) CreateCollection(ctx context.Context, sc favourites.Scope, in CreateCollectionInput) (Collection, error) {
	if err := checkUnsupported(map[string]bool{"collection descriptions": nonEmpty(in.Description)}); err != nil {
		return Collection{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return Collection{}, err
	}
	parent := sc.RootID
	if in.ParentID != nil {
		folder, err := t.collection(*in.ParentID)
		if err != nil {
			return Collection{}, errNotAccessible
		}
		parent = folder.ID
	}
	node, err := s.Favourites.Create(ctx, sc.UserID, favourites.NewNode{ParentID: parent, Kind: favourites.KindFolder, Title: &in.Name})
	if err != nil {
		return Collection{}, err
	}
	return s.collectionByID(ctx, sc, node.ID)
}

// EditCollectionInput is the body of PUT /api/v1/collections/{id}: a
// collection as GET returns it, with the fields to change. ParentID and
// Members are raw JSON to tell a missing value from null or empty. The other
// fields of the collection shape are ignored.
type EditCollectionInput struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	IsPublic    *bool           `json:"isPublic"`
	Members     json.RawMessage `json:"members"`
	ParentID    json.RawMessage `json:"parentId"`
}

// EditCollection is PUT /api/v1/collections/{id}: rename, and move to the end
// of another parent collection (null is the token's root collection). Sync
// clients such as Floccus send back the whole collection they fetched.
func (s *Service) EditCollection(ctx context.Context, sc favourites.Scope, num int64, in EditCollectionInput) (Collection, error) {
	var members []json.RawMessage
	if len(in.Members) > 0 && string(in.Members) != "null" {
		if err := json.Unmarshal(in.Members, &members); err != nil {
			return Collection{}, &InputError{"Invalid members."}
		}
	}
	if err := checkUnsupported(map[string]bool{
		"collection descriptions": nonEmpty(in.Description),
		"public collections":      in.IsPublic != nil && *in.IsPublic,
		"collection members":      len(members) > 0,
	}); err != nil {
		return Collection{}, err
	}
	t, err := s.load(ctx, sc)
	if err != nil {
		return Collection{}, err
	}
	n, err := t.collection(num)
	if err != nil {
		return Collection{}, err
	}
	rename := in.Name != nil && *in.Name != "" && *in.Name != folderName(n)
	parent := n.ParentID
	if len(in.ParentID) > 0 {
		var id *int64
		if err := json.Unmarshal(in.ParentID, &id); err != nil {
			return Collection{}, &InputError{"Invalid parentId."}
		}
		switch {
		case n.ID == sc.RootID:
			// The root collection's parent is null; any other parent is a move.
			if id != nil {
				return Collection{}, ErrRootCollectionEdit
			}
		case id == nil:
			parent = sc.RootID
		default:
			folder, err := t.collection(*id)
			if err != nil {
				return Collection{}, errNotAccessible
			}
			parent = folder.ID
		}
	}
	if n.ID == sc.RootID && rename {
		return Collection{}, ErrRootCollectionEdit
	}
	if rename {
		if _, err := s.Favourites.Update(ctx, sc.UserID, n.ID, favourites.Patch{Title: in.Name}); err != nil {
			return Collection{}, err
		}
	}
	if parent != n.ParentID {
		if err := s.Favourites.Move(ctx, sc.UserID, n.ID, parent, math.MaxInt); err != nil {
			return Collection{}, err
		}
	}
	return s.collectionByID(ctx, sc, n.ID)
}

// DeleteCollection is DELETE /api/v1/collections/{id}: the folder is deleted
// with its links and subfolders, as in Linkwarden; sync clients such as
// Floccus expect the links to be gone. It returns the collection as it was.
func (s *Service) DeleteCollection(ctx context.Context, sc favourites.Scope, num int64) (Collection, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Collection{}, err
	}
	n, err := t.collection(num)
	if err != nil {
		return Collection{}, err
	}
	if n.ID == sc.RootID {
		return Collection{}, ErrRootCollection
	}
	c := t.fullCollection(n)
	if err := s.Favourites.Delete(ctx, sc.UserID, n.ID, favourites.DeleteWithChildren); err != nil {
		return Collection{}, err
	}
	return c, nil
}
