package httpapi

import (
	"context"
	"errors"
	"net/http"

	"boardly/internal/favourites"
)

// favouritesError maps store errors to API errors; other errors pass through.
func favouritesError(err error) error {
	switch {
	case errors.Is(err, favourites.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Favourite not found")
	case errors.Is(err, favourites.ErrInvalidTitle):
		return NewError(http.StatusBadRequest, "invalid_title", "Folder names need 1 to 100 characters, link titles at most 1000")
	case errors.Is(err, favourites.ErrInvalidURL):
		return NewError(http.StatusBadRequest, "invalid_url", "The URL must be an http or https address of at most 2048 characters")
	case errors.Is(err, favourites.ErrInvalidTag):
		return NewError(http.StatusBadRequest, "invalid_tag", "Tag names need 1 to 100 characters")
	case errors.Is(err, favourites.ErrInvalidDescription):
		return NewError(http.StatusBadRequest, "invalid_description", "Descriptions can have at most 2048 characters")
	case errors.Is(err, favourites.ErrInvalidKind):
		return NewError(http.StatusBadRequest, "invalid_kind", "The kind must be folder or link")
	case errors.Is(err, favourites.ErrInvalidField):
		return NewError(http.StatusBadRequest, "invalid_field", "Folders can't have a URL or tags")
	case errors.Is(err, favourites.ErrInvalidParent):
		return NewError(http.StatusBadRequest, "invalid_parent", "The parent must be a folder")
	case errors.Is(err, favourites.ErrInvalidMove):
		return NewError(http.StatusBadRequest, "invalid_move", "A folder can't be moved into itself")
	case errors.Is(err, favourites.ErrFolderNotEmpty):
		return NewError(http.StatusBadRequest, "folder_not_empty", "Choose whether to delete or keep the folder's contents")
	case errors.Is(err, favourites.ErrRootImmutable):
		return NewError(http.StatusBadRequest, "root_immutable", "The root folder can't be changed")
	}
	return err
}

// withIcons sets the known favicon on every link in the trees.
func (a *api) withIcons(ctx context.Context, roots ...*favourites.Node) error {
	if a.favicons == nil {
		return nil
	}
	return a.favicons.SetIcons(ctx, roots...)
}

func (a *api) getFavourites(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	root, err := a.favourites.Tree(r.Context(), user.ID)
	if err != nil {
		return err
	}
	if err := a.withIcons(r.Context(), root); err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, root)
	return nil
}

func (a *api) createFavourite(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		ParentID string          `json:"parentId"`
		Kind     favourites.Kind `json:"kind"`
		Title    *string         `json:"title"`
		URL      *string         `json:"url"`
		Tags     []string        `json:"tags"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	node, err := a.favourites.Create(r.Context(), user.ID, favourites.NewNode{
		ParentID: body.ParentID, Kind: body.Kind, Title: body.Title, URL: body.URL, Tags: body.Tags,
	})
	if err != nil {
		return favouritesError(err)
	}
	if err := a.withIcons(r.Context(), node); err != nil {
		return err
	}
	WriteJSON(w, http.StatusCreated, node)
	return nil
}

func (a *api) updateFavourite(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Title *string   `json:"title"`
		URL   *string   `json:"url"`
		Tags  *[]string `json:"tags"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	node, err := a.favourites.Update(r.Context(), user.ID, r.PathValue("id"), favourites.Patch{Title: body.Title, URL: body.URL, Tags: body.Tags})
	if err != nil {
		return favouritesError(err)
	}
	if err := a.withIcons(r.Context(), node); err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, node)
	return nil
}

func (a *api) moveFavourite(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		ParentID string `json:"parentId"`
		Position int    `json:"position"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	if err := a.favourites.Move(r.Context(), user.ID, r.PathValue("id"), body.ParentID, body.Position); err != nil {
		return favouritesError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) deleteFavourite(w http.ResponseWriter, r *http.Request) error {
	mode := favourites.DeleteMode(r.URL.Query().Get("children"))
	if mode != favourites.DeleteKeepNone && mode != favourites.DeleteWithChildren && mode != favourites.DeleteMoveChildren {
		return NewError(http.StatusBadRequest, "invalid_request", "children must be delete or move")
	}
	user, _ := CurrentUser(r.Context())
	if err := a.favourites.Delete(r.Context(), user.ID, r.PathValue("id"), mode); err != nil {
		return favouritesError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) suggestTags(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	tags, err := a.favourites.SuggestTags(r.Context(), user.ID, r.URL.Query().Get("q"))
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, tags)
	return nil
}
