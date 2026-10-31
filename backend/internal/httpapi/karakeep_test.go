package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/favourites"
	"boardly/internal/notes"
)

const trpcPath = karakeepPrefix + "/api/trpc/"

// trpcQuery sends a batched tRPC query as httpBatchLink does.
func (s *server) trpcQuery(procedures, input string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	return s.req("GET", trpcPath+procedures+"?batch=1&input="+url.QueryEscape(input), "", opts...)
}

// trpcMutation sends a batched tRPC mutation from a browser extension.
func (s *server) trpcMutation(procedure, body string, opts ...func(*http.Request)) *httptest.ResponseRecorder {
	opts = append([]func(*http.Request){fromExtension}, opts...)
	return s.req("POST", trpcPath+procedure+"?batch=1", body, opts...)
}

func fromExtension(r *http.Request) { r.Header.Set("Origin", "chrome-extension://abc") }

type trpcEntry struct {
	Result *struct {
		Data struct {
			JSON json.RawMessage `json:"json"`
		} `json:"data"`
	} `json:"result"`
	Error *struct {
		JSON struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Data    struct {
				Code string `json:"code"`
			} `json:"data"`
		} `json:"json"`
	} `json:"error"`
}

// trpcEntries decodes a tRPC response and checks its CORS header.
func trpcEntries(t *testing.T, rec *httptest.ResponseRecorder) []trpcEntry {
	t.Helper()
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("no CORS header: %v", rec.Header())
	}
	return decodeJSONBody[[]trpcEntry](t, rec)
}

// trpcErrorCode returns the tRPC error code of a single-call response.
func trpcErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	entries := trpcEntries(t, rec)
	if len(entries) != 1 || entries[0].Error == nil {
		t.Fatalf("want one error entry, got %s", rec.Body)
	}
	return entries[0].Error.JSON.Data.Code
}

func TestKarakeepLogin(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)
	revoked := s.createToken(t, ann, nil)
	s.req("POST", "/api/app-tokens/"+revoked.ID+"/revoke", "", withCookie(ann))

	// The extension validates the key before it has a bearer token.
	rec := s.trpcMutation("apiKeys.validate", `{"0":{"json":{"apiKey":"`+token.Secret+`"}}}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `[{"result":{"data":{"json":{"success":true}}}}]`+"\n" {
		t.Fatalf("validate: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); list[1].LastUsedAt == nil {
		t.Error("validate didn't record the last use")
	}
	for name, key := range map[string]string{"revoked": revoked.Secret, "unknown": "bdly_nope", "empty": ""} {
		rec := s.trpcMutation("apiKeys.validate", `{"0":{"json":{"apiKey":"`+key+`"}}}`)
		if rec.Code != http.StatusUnauthorized || trpcErrorCode(t, rec) != "UNAUTHORIZED" {
			t.Errorf("validate %s: %d %s", name, rec.Code, rec.Body)
		}
	}

	rec = s.trpcMutation("apiKeys.exchange", `{"0":{"json":{"email":"ann@example.org","password":"x","keyName":"ext"}}}`)
	if rec.Code != http.StatusUnauthorized || trpcErrorCode(t, rec) != "UNAUTHORIZED" {
		t.Errorf("exchange: %d %s", rec.Code, rec.Body)
	}

	// After login, the apps query their user and the configuration in one batch.
	rec = s.trpcQuery("users.whoami,config.clientConfig", `{"0":{"json":null},"1":{"json":null}}`, withBearer(token.Secret))
	entries := trpcEntries(t, rec)
	if rec.Code != http.StatusOK || len(entries) != 2 || entries[0].Result == nil || entries[1].Result == nil {
		t.Fatalf("batch: %d %s", rec.Code, rec.Body)
	}
	var me struct {
		ID        string  `json:"id"`
		Email     string  `json:"email"`
		Image     *string `json:"image"`
		LocalUser bool    `json:"localUser"`
	}
	if err := json.Unmarshal(entries[0].Result.Data.JSON, &me); err != nil || me.ID == "" || me.Email != "ann@example.org" || !me.LocalUser {
		t.Errorf("whoami = %+v (%v)", me, err)
	}
	var config struct {
		PublicURL string `json:"publicUrl"`
		Auth      struct {
			DisablePasswordAuth bool `json:"disablePasswordAuth"`
		} `json:"auth"`
		Inference struct {
			IsConfigured bool `json:"isConfigured"`
		} `json:"inference"`
	}
	if err := json.Unmarshal(entries[1].Result.Data.JSON, &config); err != nil || config.PublicURL != testOrigin || !config.Auth.DisablePasswordAuth || config.Inference.IsConfigured {
		t.Errorf("clientConfig = %+v (%v)", config, err)
	}
}

func TestKarakeepBearerAuth(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	expires := s.clock.Now().Add(time.Hour)
	active := s.createToken(t, ann, expires.Format(time.RFC3339))
	revoked := s.createToken(t, ann, nil)
	s.req("POST", "/api/app-tokens/"+revoked.ID+"/revoke", "", withCookie(ann))

	if rec := s.trpcQuery("users.whoami", `{}`, withBearer(active.Secret)); rec.Code != http.StatusOK {
		t.Fatalf("valid token: %d %s", rec.Code, rec.Body)
	}
	if list := decodeJSONBody[[]apptokens.Token](t, s.req("GET", "/api/app-tokens", "", withCookie(ann))); list[1].LastUsedAt == nil || !list[1].LastUsedAt.Equal(s.clock.Now()) {
		t.Errorf("last used = %v, want %v", list[1].LastUsedAt, s.clock.Now())
	}

	unknown := s.trpcQuery("users.whoami", `{}`, withBearer(apptokens.SecretPrefix+"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
	if unknown.Code != http.StatusUnauthorized || trpcErrorCode(t, unknown) != "UNAUTHORIZED" {
		t.Fatalf("unknown token: %d %s", unknown.Code, unknown.Body)
	}
	for name, opts := range map[string][]func(*http.Request){
		"cookie only":    {withCookie(ann)},
		"nothing":        nil,
		"revoked":        {withBearer(revoked.Secret)},
		"basic scheme":   {func(r *http.Request) { r.Header.Set("Authorization", "Basic "+active.Secret) }},
		"cookie and bad": {withCookie(ann), withBearer("bdly_nope")},
	} {
		rec := s.trpcQuery("users.whoami", `{}`, opts...)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
			t.Errorf("%s: %d %s, want 401 like an unknown token", name, rec.Code, rec.Body)
		}
	}

	// Public procedures and endpoints work without a token.
	if rec := s.trpcQuery("config.clientConfig", `{}`); rec.Code != http.StatusOK {
		t.Errorf("clientConfig without token: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/health", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"message":"Boardly is working","status":"ok"}`+"\n" {
		t.Errorf("health: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/version", ""); rec.Code != http.StatusOK || rec.Body.String() != `{"version":"Boardly"}`+"\n" {
		t.Errorf("version: %d %s", rec.Code, rec.Body)
	}

	// Anything else of Karakeep's API doesn't exist, the REST API included
	// beyond the endpoints that Floccus uses.
	if rec := s.trpcQuery("bookmarks.summarizeBookmark", `{}`, withBearer(active.Secret)); rec.Code != http.StatusNotFound || trpcErrorCode(t, rec) != "NOT_FOUND" {
		t.Errorf("unknown procedure: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", karakeepPrefix+"/api/v1/highlights", "", withBearer(active.Secret)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Errorf("REST path: %d %s", rec.Code, rec.Body)
	}

	s.clock.Advance(time.Hour)
	if rec := s.trpcQuery("users.whoami", `{}`, withBearer(active.Secret)); rec.Code != http.StatusUnauthorized || rec.Body.String() != unknown.Body.String() {
		t.Errorf("expired: %d %s", rec.Code, rec.Body)
	}
}

func TestKarakeepSkipsCSRFAndAllowsCORS(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)

	rec := s.req("OPTIONS", trpcPath+"users.whoami?batch=1", "", fromExtension, func(r *http.Request) {
		r.Header.Set("Access-Control-Request-Method", "GET")
		r.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-proxy-auth")
	})
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "*" ||
		rec.Header().Get("Access-Control-Allow-Headers") != "authorization,content-type,x-proxy-auth" {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}

	// A cross-origin request with a non-JSON body isn't rejected by the
	// cross-site check: it reaches the tRPC layer, which rejects the batch.
	rec = s.trpcMutation("apiKeys.validate", "x", withBearer(token.Secret), func(r *http.Request) {
		r.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	})
	if rec.Code != http.StatusBadRequest || trpcErrorCode(t, rec) != "BAD_REQUEST" {
		t.Errorf("cross-origin POST: %d %s, want 400 BAD_REQUEST", rec.Code, rec.Body)
	}

	// The web app's API keeps its cross-site check and has no CORS headers.
	rec = s.req("POST", "/api/app-tokens", `{}`, withCookie(ann), fromExtension)
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != "forbidden_origin" || rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("web API cross-origin: %d %s", rec.Code, rec.Body)
	}
}

// trpcData returns the json and meta of a single successful call.
func trpcData(t *testing.T, rec *httptest.ResponseRecorder) (json.RawMessage, map[string][]string) {
	t.Helper()
	var entries []struct {
		Result *struct {
			Data struct {
				JSON json.RawMessage `json:"json"`
				Meta *struct {
					Values map[string][]string `json:"values"`
				} `json:"meta"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil || len(entries) != 1 || entries[0].Result == nil {
		t.Fatalf("want one result, got %d %s", rec.Code, rec.Body)
	}
	data := entries[0].Result.Data
	if data.Meta == nil {
		return data.JSON, nil
	}
	return data.JSON, data.Meta.Values
}

func input(v any) string { return `{"0":{"json":` + mustJSON(v) + `}}` }

func TestKarakeepSavePageFromExtension(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)
	bearer := withBearer(token.Secret)
	s.create(t, ann, map[string]any{"kind": "link", "parentId": *token.RootID, "url": "https://example.org/old", "title": "Old"})
	const page = "https://example.org/article"

	check := func() *string {
		t.Helper()
		data, _ := trpcData(t, s.trpcQuery("bookmarks.checkUrl", input(map[string]string{"url": page}), bearer))
		var out struct {
			BookmarkID *string `json:"bookmarkId"`
		}
		_ = json.Unmarshal(data, &out)
		return out.BookmarkID
	}
	if id := check(); id != nil {
		t.Fatalf("checkUrl before saving = %v", *id)
	}

	rec := s.trpcMutation("bookmarks.createBookmark", input(map[string]any{
		"type": "link", "url": page, "title": "Article", "source": "extension",
	}), bearer)
	data, meta := trpcData(t, rec)
	var created struct {
		ID            string `json:"id"`
		AlreadyExists bool   `json:"alreadyExists"`
	}
	_ = json.Unmarshal(data, &created)
	if rec.Code != http.StatusOK || created.ID == "" || created.AlreadyExists {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"createdAt", "modifiedAt", "content.crawledAt"} {
		if !reflect.DeepEqual(meta[path], []string{"Date"}) {
			t.Errorf("meta[%s] = %v (%v)", path, meta[path], meta)
		}
	}
	if id := check(); id == nil || *id != created.ID {
		t.Errorf("checkUrl after saving = %v", id)
	}

	// The bookmark page of the extension loads the bookmark and its lists.
	rec = s.trpcQuery("bookmarks.getBookmark,lists.getListsOfBookmark",
		`{"0":{"json":{"bookmarkId":"`+created.ID+`","includeContent":false}},"1":{"json":{"bookmarkId":"`+created.ID+`"}}}`, bearer)
	if entries := trpcEntries(t, rec); rec.Code != http.StatusOK || entries[0].Result == nil || entries[1].Result == nil {
		t.Fatalf("bookmark page: %d %s", rec.Code, rec.Body)
	}

	// The link is first in the token's folder.
	tree := decodeJSONBody[favourites.Node](t, s.req("GET", "/api/favourites", "", withCookie(ann)))
	reading := tree.Children[0]
	if len(reading.Children) != 2 || *reading.Children[0].Title != "Article" {
		t.Errorf("Reading = %+v", reading.Children)
	}

	// A file bookmark without an uploaded file fails and adds nothing.
	rec = s.trpcMutation("bookmarks.createBookmark", input(map[string]any{"type": "asset", "assetType": "image"}), bearer)
	if rec.Code != http.StatusBadRequest || trpcErrorCode(t, rec) != "BAD_REQUEST" {
		t.Errorf("asset bookmark: %d %s", rec.Code, rec.Body)
	}
	rec = s.trpcMutation("bookmarks.deleteBookmark", input(map[string]string{"bookmarkId": created.ID}), bearer)
	if rec.Code != http.StatusOK || rec.Body.String() != `[{"result":{"data":{"json":null}}}]`+"\n" {
		t.Errorf("delete: %d %s", rec.Code, rec.Body)
	}
}

func TestKarakeepBrowseInApp(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	token := s.createToken(t, ann, nil)
	bearer := withBearer(token.Secret)
	for _, title := range []string{"One", "Two", "Three"} {
		s.clock.Advance(time.Minute)
		s.trpcMutation("bookmarks.createBookmark", input(map[string]any{"type": "link", "url": "https://example.org/" + title, "title": title}), bearer)
	}
	outside := s.create(t, ann, map[string]any{"kind": "link", "url": "https://example.org/outside"})

	// First page of two, then the next page with the cursor sent back as the
	// client does: a superjson Date with its meta.
	data, meta := trpcData(t, s.trpcQuery("bookmarks.getBookmarks", input(map[string]any{"limit": 2, "useCursorV2": true}), bearer))
	var first struct {
		Bookmarks []struct {
			Title string `json:"title"`
		} `json:"bookmarks"`
		NextCursor json.RawMessage `json:"nextCursor"`
	}
	_ = json.Unmarshal(data, &first)
	if len(first.Bookmarks) != 2 || first.Bookmarks[0].Title != "Three" || !reflect.DeepEqual(meta["nextCursor.createdAt"], []string{"Date"}) {
		t.Fatalf("first page = %s %v", data, meta)
	}
	next := `{"0":{"json":{"limit":2,"cursor":` + string(first.NextCursor) + `},"meta":{"values":{"cursor.createdAt":["Date"]}}}}`
	data, _ = trpcData(t, s.trpcQuery("bookmarks.getBookmarks", next, bearer))
	var second struct {
		Bookmarks []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"bookmarks"`
		NextCursor *json.RawMessage `json:"nextCursor"`
	}
	_ = json.Unmarshal(data, &second)
	if len(second.Bookmarks) != 1 || second.Bookmarks[0].Title != "One" || second.NextCursor != nil {
		t.Errorf("second page = %s", data)
	}

	// Lists: create one, add a bookmark, then the app's list screen.
	data, _ = trpcData(t, s.trpcMutation("lists.create", input(map[string]any{"name": "Later", "icon": "🚀", "type": "manual"}), bearer))
	var list struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(data, &list)
	if rec := s.trpcMutation("lists.addToList", input(map[string]string{"listId": list.ID, "bookmarkId": second.Bookmarks[0].ID}), bearer); rec.Code != http.StatusOK {
		t.Fatalf("addToList: %d %s", rec.Code, rec.Body)
	}
	rec := s.trpcQuery("lists.list,lists.stats,lists.getPendingInvitations", `{}`, bearer)
	var entries []struct {
		Result struct {
			Data struct {
				JSON json.RawMessage `json:"json"`
				Meta json.RawMessage `json:"meta"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil || rec.Code != http.StatusOK || len(entries) != 3 {
		t.Fatalf("list screen: %d %s", rec.Code, rec.Body)
	}
	if got := string(entries[1].Result.Data.JSON); got != `{"stats":[["`+list.ID+`",1]]}` || string(entries[1].Result.Data.Meta) != `{"values":{"stats":["map"]}}` {
		t.Errorf("stats = %s %s", got, entries[1].Result.Data.Meta)
	}
	if got := string(entries[2].Result.Data.JSON); got != `[]` {
		t.Errorf("invitations = %s", got)
	}

	// Tags, settings and stats.
	s.trpcMutation("bookmarks.updateTags", input(map[string]any{"bookmarkId": second.Bookmarks[0].ID, "attach": []any{map[string]string{"tagName": "dev"}}, "detach": []any{}}), bearer)
	data, _ = trpcData(t, s.trpcQuery("tags.list", input(map[string]any{"sortBy": "usage"}), bearer))
	if !strings.Contains(string(data), `"name":"dev","numBookmarks":1`) {
		t.Errorf("tags = %s", data)
	}
	if rec := s.trpcQuery("users.settings,users.stats", `{}`, bearer); rec.Code != http.StatusOK {
		t.Errorf("settings and stats: %d %s", rec.Code, rec.Body)
	}

	// Nodes outside the token's folder don't exist for it.
	if rec := s.trpcQuery("bookmarks.getBookmark", input(map[string]string{"bookmarkId": outside.ID}), bearer); rec.Code != http.StatusNotFound || trpcErrorCode(t, rec) != "NOT_FOUND" {
		t.Errorf("outside bookmark: %d %s", rec.Code, rec.Body)
	}
	// Unsupported features.
	if rec := s.trpcQuery("highlights.getForBookmark", input(map[string]string{"bookmarkId": second.Bookmarks[0].ID}), bearer); rec.Code != http.StatusNotFound {
		t.Errorf("highlights: %d %s", rec.Code, rec.Body)
	}
	if rec := s.trpcMutation("tags.create", input(map[string]string{"name": "x"}), bearer); rec.Code != http.StatusForbidden || trpcErrorCode(t, rec) != "FORBIDDEN" {
		t.Errorf("tags.create: %d %s", rec.Code, rec.Body)
	}
	if rec := s.trpcMutation("bookmarks.updateBookmark", input(map[string]any{"bookmarkId": second.Bookmarks[0].ID, "favourited": true}), bearer); rec.Code != http.StatusBadRequest {
		t.Errorf("favourite: %d %s", rec.Code, rec.Body)
	}
}

func TestKarakeepTextBookmarks(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	inbox := s.noteCategory(t, ann, "Inbox", "")
	ideas := s.noteCategory(t, ann, "Ideas", inbox)
	other := s.noteCategory(t, ann, "Other", "")
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Reading"})
	later := s.create(t, ann, map[string]any{"kind": "folder", "title": "Later", "parentId": folder.ID})
	rec := s.req("POST", "/api/app-tokens", mustJSON(map[string]any{
		"name": "Phone", "app": "karakeep", "rootId": folder.ID, "rootCategoryId": inbox,
	}), withCookie(ann))
	bearer := withBearer(decodeJSONBody[createdToken](t, rec).Secret)
	input := func(v any) string { return `{"0":{"json":` + mustJSON(v) + `}}` }
	result := func(rec *httptest.ResponseRecorder) map[string]any {
		t.Helper()
		entries := trpcEntries(t, rec)
		if len(entries) != 1 || entries[0].Result == nil {
			t.Fatalf("want one result, got %d %s", rec.Code, rec.Body)
		}
		var v map[string]any
		_ = json.Unmarshal(entries[0].Result.Data.JSON, &v)
		return v
	}

	// A text bookmark from the app is a note in the root category.
	created := result(s.trpcMutation("bookmarks.createBookmark", input(map[string]any{
		"type": "text", "title": "Shopping", "text": "- [x] milk\n- [ ] eggs",
	}), bearer))
	id, _ := created["id"].(string)
	content, _ := created["content"].(map[string]any)
	if content["type"] != "text" || content["text"] != "- [x] milk\n- [ ] eggs" || created["alreadyExists"] != false {
		t.Fatalf("created = %v", created)
	}
	note := decodeJSONBody[notes.Note](t, s.req("GET", "/api/notes/"+id, "", withCookie(ann)))
	if note.CategoryID == nil || *note.CategoryID != inbox || *note.Title != "Shopping" {
		t.Fatalf("note = %+v", note)
	}
	milk := (*note.Content.Blocks[0].Items)[0].ID

	// Notes of subcategories are visible, other notes aren't.
	s.req("POST", "/api/notes", `{"title":"Idea","categoryId":"`+ideas+`","text":"call the plumber"}`, withCookie(ann))
	hidden := decodeJSONBody[notes.Note](t, s.req("POST", "/api/notes", `{"title":"Hidden","categoryId":"`+other+`"}`, withCookie(ann)))
	page := result(s.trpcQuery("bookmarks.getBookmarks", input(map[string]any{}), bearer))
	if got := len(page["bookmarks"].([]any)); got != 2 {
		t.Fatalf("getBookmarks has %d bookmarks, want 2: %v", got, page)
	}
	if rec := s.trpcQuery("bookmarks.getBookmark", input(map[string]string{"bookmarkId": hidden.ID}), bearer); trpcErrorCode(t, rec) != "NOT_FOUND" {
		t.Errorf("note outside the scope: %s", rec.Body)
	}
	found := result(s.trpcQuery("bookmarks.searchBookmarks", input(map[string]any{"text": "plumber"}), bearer))
	if bs := found["bookmarks"].([]any); len(bs) != 1 || bs[0].(map[string]any)["title"] != "Idea" {
		t.Errorf("search = %v", found)
	}

	// Editing the text keeps the task IDs.
	result(s.trpcMutation("bookmarks.updateBookmark", input(map[string]any{
		"bookmarkId": id, "text": "- [x] milk\n- [ ] eggs\n- [ ] bread",
	}), bearer))
	note = decodeJSONBody[notes.Note](t, s.req("GET", "/api/notes/"+id, "", withCookie(ann)))
	if items := *note.Content.Blocks[0].Items; len(items) != 3 || items[0].ID != milk || note.Version != 2 {
		t.Errorf("after the edit: version %d, items %+v, want milk's ID %s", note.Version, items, milk)
	}

	for name, call := range map[string]*httptest.ResponseRecorder{
		"url on a note":  s.trpcMutation("bookmarks.updateBookmark", input(map[string]any{"bookmarkId": id, "url": "https://example.org"}), bearer),
		"tag a note":     s.trpcMutation("bookmarks.updateTags", input(map[string]any{"bookmarkId": id, "attach": []map[string]string{{"tagName": "x"}}, "detach": []any{}}), bearer),
		"note in a list": s.trpcMutation("lists.addToList", input(map[string]string{"listId": later.ID, "bookmarkId": id}), bearer),
	} {
		if code := trpcErrorCode(t, call); code != "BAD_REQUEST" {
			t.Errorf("%s: %s, want BAD_REQUEST", name, call.Body)
		}
	}
	lists := result(s.trpcQuery("lists.getListsOfBookmark", input(map[string]string{"bookmarkId": id}), bearer))
	if ls := lists["lists"].([]any); len(ls) != 0 {
		t.Errorf("lists of a note = %v", lists)
	}

	result(s.trpcMutation("bookmarks.deleteBookmark", input(map[string]string{"bookmarkId": id}), bearer))
	if rec := s.req("GET", "/api/notes/"+id, "", withCookie(ann)); rec.Code != http.StatusNotFound {
		t.Errorf("deleted note: %d", rec.Code)
	}
}
