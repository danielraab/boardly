package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"boardly/internal/apptokens"
	"boardly/internal/memos"
	"boardly/internal/notes"
	"boardly/internal/users"
)

// memosPrefix is the server address that Memos clients (MoeMemos) are given;
// they append Memos' own paths (api/v1/memos, …).
const memosPrefix = "/api/memos"

// memosAPI serves the Memos v1 endpoints MoeMemos calls. Every path that
// isn't listed, among them the legacy api/v1/status, answers 404 in Memos'
// error format, so MoeMemos uses the v1 API.
func (a *api) memosAPI() *http.ServeMux {
	svc := &memos.Service{Notes: a.notes}
	v1 := memosPrefix + "/api/v1"
	mux := http.NewServeMux()
	// respond registers a handler that needs a token; its result is written
	// as JSON.
	respond := func(pattern string, h func(r *http.Request, token apptokens.Token, sc notes.Scope) (any, error)) {
		mux.Handle(pattern, memosServe(func(w http.ResponseWriter, r *http.Request) error {
			token, ok := appToken(r.Context())
			if !ok {
				return memos.ErrUnauthenticated
			}
			v, err := h(r, token, memosScope(token))
			if err != nil {
				return err
			}
			WriteJSON(w, http.StatusOK, v)
			return nil
		}))
	}

	mux.Handle("GET "+v1+"/instance/profile", memosServe(func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, memos.Profile{Version: memos.Version, Mode: "prod", InstanceURL: a.origin})
		return nil
	}))
	respond("GET "+v1+"/auth/me", func(r *http.Request, token apptokens.Token, _ notes.Scope) (any, error) {
		user, err := a.memosUser(r, token)
		if err != nil {
			return nil, err
		}
		return map[string]memos.User{"user": user}, nil
	})
	// users/{id}:getStats is one path segment, so it is told apart here.
	respond("GET "+v1+"/users/{id}", func(r *http.Request, token apptokens.Token, _ notes.Scope) (any, error) {
		id, stats := strings.CutSuffix(r.PathValue("id"), ":getStats")
		if id != token.UserID {
			return nil, memos.ErrNotFound
		}
		if stats {
			return memos.StatsOf(id), nil
		}
		return a.memosUser(r, token)
	})
	respond("GET "+v1+"/users/{id}/settings/{setting}", func(r *http.Request, token apptokens.Token, _ notes.Scope) (any, error) {
		if r.PathValue("id") != token.UserID || r.PathValue("setting") != "GENERAL" {
			return nil, memos.ErrNotFound
		}
		return memos.SettingOf(token.UserID), nil
	})

	respond("GET "+v1+"/memos", func(r *http.Request, _ apptokens.Token, sc notes.Scope) (any, error) {
		q := r.URL.Query()
		size, _ := strconv.Atoi(q.Get("pageSize"))
		return svc.List(r.Context(), sc, memos.ListInput{
			State: q.Get("state"), PageSize: size, PageToken: q.Get("pageToken"), Filter: q.Get("filter"),
		})
	})
	respond("POST "+v1+"/memos", func(r *http.Request, _ apptokens.Token, sc notes.Scope) (any, error) {
		var in memos.CreateInput
		if err := decodeMemos(r, &in); err != nil {
			return nil, err
		}
		return svc.Create(r.Context(), sc, in)
	})
	respond("PATCH "+v1+"/memos/{id}", func(r *http.Request, _ apptokens.Token, sc notes.Scope) (any, error) {
		var body map[string]json.RawMessage
		if err := decodeMemos(r, &body); err != nil {
			return nil, err
		}
		in, err := memos.UpdateFromBody(body, r.URL.Query().Get("updateMask"))
		if err != nil {
			return nil, err
		}
		return svc.Update(r.Context(), sc, r.PathValue("id"), in)
	})
	respond("DELETE "+v1+"/memos/{id}", func(r *http.Request, _ apptokens.Token, sc notes.Scope) (any, error) {
		return struct{}{}, svc.Delete(r.Context(), sc, r.PathValue("id"))
	})

	// MoeMemos lists attachments while syncing, so an empty list keeps the
	// sync going; uploads fail until attachments are supported.
	respond("GET "+v1+"/attachments", func(*http.Request, apptokens.Token, notes.Scope) (any, error) {
		return map[string]any{"attachments": []struct{}{}, "nextPageToken": ""}, nil
	})
	respond("POST "+v1+"/attachments", func(*http.Request, apptokens.Token, notes.Scope) (any, error) {
		return nil, memos.ErrAttachments
	})
	respond("DELETE "+v1+"/attachments/{id}", func(*http.Request, apptokens.Token, notes.Scope) (any, error) {
		return nil, memos.ErrNotFound
	})

	mux.Handle(memosPrefix+"/", memosServe(func(http.ResponseWriter, *http.Request) error {
		return memos.ErrNotFound
	}))
	return mux
}

// memosScope is the notes a Memos token sees.
func memosScope(token apptokens.Token) notes.Scope {
	sc := notes.Scope{UserID: token.UserID, Mode: notes.ScopeMode(token.NoteScope)}
	if token.RootCategoryID != nil {
		sc.CategoryID = *token.RootCategoryID
	}
	return sc
}

// memosUser is the token's owner as a Memos user.
func (a *api) memosUser(r *http.Request, token apptokens.Token) (memos.User, error) {
	user, err := a.users.Get(r.Context(), token.UserID)
	if errors.Is(err, users.ErrNotFound) {
		return memos.User{}, memos.ErrUnauthenticated
	}
	if err != nil {
		return memos.User{}, err
	}
	return memos.UserOf(user), nil
}

// decodeMemos reads a JSON body of at most the size of a note.
func decodeMemos(r *http.Request, v any) error {
	err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxNoteBytes)).Decode(v)
	if maxErr := new(http.MaxBytesError); errors.As(err, &maxErr) {
		return &memos.Error{Status: http.StatusBadRequest, Code: memos.CodeInvalidArgument, Message: "The memo is too large."}
	}
	if err != nil {
		return &memos.Error{Status: http.StatusBadRequest, Code: memos.CodeInvalidArgument, Message: "The request body is not valid JSON."}
	}
	return nil
}

// memosServe runs h and writes its error as a Memos error.
func memosServe(h func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		var e *memos.Error
		if !errors.As(err, &e) {
			slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
			e = &memos.Error{Status: http.StatusInternalServerError, Code: 13, Message: "Internal server error."}
		}
		WriteJSON(w, e.Status, map[string]any{"code": e.Code, "message": e.Message, "details": []struct{}{}})
	})
}
