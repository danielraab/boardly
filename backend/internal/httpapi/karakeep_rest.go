package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"boardly/internal/apptokens"
	"boardly/internal/karakeep"
	"boardly/internal/trpc"
)

// karakeepV1 is the prefix of Karakeep's REST API, which sync clients such as
// Floccus call as <server address>/api/v1/….
const karakeepV1 = karakeepPrefix + "/api/v1"

// karakeepRESTv1 registers the Karakeep REST endpoints that Floccus' Karakeep
// adapter calls. Each behaves like the matching tRPC procedure on the token's
// subtree; other REST paths stay unimplemented.
func (a *api) karakeepRESTv1(mux *http.ServeMux) {
	k := a.karakeepService()
	handle := func(pattern string, h karakeepRESTHandler) {
		mux.Handle(pattern, karakeepREST(h.serve))
	}

	// Paths and methods not named here are not found (not 405), as for any
	// unimplemented part of Karakeep's API.
	mux.HandleFunc(karakeepV1+"/", http.NotFound)

	handle("POST "+karakeepV1+"/bookmarks", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in karakeep.CreateBookmarkInput
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		created, err := k.CreateBookmark(r.Context(), sc, in)
		if err != nil {
			return 0, nil, err
		}
		if created.AlreadyExists {
			return http.StatusOK, created, nil
		}
		return http.StatusCreated, created, nil
	})
	handle("GET "+karakeepV1+"/bookmarks/{id}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		b, err := k.GetBookmark(r.Context(), sc, r.PathValue("id"))
		return http.StatusOK, b, err
	})
	handle("PATCH "+karakeepV1+"/bookmarks/{id}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in karakeep.UpdateBookmarkInput
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		in.BookmarkID = r.PathValue("id")
		b, err := k.UpdateBookmark(r.Context(), sc, in)
		return http.StatusOK, b, err
	})
	handle("DELETE "+karakeepV1+"/bookmarks/{id}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		return http.StatusNoContent, nil, k.DeleteBookmark(r.Context(), sc, r.PathValue("id"))
	})
	handle("GET "+karakeepV1+"/bookmarks/{id}/lists", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		lists, err := k.ListsOfBookmark(r.Context(), sc, r.PathValue("id"))
		return http.StatusOK, map[string][]karakeep.List{"lists": lists}, err
	})
	type tagsBody struct {
		Tags []karakeep.TagRef `json:"tags"`
	}
	handle("POST "+karakeepV1+"/bookmarks/{id}/tags", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in tagsBody
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		res, err := k.UpdateTags(r.Context(), sc, karakeep.UpdateTagsInput{BookmarkID: r.PathValue("id"), Attach: in.Tags})
		return http.StatusOK, map[string][]string{"attached": res.Attached}, err
	})
	handle("DELETE "+karakeepV1+"/bookmarks/{id}/tags", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in tagsBody
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		res, err := k.UpdateTags(r.Context(), sc, karakeep.UpdateTagsInput{BookmarkID: r.PathValue("id"), Detach: in.Tags})
		return http.StatusOK, map[string][]string{"detached": res.Detached}, err
	})

	handle("GET "+karakeepV1+"/lists", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		lists, err := k.Lists(r.Context(), sc)
		return http.StatusOK, map[string][]karakeep.List{"lists": lists}, err
	})
	handle("POST "+karakeepV1+"/lists", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in karakeep.CreateListInput
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		list, err := k.CreateList(r.Context(), sc, in)
		return http.StatusCreated, list, err
	})
	handle("PATCH "+karakeepV1+"/lists/{id}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		var in karakeep.EditListInput
		if err := decodeKarakeepREST(r, &in); err != nil {
			return 0, nil, err
		}
		in.ListID = r.PathValue("id")
		list, err := k.EditList(r.Context(), sc, in)
		return http.StatusOK, list, err
	})
	handle("DELETE "+karakeepV1+"/lists/{id}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		return http.StatusNoContent, nil, k.DeleteList(r.Context(), sc, r.PathValue("id"))
	})
	handle("GET "+karakeepV1+"/lists/{id}/bookmarks", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		in, err := listBookmarksInput(r)
		if err != nil {
			return 0, nil, err
		}
		page, err := k.Bookmarks(r.Context(), sc, in)
		if err != nil {
			return 0, nil, err
		}
		next, err := encodeKarakeepCursor(page.NextCursor)
		return http.StatusOK, map[string]any{"bookmarks": page.Bookmarks, "nextCursor": next}, err
	})
	handle("PUT "+karakeepV1+"/lists/{id}/bookmarks/{bookmarkId}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		return http.StatusNoContent, nil, k.AddToList(r.Context(), sc, r.PathValue("id"), r.PathValue("bookmarkId"))
	})
	handle("DELETE "+karakeepV1+"/lists/{id}/bookmarks/{bookmarkId}", func(r *http.Request, sc karakeep.Scope) (int, any, error) {
		err := k.RemoveFromList(r.Context(), sc, r.PathValue("id"), r.PathValue("bookmarkId"))
		// Floccus moves a bookmark with the add to the new list and this
		// remove in parallel; when the add came first, there's nothing to do.
		if errors.Is(err, karakeep.ErrNotInList) {
			err = nil
		}
		return http.StatusNoContent, nil, err
	})
}

// karakeepRESTHandler handles a Karakeep REST request on the token's scope
// and returns the status and body (nil for none) of a success.
type karakeepRESTHandler func(r *http.Request, sc karakeep.Scope) (int, any, error)

func (h karakeepRESTHandler) serve(w http.ResponseWriter, r *http.Request, token apptokens.Token) error {
	// A token without a root folder can't authenticate.
	if token.RootID == nil {
		return &karakeepRESTError{http.StatusUnauthorized, "Unauthorized"}
	}
	status, body, err := h(r, karakeepScope(token))
	if err != nil {
		return karakeepRESTErrorOf(err)
	}
	if body == nil {
		w.WriteHeader(status)
		return nil
	}
	WriteJSON(w, status, body)
	return nil
}

// karakeepRESTErrorOf maps errors as the procedures do (karakeepError) and
// turns their tRPC codes into HTTP statuses; other errors pass through and
// become internal errors.
func karakeepRESTErrorOf(err error) error {
	var rest *karakeepRESTError
	if errors.As(err, &rest) {
		return rest
	}
	var te *trpc.Error
	if errors.As(karakeepError(err), &te) {
		return &karakeepRESTError{trpc.HTTPStatus(te.Code), te.Message}
	}
	return err
}

var errKarakeepJSON = &karakeepRESTError{http.StatusBadRequest, "The request body is not valid JSON"}

// decodeKarakeepREST decodes a JSON body; an empty body leaves v unchanged.
func decodeKarakeepREST(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return errKarakeepJSON
	}
	return nil
}

var errKarakeepCursor = &karakeepRESTError{http.StatusBadRequest, "Invalid cursor"}

// listBookmarksInput reads GET /lists/{id}/bookmarks: limit, sortOrder and
// the cursor of the previous page. includeContent is ignored.
func listBookmarksInput(r *http.Request) (karakeep.BookmarksInput, error) {
	q := r.URL.Query()
	in := karakeep.BookmarksInput{ListID: r.PathValue("id"), SortOrder: q.Get("sortOrder")}
	if l := q.Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil {
			return in, &karakeepRESTError{http.StatusBadRequest, "The limit must be between 1 and 100"}
		}
		in.Limit = n
	}
	if c := q.Get("cursor"); c != "" {
		raw, err := base64.RawURLEncoding.DecodeString(c)
		if err != nil {
			return in, errKarakeepCursor
		}
		var wrapped struct {
			Cursor json.RawMessage `json:"cursor"`
		}
		wrapped.Cursor = raw
		b, err := json.Marshal(wrapped)
		if err != nil {
			return in, errKarakeepCursor
		}
		if err := json.Unmarshal(b, &in); err != nil || in.Cursor == nil || in.Cursor.ID == "" {
			return in, errKarakeepCursor
		}
	}
	return in, nil
}

// encodeKarakeepCursor turns the procedures' cursor into an opaque string of
// URL-safe characters: Floccus appends it to the query without encoding it.
func encodeKarakeepCursor(c *karakeep.Cursor) (*string, error) {
	if c == nil {
		return nil, nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	return &s, nil
}
