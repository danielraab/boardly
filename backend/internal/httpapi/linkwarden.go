package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"famodash/internal/favourites"
	"famodash/internal/images"
	"famodash/internal/linkwarden"
	"famodash/internal/users"
)

// linkwardenPrefix is the server address that Linkwarden clients are given;
// they append Linkwarden's own paths (api/v1/links, api/v2/dashboard, …).
const linkwardenPrefix = "/api/linkwarden"

// linkwardenFailure is a Linkwarden error response: {"response": message}.
type linkwardenFailure struct {
	status  int
	message string
}

func (f *linkwardenFailure) Error() string { return f.message }

var errLinkwardenUnauthorized = &linkwardenFailure{http.StatusUnauthorized, "You must be logged in."}

// linkwardenHandler handles a request with a token, given as its scope.
type linkwardenHandler func(w http.ResponseWriter, r *http.Request, sc favourites.Scope) error

// linkwardenAPI serves what the Linkwarden mobile app and browser extension
// call. Every path that isn't listed answers 404 in Linkwarden's format.
func (a *api) linkwardenAPI() *http.ServeMux {
	var icons linkwarden.Favicons = noFavicons{}
	if a.favicons != nil {
		icons = a.favicons
	}
	lw := &linkwarden.Service{Favourites: a.favourites, Favicons: icons}
	v1 := linkwardenPrefix + "/api/v1"
	mux := http.NewServeMux()
	// respond registers a handler whose result is written as {"response": v},
	// or as {"data": v, …} when it is wrapped by data.
	respond := func(pattern string, h func(r *http.Request, sc favourites.Scope) (any, error)) {
		mux.Handle(pattern, a.linkwardenScoped(func(w http.ResponseWriter, r *http.Request, sc favourites.Scope) error {
			v, err := h(r, sc)
			if err != nil {
				return err
			}
			if d, ok := v.(linkwardenData); ok {
				WriteJSON(w, http.StatusOK, map[string]any{"data": d.v, "message": "Success", "success": true})
			} else {
				writeLinkwarden(w, v)
			}
			return nil
		}))
	}
	public := func(pattern string, h func(w http.ResponseWriter, r *http.Request) error) {
		mux.Handle(pattern, linkwardenServe(h))
	}

	public("GET "+v1+"/config", func(w http.ResponseWriter, _ *http.Request) error {
		writeLinkwarden(w, linkwarden.Config())
		return nil
	})
	public("GET "+v1+"/logins", func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, map[string][]struct{}{"buttonAuths": {}})
		return nil
	})
	public("POST "+v1+"/session", func(http.ResponseWriter, *http.Request) error {
		return &linkwardenFailure{http.StatusUnauthorized, "Famodash has no passwords. Log in with an access token from Settings → Connected apps."}
	})
	respond("GET "+v1+"/users/me", func(r *http.Request, _ favourites.Scope) (any, error) {
		token, _ := appToken(r.Context())
		user, err := a.users.Get(r.Context(), token.UserID)
		if errors.Is(err, users.ErrNotFound) {
			return nil, errLinkwardenUnauthorized
		}
		if err != nil {
			return nil, err
		}
		return linkwarden.Me(user), nil
	})
	respond("GET "+linkwardenPrefix+"/api/v2/dashboard", func(r *http.Request, sc favourites.Scope) (any, error) {
		return data(lw.Dashboard(r.Context(), sc))
	})

	respond("GET "+v1+"/search", func(r *http.Request, sc favourites.Scope) (any, error) {
		q := r.URL.Query()
		return data(lw.Search(r.Context(), sc, linkwarden.SearchInput{
			Sort: int(queryInt(r, "sort")), Cursor: queryInt(r, "cursor"),
			CollectionID: queryInt(r, "collectionId"), TagID: queryInt(r, "tagId"),
			PinnedOnly: q.Get("pinnedOnly") == "true", Query: q.Get("searchQueryString"),
		}))
	})
	respond("GET "+v1+"/links/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrLinkNotFound)
		if err != nil {
			return nil, err
		}
		return lw.Link(r.Context(), sc, num)
	})
	respond("POST "+v1+"/links", func(r *http.Request, sc favourites.Scope) (any, error) {
		var in linkwarden.CreateLinkInput
		if err := decodeLinkwarden(r, &in); err != nil {
			return nil, err
		}
		return lw.CreateLink(r.Context(), sc, in)
	})
	respond("PUT "+v1+"/links/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrLinkNotFound)
		if err != nil {
			return nil, err
		}
		var in linkwarden.UpdateLinkInput
		if err := decodeLinkwarden(r, &in); err != nil {
			return nil, err
		}
		return lw.UpdateLink(r.Context(), sc, num, in)
	})
	respond("DELETE "+v1+"/links/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrLinkNotFound)
		if err != nil {
			return nil, err
		}
		return lw.DeleteLink(r.Context(), sc, num)
	})

	respond("GET "+v1+"/collections", func(r *http.Request, sc favourites.Scope) (any, error) {
		return lw.Collections(r.Context(), sc)
	})
	respond("POST "+v1+"/collections", func(r *http.Request, sc favourites.Scope) (any, error) {
		var in linkwarden.CreateCollectionInput
		if err := decodeLinkwarden(r, &in); err != nil {
			return nil, err
		}
		return lw.CreateCollection(r.Context(), sc, in)
	})
	respond("DELETE "+v1+"/collections/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrCollectionNotFound)
		if err != nil {
			return nil, err
		}
		return lw.DeleteCollection(r.Context(), sc, num)
	})

	respond("GET "+v1+"/tags", func(r *http.Request, sc favourites.Scope) (any, error) {
		return data(lw.Tags(r.Context(), sc, linkwarden.TagsInput{
			Sort: int(queryInt(r, "sort")), Cursor: queryInt(r, "cursor"), Search: r.URL.Query().Get("search"),
		}))
	})
	respond("POST "+v1+"/tags", func(*http.Request, favourites.Scope) (any, error) {
		return nil, lw.CreateTag()
	})
	respond("GET "+v1+"/tags/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrTagNotFound)
		if err != nil {
			return nil, err
		}
		return lw.Tag(r.Context(), sc, num)
	})
	respond("DELETE "+v1+"/tags/{id}", func(r *http.Request, sc favourites.Scope) (any, error) {
		num, err := pathNum(r, linkwarden.ErrTagNotFound)
		if err != nil {
			return nil, err
		}
		return lw.DeleteTag(r.Context(), sc, num)
	})

	mux.Handle("GET "+v1+"/archives/{id}", a.linkwardenScoped(func(w http.ResponseWriter, r *http.Request, sc favourites.Scope) error {
		num, err := pathNum(r, linkwarden.ErrLinkNotFound)
		if err != nil {
			return err
		}
		return a.linkwardenPreview(w, r, lw, sc, num)
	}))
	respond("POST "+v1+"/archives/{id}", func(*http.Request, favourites.Scope) (any, error) {
		return nil, &linkwardenFailure{http.StatusBadRequest, "Famodash doesn't store screenshots or files. The link was saved without it."}
	})

	mux.Handle(linkwardenPrefix+"/", linkwardenServe(func(http.ResponseWriter, *http.Request) error {
		return &linkwardenFailure{http.StatusNotFound, "Not found."}
	}))
	return mux
}

// linkwardenPreview serves a link's favicon as its preview image.
func (a *api) linkwardenPreview(w http.ResponseWriter, r *http.Request, lw *linkwarden.Service, sc favourites.Scope, num int64) error {
	format := r.URL.Query().Get("format")
	if r.URL.Query().Get("preview") == "" || (format != "0" && format != "1") || a.images == nil {
		return linkwarden.ErrPreviewNotFound
	}
	icon, err := lw.Preview(r.Context(), sc, num)
	if err != nil {
		return err
	}
	img, f, err := a.images.Open(r.Context(), icon.ID)
	if errors.Is(err, images.ErrNotFound) {
		return linkwarden.ErrPreviewNotFound
	}
	if err != nil {
		return err
	}
	defer f.Close()
	h := w.Header()
	h.Set("Content-Type", img.ContentType)
	h.Set("Content-Length", strconv.FormatInt(img.Size, 10))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "private, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
	return nil
}

// linkwardenScoped runs h with the token's scope, or answers 401 without a
// token.
func (a *api) linkwardenScoped(h linkwardenHandler) http.Handler {
	return linkwardenServe(func(w http.ResponseWriter, r *http.Request) error {
		token, ok := appToken(r.Context())
		if !ok || token.RootID == nil {
			return errLinkwardenUnauthorized
		}
		return h(w, r, favourites.Scope{UserID: token.UserID, RootID: *token.RootID})
	})
}

// pathNum parses the integer ID in the path; an invalid one is notFound.
func pathNum(r *http.Request, notFound error) (int64, error) {
	num, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || num <= 0 {
		return 0, notFound
	}
	return num, nil
}

// queryInt reads an integer query parameter leniently, as Linkwarden's
// Number(): anything that isn't a number is 0, meaning "not set".
func queryInt(r *http.Request, name string) int64 {
	n, err := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func decodeLinkwarden(r *http.Request, v any) error {
	if err := decodeJSON(r, v); err != nil {
		return &linkwardenFailure{http.StatusBadRequest, "The request body is not valid JSON."}
	}
	return nil
}

// writeLinkwarden writes {"response": v}.
func writeLinkwarden(w http.ResponseWriter, v any) {
	WriteJSON(w, http.StatusOK, map[string]any{"response": v})
}

// linkwardenData marks a result that is written as {"data": v, …}, the
// envelope of search, tags and the dashboard.
type linkwardenData struct{ v any }

func data[T any](v T, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return linkwardenData{v}, nil
}

// linkwardenServe runs h and writes its error in Linkwarden's format.
func linkwardenServe(h func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		f := linkwardenErrorOf(err)
		if f == nil {
			slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
			f = &linkwardenFailure{http.StatusInternalServerError, "Internal server error."}
		}
		WriteJSON(w, f.status, map[string]string{"response": f.message})
	})
}

// linkwardenErrorOf maps the errors of the Linkwarden service and the
// favourites store to responses, or returns nil for an internal error.
func linkwardenErrorOf(err error) *linkwardenFailure {
	var failure *linkwardenFailure
	var input *linkwarden.InputError
	switch {
	case errors.As(err, &failure):
		return failure
	case errors.Is(err, linkwarden.ErrLinkNotFound), errors.Is(err, linkwarden.ErrCollectionNotFound),
		errors.Is(err, linkwarden.ErrTagNotFound), errors.Is(err, linkwarden.ErrPreviewNotFound):
		return &linkwardenFailure{http.StatusNotFound, err.Error()}
	case errors.Is(err, favourites.ErrNotFound):
		return &linkwardenFailure{http.StatusNotFound, "Not found."}
	case errors.As(err, &input):
		return &linkwardenFailure{http.StatusBadRequest, input.Message}
	case errors.Is(err, linkwarden.ErrRootCollection), errors.Is(err, linkwarden.ErrTagCreate):
		return &linkwardenFailure{http.StatusForbidden, err.Error()}
	}
	var apiErr *Error
	if errors.As(favouritesError(err), &apiErr) && apiErr.Status == http.StatusBadRequest {
		return &linkwardenFailure{http.StatusBadRequest, apiErr.Message + "."}
	}
	return nil
}

// LookupImages makes noFavicons a linkwarden.Favicons too.
func (noFavicons) LookupImages(context.Context, []string) (map[string]images.FaviconImage, error) {
	return map[string]images.FaviconImage{}, nil
}
