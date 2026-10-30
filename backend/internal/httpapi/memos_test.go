package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"boardly/internal/memos"
	"boardly/internal/notes"
)

const memosV1 = memosPrefix + "/api/v1"

// memosClient is a Memos token of a signed-in user, as MoeMemos uses it.
type memosClient struct {
	s      *server
	cookie string
	secret string
	user   string // "users/<id>"
}

func (s *server) memosToken(t *testing.T, cookie string, categoryID string) memosClient {
	t.Helper()
	body := map[string]any{"name": "MoeMemos", "app": "memos"}
	if categoryID != "" {
		body["rootCategoryId"] = categoryID
	}
	rec := s.req("POST", "/api/app-tokens", mustJSON(body), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create token: %d %s", rec.Code, rec.Body)
	}
	c := memosClient{s: s, cookie: cookie, secret: decodeJSONBody[createdToken](t, rec).Secret}
	c.user = decodeJSONBody[struct{ User memos.User }](t, c.do("GET", memosV1+"/auth/me", "")).User.Name
	return c
}

func (c memosClient) do(method, path, body string) *httptest.ResponseRecorder {
	return c.s.req(method, path, body, withBearer(c.secret))
}

// memosError is a Memos error response.
type memosError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

func wantMemosError(t *testing.T, rec *httptest.ResponseRecorder, status, code int) {
	t.Helper()
	e := decodeJSONBody[memosError](t, rec)
	if rec.Code != status || e.Code != code || e.Message == "" || string(e.Details) != "[]" {
		t.Errorf("error = %d %s, want %d with code %d", rec.Code, rec.Body, status, code)
	}
}

func (c memosClient) list(t *testing.T, query string) memos.ListResponse {
	t.Helper()
	rec := c.do("GET", memosV1+"/memos?"+query, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list %s: %d %s", query, rec.Code, rec.Body)
	}
	return decodeJSONBody[memos.ListResponse](t, rec)
}

func (c memosClient) create(t *testing.T, body map[string]any) memos.Memo {
	t.Helper()
	rec := c.do("POST", memosV1+"/memos", mustJSON(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	return decodeJSONBody[memos.Memo](t, rec)
}

func (c memosClient) note(t *testing.T, memo memos.Memo) notes.Note {
	t.Helper()
	id := strings.TrimPrefix(memo.Name, "memos/")
	rec := c.s.req("GET", "/api/notes/"+id, "", withCookie(c.cookie))
	if rec.Code != http.StatusOK {
		t.Fatalf("note %s: %d %s", id, rec.Code, rec.Body)
	}
	return decodeJSONBody[notes.Note](t, rec)
}

func memoContents(list memos.ListResponse) []string {
	var out []string
	for _, m := range list.Memos {
		out = append(out, m.Content)
	}
	return out
}

func TestMemosConnect(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	// MoeMemos first tries the legacy status endpoint, then the profile,
	// both without a token.
	wantMemosError(t, s.req("GET", memosV1+"/status", ""), http.StatusNotFound, memos.CodeNotFound)
	rec := s.req("GET", memosV1+"/instance/profile", "")
	if p := decodeJSONBody[memos.Profile](t, rec); rec.Code != http.StatusOK || p.Version != "0.31.0" {
		t.Fatalf("profile: %d %s", rec.Code, rec.Body)
	}

	c := s.memosToken(t, ann, "")
	rec = c.do("GET", memosV1+"/auth/me", "")
	if me := decodeJSONBody[struct{ User memos.User }](t, rec).User; rec.Code != http.StatusOK || me.Username != "ann@example.org" || me.DisplayName == "" || !strings.HasPrefix(me.Name, "users/") {
		t.Fatalf("auth/me: %d %s", rec.Code, rec.Body)
	}
	id := strings.TrimPrefix(c.user, "users/")
	if rec := c.do("GET", memosV1+"/users/"+id+"/settings/GENERAL", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"memoVisibility":"PRIVATE"`) {
		t.Errorf("settings: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do("GET", memosV1+"/users/"+id+":getStats", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"tagCount":{}`) {
		t.Errorf("stats: %d %s", rec.Code, rec.Body)
	}
	if rec := c.do("GET", memosV1+"/users/"+id, ""); rec.Code != http.StatusOK || decodeJSONBody[memos.User](t, rec).Name != c.user {
		t.Errorf("user: %d %s", rec.Code, rec.Body)
	}

	other := s.memosToken(t, bob, "")
	wantMemosError(t, c.do("GET", memosV1+"/users/"+strings.TrimPrefix(other.user, "users/"), ""), http.StatusNotFound, memos.CodeNotFound)

	// Without a token, with a token of another app, or after revoking.
	wantMemosError(t, s.req("GET", memosV1+"/auth/me", ""), http.StatusUnauthorized, memos.CodeUnauthenticated)
	karakeep := s.createToken(t, ann, nil)
	wantMemosError(t, s.req("GET", memosV1+"/auth/me", "", withBearer(karakeep.Secret)), http.StatusUnauthorized, memos.CodeUnauthenticated)
	wantMemosError(t, c.do("DELETE", memosV1+"/memos/doesnotexist", ""), http.StatusNotFound, memos.CodeNotFound)
	wantMemosError(t, c.do("GET", memosPrefix+"/nope", ""), http.StatusNotFound, memos.CodeNotFound)

	// Preflights allow PATCH for memo updates.
	rec = s.req("OPTIONS", memosV1+"/memos/x", "", func(r *http.Request) {
		r.Header.Set("Origin", "https://app.example")
		r.Header.Set("Access-Control-Request-Method", "PATCH")
	})
	if rec.Code != http.StatusNoContent || !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Errorf("preflight: %d %v", rec.Code, rec.Header())
	}
}

func TestMemosSync(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	home := s.noteCategory(t, ann, "Home", "")
	garden := s.noteCategory(t, ann, "Garden", home)
	work := s.noteCategory(t, ann, "Work", "")
	for _, n := range []map[string]any{
		{"title": "Groceries", "categoryId": home, "content": json.RawMessage(`{"v":1,"blocks":[{"type":"list","style":"task","items":[{"checked":false,"content":[{"type":"text","text":"milk"}]}]}]}`)},
		{"text": "call the plumber", "categoryId": garden},
		{"title": "Sprint", "categoryId": work},
		{"title": "Loose"},
		{"title": "Old plan", "categoryId": home, "archived": true, "pinned": true},
	} {
		if rec := s.req("POST", "/api/notes", mustJSON(n), withCookie(ann)); rec.Code != http.StatusCreated {
			t.Fatalf("create note: %d %s", rec.Code, rec.Body)
		}
	}

	c := s.memosToken(t, ann, home)
	filter := "creator == \"" + c.user + "\""
	list := c.list(t, "pageSize=200&state=NORMAL&filter="+escapeQuery(filter))
	if got := memoContents(list); len(got) != 2 || got[0] != "call the plumber" || got[1] != "# Groceries\n\n- [ ] milk" || list.NextPageToken != "" {
		t.Fatalf("sync = %q, token %q", got, list.NextPageToken)
	}
	m := list.Memos[1]
	if m.State != "NORMAL" || m.Visibility != "PRIVATE" || m.Creator != c.user || m.Pinned || len(m.Tags) != 0 {
		t.Errorf("memo = %+v", m)
	}
	if !regexp.MustCompile(`^"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z"$`).MatchString(mustJSON(m.UpdateTime)) {
		t.Errorf("update time %s, want milliseconds in UTC", mustJSON(m.UpdateTime))
	}
	if again := c.list(t, "state=NORMAL"); mustJSON(again.Memos) != mustJSON(list.Memos) {
		t.Error("listing twice gave different memos")
	}
	if got := memoContents(c.list(t, "state=ARCHIVED")); len(got) != 1 || got[0] != "# Old plan" {
		t.Errorf("archive = %q", got)
	}
	if got := c.list(t, "filter="+escapeQuery(`visibility in ["PUBLIC", "PROTECTED"]`)); len(got.Memos) != 0 {
		t.Errorf("explore = %q", memoContents(got))
	}
	if got := c.list(t, "filter="+escapeQuery(`creator == "users/someone"`)); len(got.Memos) != 0 {
		t.Errorf("other creator = %q", memoContents(got))
	}
	wantMemosError(t, c.do("GET", memosV1+"/memos?pageToken=%21", ""), http.StatusBadRequest, memos.CodeInvalidArgument)

	// Pages.
	first := c.list(t, "pageSize=1")
	second := c.list(t, "pageSize=1&pageToken="+first.NextPageToken)
	if len(first.Memos) != 1 || first.NextPageToken == "" || len(second.Memos) != 1 || second.NextPageToken != "" || first.Memos[0].Name == second.Memos[0].Name {
		t.Errorf("pages = %+v, %+v", first, second)
	}

	// A token for all notes sees every unarchived note.
	all := s.memosToken(t, ann, "")
	if got := all.list(t, ""); len(got.Memos) != 4 {
		t.Errorf("all notes = %q", memoContents(got))
	}
}

func TestMemosWrite(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	inbox := s.noteCategory(t, ann, "Inbox", "")
	c := s.memosToken(t, ann, inbox)

	// Written in the app, also offline yesterday.
	m := c.create(t, map[string]any{"content": "milk\neggs", "visibility": "PRIVATE", "attachments": []any{}})
	n := c.note(t, m)
	if n.Title != nil || n.CategoryID == nil || *n.CategoryID != inbox || n.Pinned || n.Archived || m.Content != "milk\neggs" {
		t.Errorf("created note = %+v, memo %+v", n, m)
	}
	offline := c.create(t, map[string]any{"content": "# Trip\n\n- [ ] tickets", "createTime": "2026-02-28T08:00:00Z"})
	n = c.note(t, offline)
	if n.Title == nil || *n.Title != "Trip" || n.CreatedAt.Format("2006-01-02T15:04:05Z") != "2026-02-28T08:00:00Z" || strings.Contains(mustJSON(n.Content), `"heading"`) {
		t.Errorf("offline note = %+v", n)
	}
	second := c.create(t, map[string]any{"content": "## Week\n\n- [ ] review"})
	if n := c.note(t, second); n.Title != nil || !strings.Contains(mustJSON(n.Content), `"heading"`) {
		t.Errorf("second-level heading = %+v", n)
	}
	wantMemosError(t, c.do("POST", memosV1+"/memos", `{"content":"photo","attachments":[{"name":"attachments/1"}]}`), http.StatusBadRequest, memos.CodeInvalidArgument)
	if got := c.list(t, ""); len(got.Memos) != 3 {
		t.Errorf("a memo with an unknown attachment was created: %q", memoContents(got))
	}
	wantMemosError(t, c.do("POST", memosV1+"/memos", `{}`), http.StatusBadRequest, memos.CodeInvalidArgument)

	// Tick off a task: it keeps its ID, and the version goes up by one.
	before := c.note(t, offline)
	patch := func(memo memos.Memo, query, body string) *httptest.ResponseRecorder {
		return c.do("PATCH", memosV1+"/"+memo.Name+query, body)
	}
	rec := patch(offline, "", mustJSON(map[string]any{"content": "# Trip\n\n- [x] tickets", "updateTime": "2026-03-01T12:00:00Z"}))
	after := c.note(t, offline)
	if rec.Code != http.StatusOK || after.Version != before.Version+1 || !strings.Contains(mustJSON(after.Content), `"checked":true`) ||
		taskIDsOf(t, after) != taskIDsOf(t, before) {
		t.Fatalf("tick: %d %s, note %+v", rec.Code, rec.Body, after)
	}
	// Saving it unchanged is no change.
	if rec := patch(offline, "", mustJSON(map[string]any{"content": decodeJSONBody[memos.Memo](t, rec).Content})); rec.Code != http.StatusOK || c.note(t, offline).Version != after.Version {
		t.Errorf("unchanged save: %d %s", rec.Code, rec.Body)
	}
	// Rename by heading, then remove the heading.
	patch(offline, "", `{"content":"# Holiday\n\n- [x] tickets"}`)
	if n := c.note(t, offline); n.Title == nil || *n.Title != "Holiday" {
		t.Errorf("renamed = %v", n.Title)
	}
	patch(offline, "", `{"content":"- [x] tickets"}`)
	if n := c.note(t, offline); n.Title != nil {
		t.Errorf("heading removed, title = %v", *n.Title)
	}

	// Pin and archive.
	if rec := patch(m, "", `{"pinned":true}`); rec.Code != http.StatusOK || !decodeJSONBody[memos.Memo](t, rec).Pinned || !c.note(t, m).Pinned {
		t.Errorf("pin: %d %s", rec.Code, rec.Body)
	}
	if rec := patch(m, "", `{"state":"ARCHIVED"}`); rec.Code != http.StatusOK || !c.note(t, m).Archived {
		t.Errorf("archive: %d %s", rec.Code, rec.Body)
	}
	for _, memo := range c.list(t, "state=NORMAL").Memos {
		if memo.Name == m.Name {
			t.Error("the archived memo is still listed as NORMAL")
		}
	}
	// A mask limits what is applied.
	patch(m, "?updateMask=state", `{"state":"NORMAL","pinned":false}`)
	if n := c.note(t, m); n.Archived || !n.Pinned {
		t.Errorf("masked update: archived %v, pinned %v; want unarchived and still pinned", n.Archived, n.Pinned)
	}
	wantMemosError(t, patch(m, "", `{"state":"DELETED"}`), http.StatusBadRequest, memos.CodeInvalidArgument)
	wantMemosError(t, patch(m, "", `{"content":"x","attachments":[{"name":"attachments/1"}]}`), http.StatusBadRequest, memos.CodeInvalidArgument)
	if c.note(t, m).Title != nil || c.note(t, m).Content.Blocks[0].Type != "paragraph" {
		t.Error("a failed update changed the note")
	}

	// The app's change wins over the web editor, which then sees a conflict.
	web := c.note(t, m)
	patch(m, "", `{"content":"oat milk"}`)
	if rec := s.req("PATCH", "/api/notes/"+web.ID, mustJSON(map[string]any{"version": web.Version, "title": "x"}), withCookie(ann)); rec.Code != http.StatusConflict {
		t.Errorf("web editor after the app: %d %s", rec.Code, rec.Body)
	}

	// Notes outside the scope don't exist for the token.
	rec = s.req("POST", "/api/notes", `{"title":"Elsewhere"}`, withCookie(ann))
	outside := memos.Memo{Name: "memos/" + decodeJSONBody[notes.Note](t, rec).ID}
	wantMemosError(t, patch(outside, "", `{"pinned":true}`), http.StatusNotFound, memos.CodeNotFound)
	wantMemosError(t, c.do("DELETE", memosV1+"/"+outside.Name, ""), http.StatusNotFound, memos.CodeNotFound)

	// Delete.
	if rec := c.do("DELETE", memosV1+"/"+m.Name, ""); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "{}" {
		t.Errorf("delete: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/notes/"+strings.TrimPrefix(m.Name, "memos/"), "", withCookie(ann)); rec.Code != http.StatusNotFound {
		t.Errorf("deleted note: %d", rec.Code)
	}
}

func (c memosClient) upload(t *testing.T, filename string, data []byte) memos.Attachment {
	t.Helper()
	rec := c.do("POST", memosV1+"/attachments", mustJSON(map[string]any{
		"filename": filename, "type": "application/octet-stream", "content": base64.StdEncoding.EncodeToString(data),
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("upload %s: %d %s", filename, rec.Code, rec.Body)
	}
	return decodeJSONBody[memos.Attachment](t, rec)
}

func attachmentNames(list []memos.Attachment) []string {
	out := []string{}
	for _, a := range list {
		out = append(out, a.Filename)
	}
	return out
}

func TestMemosAttachments(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	home := s.noteCategory(t, ann, "Home", "")
	work := s.noteCategory(t, ann, "Work", "")
	c := s.memosToken(t, ann, home)

	// Upload a photo: the type comes from the file, not from the app.
	roses := c.upload(t, "roses.png", testPNG)
	if !strings.HasPrefix(roses.Name, "attachments/") || roses.Filename != "roses.png" || roses.Type != "image/png" ||
		roses.Size != strconv.Itoa(len(testPNG)) || roses.Memo != "" ||
		roses.ExternalLink != testOrigin+"/api/attachments/"+strings.TrimPrefix(roses.Name, "attachments/")+"/roses.png" {
		t.Fatalf("uploaded = %+v", roses)
	}
	if rec := s.req("GET", strings.TrimPrefix(roses.ExternalLink, testOrigin), ""); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), testPNG) {
		t.Errorf("external link: %d", rec.Code)
	}
	plan := c.upload(t, "my plan.pdf", testPDF)
	if !strings.HasSuffix(plan.ExternalLink, "/my%20plan.pdf") {
		t.Errorf("escaped link = %s", plan.ExternalLink)
	}

	// Uploads that fail store nothing.
	before := len(decodeJSONBody[struct{ Attachments []any }](t, s.req("GET", "/api/attachments", "", withCookie(ann))).Attachments)
	for name, body := range map[string]string{
		"too large":   mustJSON(map[string]any{"filename": "big.pdf", "content": base64.StdEncoding.EncodeToString(append([]byte("%PDF-"), make([]byte, 1<<20)...))}),
		"huge body":   mustJSON(map[string]any{"filename": "big.pdf", "content": strings.Repeat("A", 2<<20)}),
		"svg":         mustJSON(map[string]any{"filename": "x.svg", "content": base64.StdEncoding.EncodeToString([]byte("<svg/>"))}),
		"no content":  `{"filename":"a.png"}`,
		"not base64":  `{"filename":"a.png","content":"!!"}`,
		"no filename": mustJSON(map[string]any{"content": base64.StdEncoding.EncodeToString(testPNG)}),
	} {
		rec := c.do("POST", memosV1+"/attachments", body)
		wantMemosError(t, rec, http.StatusBadRequest, memos.CodeInvalidArgument)
		if name == "too large" && !strings.Contains(rec.Body.String(), "1 MB") {
			t.Errorf("too large message = %s", rec.Body)
		}
	}
	if after := len(decodeJSONBody[struct{ Attachments []any }](t, s.req("GET", "/api/attachments", "", withCookie(ann))).Attachments); after != before {
		t.Errorf("failed uploads stored %d files", after-before)
	}
	// URL-safe base64 is read too.
	rec := c.do("POST", memosV1+"/attachments", mustJSON(map[string]any{"filename": "b.png", "content": base64.RawURLEncoding.EncodeToString(testPNG)}))
	if rec.Code != http.StatusOK {
		t.Errorf("URL-safe base64: %d %s", rec.Code, rec.Body)
	}

	// Memo with a photo.
	ref := func(a ...memos.Attachment) []map[string]string {
		out := []map[string]string{}
		for _, x := range a {
			out = append(out, map[string]string{"name": x.Name})
		}
		return out
	}
	m := c.create(t, map[string]any{"content": "Roses are out", "attachments": ref(roses, plan)})
	n := c.note(t, m)
	if m.Content != "Roses are out" || !reflect.DeepEqual(attachmentNames(m.Attachments), []string{"roses.png", "my plan.pdf"}) ||
		m.Attachments[0].Memo != m.Name || len(n.Content.Blocks) != 3 || n.Content.Blocks[2].Type != notes.BlockAttachment {
		t.Fatalf("created memo %+v, note %+v", m, n.Content)
	}
	patch := func(body any) memos.Memo {
		t.Helper()
		rec := c.do("PATCH", memosV1+"/"+m.Name, mustJSON(body))
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %s: %d %s", mustJSON(body), rec.Code, rec.Body)
		}
		return decodeJSONBody[memos.Memo](t, rec)
	}

	// Unchanged memo: MoeMemos sends everything again.
	version := n.Version
	patch(map[string]any{"content": m.Content, "attachments": ref(roses, plan), "pinned": false, "state": "NORMAL"})
	if c.note(t, m).Version != version {
		t.Error("an unchanged memo was a change")
	}
	// Edit text only.
	if got := patch(map[string]any{"content": "# Garden\n\nRoses"}); got.Content != "# Garden\n\nRoses" || len(got.Attachments) != 2 {
		t.Errorf("text only = %+v", got)
	}
	// Remove a photo; the file still exists.
	if got := patch(map[string]any{"content": "# Garden\n\nRoses", "attachments": ref(plan)}); !reflect.DeepEqual(attachmentNames(got.Attachments), []string{"my plan.pdf"}) {
		t.Errorf("removed = %v", attachmentNames(got.Attachments))
	}
	if rec := s.req("GET", strings.TrimPrefix(roses.ExternalLink, testOrigin), ""); rec.Code != http.StatusOK {
		t.Errorf("removed file: %d", rec.Code)
	}
	// Attachment on update.
	if got := patch(map[string]any{"attachments": ref(plan, roses)}); !reflect.DeepEqual(attachmentNames(got.Attachments), []string{"my plan.pdf", "roses.png"}) || got.Content != "# Garden\n\nRoses" {
		t.Errorf("added = %+v", got)
	}
	if n := c.note(t, m); n.Title == nil || *n.Title != "Garden" || len(n.Attachments) != 2 {
		t.Errorf("note after update = %+v", n)
	}

	// Unknown attachments change nothing.
	bobs := s.memosToken(t, bob, "").upload(t, "bob.png", testPNG)
	version = c.note(t, m).Version
	for _, body := range []any{
		map[string]any{"content": "x", "attachments": ref(bobs)},
		map[string]any{"attachments": []map[string]string{{"name": "roses.png"}}},
	} {
		wantMemosError(t, c.do("PATCH", memosV1+"/"+m.Name, mustJSON(body)), http.StatusBadRequest, memos.CodeInvalidArgument)
	}
	wantMemosError(t, c.do("POST", memosV1+"/memos", mustJSON(map[string]any{"content": "x", "attachments": ref(bobs)})), http.StatusBadRequest, memos.CodeInvalidArgument)
	if c.note(t, m).Version != version {
		t.Error("a failed update changed the note")
	}

	// Inline attachment kept: a file with text after it stays in the content.
	rec = s.req("POST", "/api/notes", `{"categoryId":"`+home+`","content":{"v":1,"blocks":[{"type":"attachment","id":"`+strings.TrimPrefix(roses.Name, "attachments/")+`"},{"type":"paragraph","content":[{"type":"text","text":"roses"}]}]}}`, withCookie(ann))
	inline := memos.Memo{Name: "memos/" + decodeJSONBody[notes.Note](t, rec).ID}
	inlineBefore := c.note(t, inline)
	var listed memos.Memo
	for _, x := range c.list(t, "").Memos {
		if x.Name == inline.Name {
			listed = x
		}
	}
	if len(listed.Attachments) != 0 || !strings.Contains(listed.Content, "roses.png") {
		t.Errorf("inline memo = %+v", listed)
	}
	c.do("PATCH", memosV1+"/"+inline.Name, mustJSON(map[string]any{"content": listed.Content, "attachments": []any{}}))
	if after := c.note(t, inline); after.Version != inlineBefore.Version || after.Content.Blocks[0].Type != notes.BlockAttachment {
		t.Errorf("inline after save = %+v", after.Content)
	}

	// The list has what notes in scope refer to: not the unused upload, and
	// not a file of a note in "Work".
	budget := s.mustUploadAttachment(t, ann, "budget.pdf", testPDF)
	if rec := s.req("POST", "/api/notes", `{"categoryId":"`+work+`","content":{"v":1,"blocks":[{"type":"attachment","id":"`+budget.ID+`"}]}}`, withCookie(ann)); rec.Code != http.StatusCreated {
		t.Fatalf("work note: %d %s", rec.Code, rec.Body)
	}
	rec = c.do("GET", memosV1+"/attachments", "")
	list := decodeJSONBody[memos.AttachmentList](t, rec)
	if got := attachmentNames(list.Attachments); rec.Code != http.StatusOK || len(got) != 2 || slices.Contains(got, "budget.pdf") || slices.Contains(got, "b.png") ||
		!strings.Contains(rec.Body.String(), `"nextPageToken":""`) {
		t.Errorf("list = %v (%d)", got, rec.Code)
	}

	// Delete never deletes: a file in scope or unused succeeds, others 404.
	del := func(id string) *httptest.ResponseRecorder { return c.do("DELETE", memosV1+"/attachments/"+id, "") }
	if rec := del(strings.TrimPrefix(roses.Name, "attachments/")); rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "{}" {
		t.Errorf("delete in scope: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", strings.TrimPrefix(roses.ExternalLink, testOrigin), ""); rec.Code != http.StatusOK {
		t.Errorf("deleted file is gone: %d", rec.Code)
	}
	unused := decodeJSONBody[memos.Attachment](t, c.do("POST", memosV1+"/attachments", mustJSON(map[string]any{"filename": "c.png", "content": base64.StdEncoding.EncodeToString(testPNG)})))
	if rec := del(strings.TrimPrefix(unused.Name, "attachments/")); rec.Code != http.StatusOK {
		t.Errorf("delete unused: %d %s", rec.Code, rec.Body)
	}
	wantMemosError(t, del(budget.ID), http.StatusNotFound, memos.CodeNotFound)
	wantMemosError(t, del(strings.TrimPrefix(bobs.Name, "attachments/")), http.StatusNotFound, memos.CodeNotFound)
	wantMemosError(t, del("doesnotexist"), http.StatusNotFound, memos.CodeNotFound)
}

// taskIDsOf joins the task IDs of a note.
func taskIDsOf(t *testing.T, n notes.Note) string {
	t.Helper()
	var ids []string
	for _, b := range n.Content.Blocks {
		if b.Items != nil {
			for _, it := range *b.Items {
				ids = append(ids, it.ID)
			}
		}
	}
	return strings.Join(ids, ",")
}

func escapeQuery(s string) string {
	return strings.NewReplacer("%", "%25", " ", "%20", `"`, "%22", "&", "%26", "[", "%5B", "]", "%5D", ",", "%2C", "=", "%3D").Replace(s)
}
