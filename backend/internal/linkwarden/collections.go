package linkwarden

import (
	"context"

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

// DeleteCollection is DELETE /api/v1/collections/{id}: the folder is deleted
// and its contents move to its parent, so no link is lost. It returns the
// collection as it was.
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
	if err := s.Favourites.Delete(ctx, sc.UserID, n.ID, favourites.DeleteMoveChildren); err != nil {
		return Collection{}, err
	}
	return c, nil
}
