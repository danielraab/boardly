package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/favourites"
	"boardly/internal/karakeep"
	"boardly/internal/trpc"
	"boardly/internal/users"
)

// karakeepPrefix is the server address that Karakeep clients are given; they
// append Karakeep's own paths (api/trpc, api/health, …) to it.
const karakeepPrefix = "/api/karakeep"

// karakeepAPI serves what the Karakeep browser extension and mobile app call:
// their tRPC procedures and two status endpoints. Nothing else of Karakeep's
// API is implemented.
func (a *api) karakeepAPI() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET "+karakeepPrefix+"/api/health", HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Boardly is working"})
		return nil
	}))
	mux.Handle("GET "+karakeepPrefix+"/api/version", HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, map[string]string{"version": "Boardly"})
		return nil
	}))
	mux.Handle(karakeepPrefix+"/api/trpc/", &trpc.Server{
		Prefix:     karakeepPrefix + "/api/trpc/",
		Procedures: a.karakeepProcedures(),
		Authorized: func(ctx context.Context) bool {
			_, ok := appToken(ctx)
			return ok
		},
	})
	return mux
}

// karakeepProcedures returns the tRPC procedures: login and configuration,
// and bookmarks, lists, tags and user data on the token's subtree.
func (a *api) karakeepProcedures() map[string]trpc.Procedure {
	var icons karakeep.Favicons = noFavicons{}
	if a.favicons != nil {
		icons = a.favicons
	}
	k := &karakeep.Service{Favourites: a.favourites, Favicons: icons, Origin: a.origin,
		Now: func() time.Time { return a.favourites.Now() }}
	query := func(h func(context.Context, json.RawMessage) (any, error)) trpc.Procedure {
		return trpc.Procedure{Kind: trpc.Query, Handle: h}
	}
	mutation := func(h func(context.Context, json.RawMessage) (any, error)) trpc.Procedure {
		return trpc.Procedure{Kind: trpc.Mutation, Handle: h}
	}
	type bookmarkID struct {
		BookmarkID string `json:"bookmarkId"`
	}
	type listID struct {
		ListID string `json:"listId"`
	}
	type listAndBookmark struct {
		ListID     string `json:"listId"`
		BookmarkID string `json:"bookmarkId"`
	}
	type tagID struct {
		TagID string `json:"tagId"`
	}
	return map[string]trpc.Procedure{
		"apiKeys.validate":    {Kind: trpc.Mutation, Public: true, Handle: a.karakeepValidateKey},
		"apiKeys.exchange":    {Kind: trpc.Mutation, Public: true, Handle: karakeepExchange},
		"users.whoami":        query(a.karakeepWhoAmI),
		"config.clientConfig": {Kind: trpc.Query, Public: true, Handle: a.karakeepClientConfig},

		"bookmarks.getBookmarks":    query(scoped(k.Bookmarks)),
		"bookmarks.searchBookmarks": query(scoped(k.Search)),
		"bookmarks.getBookmark": query(scoped(func(ctx context.Context, sc favourites.Scope, in bookmarkID) (karakeep.Bookmark, error) {
			return k.GetBookmark(ctx, sc, in.BookmarkID)
		})),
		"bookmarks.checkUrl": query(scoped(func(ctx context.Context, sc favourites.Scope, in struct {
			URL string `json:"url"`
		}) (any, error) {
			id, err := k.CheckURL(ctx, sc, in.URL)
			return map[string]*string{"bookmarkId": id}, err
		})),
		"bookmarks.createBookmark": mutation(scoped(k.CreateBookmark)),
		"bookmarks.updateBookmark": mutation(scoped(k.UpdateBookmark)),
		"bookmarks.updateTags":     mutation(scoped(k.UpdateTags)),
		"bookmarks.deleteBookmark": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in bookmarkID) (any, error) {
			return nil, k.DeleteBookmark(ctx, sc, in.BookmarkID)
		})),

		"lists.list": query(scoped(func(ctx context.Context, sc favourites.Scope, _ struct{}) (any, error) {
			lists, err := k.Lists(ctx, sc)
			return map[string][]karakeep.List{"lists": lists}, err
		})),
		"lists.get": query(scoped(func(ctx context.Context, sc favourites.Scope, in listID) (karakeep.List, error) {
			return k.List(ctx, sc, in.ListID)
		})),
		"lists.stats": query(scoped(func(ctx context.Context, sc favourites.Scope, _ struct{}) (any, error) {
			stats, err := k.ListStats(ctx, sc)
			return map[string]trpc.Map{"stats": stats}, err
		})),
		"lists.getListsOfBookmark": query(scoped(func(ctx context.Context, sc favourites.Scope, in bookmarkID) (any, error) {
			lists, err := k.ListsOfBookmark(ctx, sc, in.BookmarkID)
			return map[string][]karakeep.List{"lists": lists}, err
		})),
		// Boardly has no list sharing, so there are never invitations.
		"lists.getPendingInvitations": query(func(context.Context, json.RawMessage) (any, error) {
			return []struct{}{}, nil
		}),
		"lists.create": mutation(scoped(k.CreateList)),
		"lists.edit":   mutation(scoped(k.EditList)),
		"lists.delete": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in listID) (any, error) {
			return nil, k.DeleteList(ctx, sc, in.ListID)
		})),
		"lists.addToList": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in listAndBookmark) (any, error) {
			return nil, k.AddToList(ctx, sc, in.ListID, in.BookmarkID)
		})),
		"lists.removeFromList": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in listAndBookmark) (any, error) {
			return nil, k.RemoveFromList(ctx, sc, in.ListID, in.BookmarkID)
		})),

		"tags.list": query(scoped(k.Tags)),
		"tags.get": query(scoped(func(ctx context.Context, sc favourites.Scope, in tagID) (karakeep.TagInfo, error) {
			return k.Tag(ctx, sc, in.TagID)
		})),
		"tags.update": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in struct {
			TagID string  `json:"tagId"`
			Name  *string `json:"name"`
		}) (favourites.Tag, error) {
			return k.RenameTag(ctx, sc, in.TagID, in.Name)
		})),
		"tags.delete": mutation(scoped(func(ctx context.Context, sc favourites.Scope, in tagID) (any, error) {
			return nil, k.DeleteTag(ctx, sc, in.TagID)
		})),
		"tags.create": mutation(scoped(func(context.Context, favourites.Scope, struct{}) (any, error) {
			return nil, k.CreateTag()
		})),

		"users.settings": query(scoped(func(context.Context, favourites.Scope, struct{}) (karakeep.Settings, error) {
			return k.UserSettings(), nil
		})),
		"users.stats": query(scoped(func(ctx context.Context, sc favourites.Scope, _ struct{}) (karakeep.Stats, error) {
			return k.UserStats(ctx, sc)
		})),
	}
}

// scoped adapts a Karakeep service method to a procedure handler: it decodes
// the input (a missing one is the zero value), passes the token's scope and
// maps the service's errors to tRPC errors.
func scoped[In, Out any](h func(context.Context, favourites.Scope, In) (Out, error)) func(context.Context, json.RawMessage) (any, error) {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		var in In
		if len(raw) > 0 && string(raw) != "null" {
			var err error
			if in, err = trpc.Decode[In](raw); err != nil {
				return nil, err
			}
		}
		// Procedures run only with a token, and a token without a root
		// folder can't authenticate.
		token, _ := appToken(ctx)
		if token.RootID == nil {
			return nil, trpc.ErrUnauthorized
		}
		out, err := h(ctx, favourites.Scope{UserID: token.UserID, RootID: *token.RootID}, in)
		if err != nil {
			return nil, karakeepError(err)
		}
		return out, nil
	}
}

// karakeepError maps the errors of the Karakeep service and the favourites
// store to tRPC errors; other errors pass through and become internal errors.
func karakeepError(err error) error {
	var input *karakeep.InputError
	switch {
	case errors.Is(err, favourites.ErrNotFound):
		return trpc.NewError("NOT_FOUND", "Not found")
	case errors.As(err, &input):
		return trpc.NewError("BAD_REQUEST", input.Message)
	case errors.Is(err, karakeep.ErrTagShared):
		return trpc.NewError("FORBIDDEN", "This tag is also used outside this connected app's folder and can't be renamed here.")
	case errors.Is(err, karakeep.ErrTagCreate):
		return trpc.NewError("FORBIDDEN", "In Boardly, tags are created by adding them to a bookmark.")
	case errors.Is(err, favourites.ErrTagExists):
		return trpc.NewError("CONFLICT", "A tag with this name already exists.")
	}
	var apiErr *Error
	if errors.As(favouritesError(err), &apiErr) && apiErr.Status == http.StatusBadRequest {
		return trpc.NewError("BAD_REQUEST", apiErr.Message)
	}
	return err
}

// noFavicons stands in when favicons aren't configured.
type noFavicons struct{}

func (noFavicons) Lookup(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// karakeepValidateKey is apiKeys.validate: the login with an API key, which
// clients send in the input before they use it as their bearer token.
func (a *api) karakeepValidateKey(ctx context.Context, input json.RawMessage) (any, error) {
	in, err := trpc.Decode[struct {
		APIKey string `json:"apiKey"`
	}](input)
	if err != nil {
		return nil, err
	}
	if _, err := a.appTokens.Authenticate(ctx, in.APIKey, apptokens.AppKarakeep); err != nil {
		if errors.Is(err, apptokens.ErrInvalidToken) {
			return nil, trpc.ErrUnauthorized
		}
		return nil, err
	}
	return map[string]bool{"success": true}, nil
}

// karakeepExchange is apiKeys.exchange, the login with email and password,
// which Boardly doesn't have.
func karakeepExchange(context.Context, json.RawMessage) (any, error) {
	return nil, trpc.NewError("UNAUTHORIZED", "Boardly has no passwords. Log in with an API key from Settings → Connected apps.")
}

// karakeepWhoAmI is users.whoami: the token's owner.
func (a *api) karakeepWhoAmI(ctx context.Context, _ json.RawMessage) (any, error) {
	token, _ := appToken(ctx)
	user, err := a.users.Get(ctx, token.UserID)
	if errors.Is(err, users.ErrNotFound) {
		return nil, trpc.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return struct {
		ID        string  `json:"id"`
		Name      *string `json:"name"`
		Email     string  `json:"email"`
		Image     *string `json:"image"`
		LocalUser bool    `json:"localUser"`
	}{user.ID, user.Name, user.Email, nil, true}, nil
}

// karakeepClientConfig is config.clientConfig. It has every field of
// Karakeep's client configuration, because the apps read nested fields without
// checks, and turns off everything Boardly doesn't offer.
func (a *api) karakeepClientConfig(context.Context, json.RawMessage) (any, error) {
	return map[string]any{
		"publicUrl":    a.origin,
		"publicApiUrl": a.origin + karakeepPrefix + "/api",
		"auth": map[string]bool{
			"disableSignups":      true,
			"disablePasswordAuth": true,
			"oauthAutoRedirect":   false,
		},
		"turnstile": nil,
		"inference": map[string]any{
			"isConfigured":            false,
			"inferredTagLang":         "english",
			"enableAutoTagging":       false,
			"enableAutoSummarization": false,
		},
		"chat":                   map[string]bool{"enabled": false},
		"search":                 map[string]bool{"semanticSearchEnabled": false},
		"stripe":                 map[string]bool{"isConfigured": false},
		"crawler":                map[string]bool{"videoDownloadOnDemand": false},
		"legal":                  map[string]any{},
		"disableNewReleaseCheck": true,
	}, nil
}
