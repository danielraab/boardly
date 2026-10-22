package favourites

import "context"

// Scope limits access to one folder of a user's tree and everything below it,
// as for requests made with an app token. Nodes outside it are treated as if
// they didn't exist, and its root folder can't be changed through it.
//
// The checks run before the store operation they guard, outside its
// transaction; callers check and then call Create, Update, Move or Delete.
type Scope struct {
	UserID string
	RootID string
}

// Contains reports whether the user's node id is the scope's root or below it.
func (s *Store) Contains(ctx context.Context, sc Scope, id string) (bool, error) {
	var ok bool
	err := s.DB.QueryRowContext(ctx,
		`WITH RECURSIVE up (id, parent_id) AS (
			SELECT id, parent_id FROM favourite_nodes WHERE id = ? AND user_id = ?
			UNION ALL
			SELECT n.id, n.parent_id FROM favourite_nodes n JOIN up ON n.id = up.parent_id
			WHERE up.id <> ?
		 )
		 SELECT EXISTS (SELECT 1 FROM up WHERE id = ?)`,
		id, sc.UserID, sc.RootID, sc.RootID,
	).Scan(&ok)
	return ok, err
}

// CheckCreate returns ErrNotFound when a node can't be created in parentID
// because the folder is outside the scope.
func (s *Store) CheckCreate(ctx context.Context, sc Scope, parentID string) error {
	return s.checkContains(ctx, sc, parentID)
}

// CheckChange returns ErrNotFound when id is outside the scope and
// ErrRootImmutable when it is the scope's root, which can't be renamed,
// moved or deleted through the scope.
func (s *Store) CheckChange(ctx context.Context, sc Scope, id string) error {
	if err := s.checkContains(ctx, sc, id); err != nil {
		return err
	}
	if id == sc.RootID {
		return ErrRootImmutable
	}
	return nil
}

// CheckMove returns an error when id can't be moved into parentID because
// either is outside the scope or id is the scope's root.
func (s *Store) CheckMove(ctx context.Context, sc Scope, id, parentID string) error {
	if err := s.CheckChange(ctx, sc, id); err != nil {
		return err
	}
	return s.checkContains(ctx, sc, parentID)
}

func (s *Store) checkContains(ctx context.Context, sc Scope, id string) error {
	ok, err := s.Contains(ctx, sc, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
