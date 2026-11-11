package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/groups"
	"boardly/internal/jobs"
)

func (s *server) deliverInvitations(t *testing.T) {
	t.Helper()
	rows, err := s.db.Query(`SELECT id, payload FROM jobs WHERE kind = 'invitation_mail' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	type job struct {
		id      int64
		payload string
	}
	var list []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.id, &j.payload); err != nil {
			t.Fatal(err)
		}
		list = append(list, j)
	}
	rows.Close()
	for _, j := range list {
		if err := s.invitations.RunMailJob(context.Background(), jobs.Job{ID: j.id, Payload: json.RawMessage(j.payload)}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec(`DELETE FROM jobs WHERE id = ?`, j.id); err != nil {
			t.Fatal(err)
		}
	}
}

func (s *server) createGroup(t *testing.T, cookie, name string) groups.Group {
	t.Helper()
	rec := s.req("POST", "/api/groups", `{"name":"`+name+`"}`, withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: %d %s", rec.Code, rec.Body)
	}
	return decodeJSONBody[groups.Group](t, rec)
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || errorCode(t, rec) != code {
		t.Errorf("got %d %s, want %d %s", rec.Code, rec.Body, status, code)
	}
}

func (s *server) inviteDash(t *testing.T, cookie, dashboardID, email string) map[string]any {
	t.Helper()
	rec := s.req("POST", "/api/shares/dashboard/"+dashboardID+"/invitations",
		`{"email":"`+email+`","level":"read","locale":"en"}`, withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite %s: %d %s", email, rec.Code, rec.Body)
	}
	return decodeJSONBody[map[string]any](t, rec)
}

var invitationLinkRE = regexp.MustCompile(`/invitation\?token=([A-Za-z0-9_-]+)`)

func (s *server) invitationToken(t *testing.T) string {
	t.Helper()
	m := invitationLinkRE.FindStringSubmatch(s.mail.last(t).Text)
	if m == nil {
		t.Fatalf("no invitation link in %q", s.mail.last(t).Text)
	}
	return m[1]
}

// addMember makes the signed-in cookie a member of the group, through an
// accepted invitation.
func (s *server) addMember(t *testing.T, admin string, group groups.Group, member, email, role string) {
	t.Helper()
	rec := s.req("POST", "/api/groups/"+group.ID+"/invitations", `{"email":"`+email+`","role":"`+role+`","locale":"en"}`, withCookie(admin))
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite to group: %d %s", rec.Code, rec.Body)
	}
	id := decodeJSONBody[map[string]any](t, rec)["id"].(string)
	if rec := s.req("POST", "/api/invitations/accept", `{"invitationId":"`+id+`"}`, withCookie(member)); rec.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", rec.Code, rec.Body)
	}
}

func (s *server) shareDirectly(t *testing.T, owner, viewer, dashboardID, email string) {
	t.Helper()
	id := s.inviteDash(t, owner, dashboardID, email)["id"].(string)
	if rec := s.req("POST", "/api/invitations/accept", `{"invitationId":"`+id+`"}`, withCookie(viewer)); rec.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", rec.Code, rec.Body)
	}
}

func TestSharingRoutesRequireSession(t *testing.T) {
	s := newServer(t)
	for _, r := range [][2]string{
		{"GET", "/api/groups"},
		{"POST", "/api/groups"},
		{"GET", "/api/groups/x"},
		{"PATCH", "/api/groups/x"},
		{"DELETE", "/api/groups/x"},
		{"PATCH", "/api/groups/x/members/y"},
		{"DELETE", "/api/groups/x/members/y"},
		{"POST", "/api/groups/x/invitations"},
		{"DELETE", "/api/groups/x/invitations/y"},
		{"GET", "/api/shares/dashboard/x"},
		{"POST", "/api/shares/dashboard/x"},
		{"DELETE", "/api/shares/dashboard/x/y"},
		{"POST", "/api/shares/dashboard/x/invitations"},
		{"DELETE", "/api/shares/dashboard/x/invitations/y"},
		{"GET", "/api/shared-with-me"},
		{"GET", "/api/invitations"},
		{"POST", "/api/invitations/preview"},
		{"POST", "/api/invitations/accept"},
	} {
		if rec := s.req(r[0], r[1], `{}`); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: %d, want 401", r[0], r[1], rec.Code)
		}
	}
}

func TestGroupsAPI(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, bob, "bob")

	g := s.createGroup(t, ann, "Family")
	if g.Role != "admin" || g.MemberCount != 1 {
		t.Fatalf("group %+v", g)
	}
	wantError(t, s.req("POST", "/api/groups", `{"name":" "}`, withCookie(ann)), http.StatusBadRequest, "invalid_name")
	wantError(t, s.req("GET", "/api/groups/"+g.ID, "", withCookie(bob)), http.StatusNotFound, "not_found")

	s.addMember(t, ann, g, bob, "bob@example.org", "member")

	rec := s.req("GET", "/api/groups", "", withCookie(bob))
	list := decodeJSONBody[[]groups.Group](t, rec)
	if len(list) != 1 || list[0].Role != "member" || list[0].MemberCount != 2 {
		t.Fatalf("bob's groups %+v", list)
	}

	rec = s.req("GET", "/api/groups/"+g.ID, "", withCookie(bob))
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "@example.org") {
		t.Errorf("member view contains an address: %s", rec.Body)
	}
	if strings.Contains(rec.Body.String(), `"invitations"`) {
		t.Errorf("member view contains invitations: %s", rec.Body)
	}

	wantError(t, s.req("PATCH", "/api/groups/"+g.ID, `{"name":"X"}`, withCookie(bob)), http.StatusForbidden, "forbidden")
	wantError(t, s.req("DELETE", "/api/groups/"+g.ID, "", withCookie(bob)), http.StatusForbidden, "forbidden")
	wantError(t, s.req("POST", "/api/groups/"+g.ID+"/invitations", `{"email":"x@example.org"}`, withCookie(bob)), http.StatusForbidden, "forbidden")

	details := decodeJSONBody[groups.Details](t, s.req("GET", "/api/groups/"+g.ID, "", withCookie(ann)))
	var annID, bobID string
	for _, m := range details.Members {
		if m.Slug == "bob" {
			bobID = m.ID
		} else {
			annID = m.ID
		}
	}

	wantError(t, s.req("DELETE", "/api/groups/"+g.ID+"/members/"+annID, "", withCookie(bob)), http.StatusForbidden, "forbidden")
	wantError(t, s.req("DELETE", "/api/groups/"+g.ID+"/members/"+annID, "", withCookie(ann)), http.StatusConflict, "last_admin")
	wantError(t, s.req("PATCH", "/api/groups/"+g.ID+"/members/"+annID, `{"role":"member"}`, withCookie(ann)), http.StatusConflict, "last_admin")
	wantError(t, s.req("PATCH", "/api/groups/"+g.ID+"/members/"+bobID, `{"role":"owner"}`, withCookie(ann)), http.StatusBadRequest, "invalid_role")
	wantError(t, s.req("PATCH", "/api/groups/"+g.ID+"/members/nobody", `{"role":"member"}`, withCookie(ann)), http.StatusNotFound, "not_found")

	if rec := s.req("PATCH", "/api/groups/"+g.ID+"/members/"+bobID, `{"role":"admin"}`, withCookie(ann)); rec.Code != http.StatusOK {
		t.Fatalf("promote: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("PATCH", "/api/groups/"+g.ID, `{"name":"Clan"}`, withCookie(bob))
	if rec.Code != http.StatusOK || decodeJSONBody[groups.Group](t, rec).Name != "Clan" {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", "/api/groups/"+g.ID+"/members/"+annID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("leave: %d %s", rec.Code, rec.Body)
	}
	wantError(t, s.req("GET", "/api/groups/"+g.ID, "", withCookie(ann)), http.StatusNotFound, "not_found")
	if rec := s.req("DELETE", "/api/groups/"+g.ID, "", withCookie(bob)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
}

func TestSharesAPI(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	carol := s.signIn(t, "carol@example.org")
	s.setSlug(t, ann, "ann")
	s.setSlug(t, bob, "bob")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	path := "/api/shares/dashboard/" + work.ID
	g := s.createGroup(t, ann, "Family")

	wantError(t, s.req("GET", "/api/shares/note/x", "", withCookie(ann)), http.StatusNotFound, "not_found")
	wantError(t, s.req("GET", path, "", withCookie(bob)), http.StatusNotFound, "not_found")
	wantError(t, s.req("POST", path, `{"groupId":"`+g.ID+`","level":"edit"}`, withCookie(ann)), http.StatusBadRequest, "invalid_share_level")
	wantError(t, s.req("POST", path, `{"groupId":"nope","level":"read"}`, withCookie(ann)), http.StatusNotFound, "group_not_found")
	foreign := s.createGroup(t, carol, "Other")
	wantError(t, s.req("POST", path, `{"groupId":"`+foreign.ID+`","level":"read"}`, withCookie(ann)), http.StatusNotFound, "group_not_found")

	rec := s.req("POST", path, `{"groupId":"`+g.ID+`","level":"read"}`, withCookie(ann))
	if rec.Code != http.StatusCreated {
		t.Fatalf("share: %d %s", rec.Code, rec.Body)
	}
	wantError(t, s.req("POST", path, `{"groupId":"`+g.ID+`","level":"read"}`, withCookie(ann)), http.StatusConflict, "already_shared")

	s.shareDirectly(t, ann, bob, work.ID, "bob@example.org")

	rec = s.req("GET", path, "", withCookie(ann))
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "bob@example.org") {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var body struct {
		Shares []struct {
			ID    string
			Level string
			User  *struct{ Slug string }
			Group *struct{ Name string }
		}
		Invitations []any
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Shares) != 2 || body.Shares[0].User == nil || body.Shares[0].User.Slug != "bob" || body.Shares[1].Group == nil || body.Shares[1].Group.Name != "Family" {
		t.Fatalf("shares %s", rec.Body)
	}
	if body.Invitations == nil || len(body.Invitations) != 0 {
		t.Errorf("invitations %v", body.Invitations)
	}

	// A viewer can't manage the dashboard or its shares.
	wantError(t, s.req("GET", path, "", withCookie(bob)), http.StatusNotFound, "not_found")
	wantError(t, s.req("DELETE", path+"/"+body.Shares[0].ID, "", withCookie(bob)), http.StatusNotFound, "not_found")
	wantError(t, s.req("PATCH", "/api/dashboards/"+work.ID, `{"title":"Mine"}`, withCookie(bob)), http.StatusNotFound, "not_found")
	wantError(t, s.req("PUT", "/api/dashboards/"+work.ID+"/layout", `{"version":0,"sections":[]}`, withCookie(bob)), http.StatusNotFound, "not_found")
	wantError(t, s.req("POST", path+"/invitations", `{"email":"x@example.org","level":"read"}`, withCookie(bob)), http.StatusNotFound, "not_found")
	if list := decodeJSONBody[[]dashboards.Dashboard](t, s.req("GET", "/api/dashboards", "", withCookie(bob))); len(list) != 0 {
		t.Errorf("bob's dashboards %+v", list)
	}

	if rec := s.req("GET", "/api/u/ann/work", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Fatalf("bob view: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("DELETE", path+"/"+body.Shares[0].ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("remove: %d %s", rec.Code, rec.Body)
	}
	wantError(t, s.req("DELETE", path+"/"+body.Shares[0].ID, "", withCookie(ann)), http.StatusNotFound, "not_found")
	wantError(t, s.req("GET", "/api/u/ann/work", "", withCookie(bob)), http.StatusNotFound, "not_found")
}

func TestSharedWithMeAPI(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, ann, "ann")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	s.createDashboard(t, ann, map[string]any{"title": "Home", "slug": "home", "isPublic": true})
	s.shareDirectly(t, ann, bob, work.ID, "bob@example.org")
	g := s.createGroup(t, ann, "Family")
	s.addMember(t, ann, g, bob, "bob@example.org", "member")
	if rec := s.req("POST", "/api/shares/dashboard/"+work.ID, `{"groupId":"`+g.ID+`","level":"read"}`, withCookie(ann)); rec.Code != http.StatusCreated {
		t.Fatalf("share: %d %s", rec.Code, rec.Body)
	}

	rec := s.req("GET", "/api/shared-with-me", "", withCookie(bob))
	var body struct{ Dashboards []map[string]any }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Dashboards) != 1 || body.Dashboards[0]["slug"] != "work" || body.Dashboards[0]["ownerSlug"] != "ann" || body.Dashboards[0]["level"] != "read" {
		t.Fatalf("shared %s", rec.Body)
	}
	if rec := s.req("GET", "/api/shared-with-me", "", withCookie(ann)); !strings.Contains(rec.Body.String(), `"dashboards":[]`) {
		t.Errorf("ann's own dashboard is listed: %s", rec.Body)
	}
}

func TestInvitationSameResultForEveryAddress(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	s.shareDirectly(t, ann, bob, work.ID, "bob@example.org") // Bob already has access

	s.signIn(t, "carol@example.org") // an account without any access
	path := "/api/shares/dashboard/" + work.ID + "/invitations"
	responses := map[string]*httptest.ResponseRecorder{}
	addresses := []string{"carol@example.org", "unknown@example.org", "bob@example.org"}
	for _, address := range addresses {
		responses[address] = s.req("POST", path, `{"email":"`+address+`","level":"read","locale":"en"}`, withCookie(ann))
	}
	normalize := func(rec *httptest.ResponseRecorder, address string) map[string]any {
		m := decodeJSONBody[map[string]any](t, rec)
		if m["email"] != address {
			t.Errorf("email = %v, want %s", m["email"], address)
		}
		delete(m, "id")
		delete(m, "email")
		return m
	}
	want := normalize(responses[addresses[0]], addresses[0])
	headers := responses[addresses[0]].Header()
	for _, address := range addresses {
		rec := responses[address]
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s: %d %s", address, rec.Code, rec.Body)
		}
		if got := normalize(rec, address); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s: %v, want %v", address, got, want)
		}
		if len(rec.Header()) != len(headers) {
			t.Errorf("%s: headers %v, want %v", address, rec.Header(), headers)
		}
	}

	rec := s.req("GET", "/api/shares/dashboard/"+work.ID, "", withCookie(ann))
	var body struct{ Invitations []map[string]any }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Invitations) != 3 {
		t.Fatalf("listed %s", rec.Body)
	}
	for _, inv := range body.Invitations {
		var keys []string
		for k := range inv {
			keys = append(keys, k)
		}
		if len(keys) != 5 {
			t.Errorf("entry %v has fields beyond id, email, level, createdAt and expiresAt", inv)
		}
	}

	before := s.mail.count()
	s.deliverInvitations(t)
	if got := s.mail.count() - before; got != 3 {
		t.Errorf("mails = %d, want 3", got)
	}
}

func TestInvitationFlow(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	carol := s.signIn(t, "carol@example.org")
	s.setSlug(t, ann, "ann")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	path := "/api/shares/dashboard/" + work.ID + "/invitations"

	wantError(t, s.req("POST", path, `{"email":"nope","level":"read"}`, withCookie(ann)), http.StatusBadRequest, "invalid_email")
	wantError(t, s.req("POST", path, `{"email":"x@example.org","level":"edit"}`, withCookie(ann)), http.StatusBadRequest, "invalid_share_level")
	wantError(t, s.req("POST", path, `{"email":"ann@example.org","level":"read"}`, withCookie(ann)), http.StatusBadRequest, "cannot_invite_self")

	inv := s.inviteDash(t, ann, work.ID, "bob@example.org")
	wantError(t, s.req("POST", path, `{"email":"bob@example.org","level":"read"}`, withCookie(ann)), http.StatusConflict, "already_invited")
	wantError(t, s.req("GET", "/api/u/ann/work", "", withCookie(bob)), http.StatusNotFound, "not_found")

	s.deliverInvitations(t)
	token := s.invitationToken(t)

	wantError(t, s.req("POST", "/api/invitations/preview", `{"token":"`+token+`"}`, withCookie(carol)), http.StatusForbidden, "invitation_wrong_account")
	wantError(t, s.req("POST", "/api/invitations/accept", `{"token":"`+token+`"}`, withCookie(carol)), http.StatusForbidden, "invitation_wrong_account")
	wantError(t, s.req("POST", "/api/invitations/accept", `{"invitationId":"`+inv["id"].(string)+`"}`, withCookie(carol)), http.StatusNotFound, "invitation_not_found")
	wantError(t, s.req("POST", "/api/invitations/preview", `{"token":"nope"}`, withCookie(bob)), http.StatusNotFound, "invitation_not_found")
	wantError(t, s.req("POST", "/api/invitations/accept", `{}`, withCookie(bob)), http.StatusBadRequest, "invalid_request")
	wantError(t, s.req("POST", "/api/invitations/accept", `{"token":"a","invitationId":"b"}`, withCookie(bob)), http.StatusBadRequest, "invalid_request")

	rec := s.req("POST", "/api/invitations/preview", `{"token":"`+token+`"}`, withCookie(bob))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"title":"Work"`) || !strings.Contains(rec.Body.String(), `"type":"dashboard"`) {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body)
	}

	rec = s.req("GET", "/api/invitations", "", withCookie(bob))
	if !strings.Contains(rec.Body.String(), `"title":"Work"`) {
		t.Fatalf("open invitations: %s", rec.Body)
	}
	if rec := s.req("GET", "/api/invitations", "", withCookie(carol)); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("carol's invitations: %s", rec.Body)
	}

	rec = s.req("POST", "/api/invitations/accept", `{"token":"`+token+`"}`, withCookie(bob))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ownerSlug":"ann"`) || !strings.Contains(rec.Body.String(), `"slug":"work"`) {
		t.Fatalf("accept: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/u/ann/work", "", withCookie(bob)); rec.Code != http.StatusOK {
		t.Fatalf("view after accepting: %d %s", rec.Code, rec.Body)
	}
	wantError(t, s.req("POST", "/api/invitations/accept", `{"token":"`+token+`"}`, withCookie(bob)), http.StatusNotFound, "invitation_not_found")

	second := s.inviteDash(t, ann, work.ID, "new@example.org")
	if rec := s.req("DELETE", path+"/"+second["id"].(string), "", withCookie(bob)); rec.Code != http.StatusNotFound {
		t.Errorf("viewer revoke: %d", rec.Code)
	}
	if rec := s.req("DELETE", path+"/"+second["id"].(string), "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body)
	}
}

func TestInvitationBeforeRegistration(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	s.setSlug(t, ann, "ann")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	inv := s.inviteDash(t, ann, work.ID, "new@example.org")
	s.deliverInvitations(t)
	notice := s.mail.last(t)
	if notice.To != "new@example.org" || strings.Contains(notice.Text, "Work") || strings.Contains(notice.Text, "/invitation") {
		t.Fatalf("notice %+v", notice)
	}

	s.clock.Advance(24 * time.Hour)
	fresh := s.signIn(t, "new@example.org")
	rec := s.req("GET", "/api/invitations", "", withCookie(fresh))
	if !strings.Contains(rec.Body.String(), `"title":"Work"`) || !strings.Contains(rec.Body.String(), `"inviterName"`) {
		t.Fatalf("open invitations: %s", rec.Body)
	}
	wantError(t, s.req("GET", "/api/u/ann/work", "", withCookie(fresh)), http.StatusNotFound, "not_found")
	if rec := s.req("POST", "/api/invitations/accept", `{"invitationId":"`+inv["id"].(string)+`"}`, withCookie(fresh)); rec.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("GET", "/api/u/ann/work", "", withCookie(fresh)); rec.Code != http.StatusOK {
		t.Fatalf("view: %d %s", rec.Code, rec.Body)
	}
}

func TestInvitationLimit(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	path := "/api/shares/dashboard/" + work.ID + "/invitations"
	for i := range invitationsPerHour {
		s.inviteDash(t, ann, work.ID, fmt.Sprintf("x%d@example.org", i))
	}
	rec := s.req("POST", path, `{"email":"more@example.org","level":"read"}`, withCookie(ann))
	wantError(t, rec, http.StatusTooManyRequests, "rate_limited")
	if rec.Header().Get("Retry-After") == "" {
		t.Error("no Retry-After header")
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE kind = 'invitation_mail'`).Scan(&n); err != nil || n != invitationsPerHour {
		t.Errorf("jobs = %d, %v", n, err)
	}
	s.clock.Advance(time.Hour + time.Minute)
	s.inviteDash(t, ann, work.ID, "more@example.org")
}

func TestGroupInvitationsListedForAdminsOnly(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	g := s.createGroup(t, ann, "Family")
	s.addMember(t, ann, g, bob, "bob@example.org", "member")

	rec := s.req("POST", "/api/groups/"+g.ID+"/invitations", `{"email":"new@example.org"}`, withCookie(ann))
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"role":"member"`) {
		t.Fatalf("invite: %d %s", rec.Code, rec.Body)
	}
	id := decodeJSONBody[map[string]any](t, rec)["id"].(string)

	if rec := s.req("GET", "/api/groups/"+g.ID, "", withCookie(ann)); !strings.Contains(rec.Body.String(), "new@example.org") {
		t.Errorf("admin view: %s", rec.Body)
	}
	if rec := s.req("GET", "/api/groups/"+g.ID, "", withCookie(bob)); strings.Contains(rec.Body.String(), "new@example.org") {
		t.Errorf("member view: %s", rec.Body)
	}
	wantError(t, s.req("DELETE", "/api/groups/"+g.ID+"/invitations/"+id, "", withCookie(bob)), http.StatusForbidden, "forbidden")
	if rec := s.req("DELETE", "/api/groups/"+g.ID+"/invitations/"+id, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body)
	}
}

func TestSharedViewerSeesLikeAVisitor(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	s.setSlug(t, ann, "ann")
	work := s.createDashboard(t, ann, map[string]any{"title": "Work", "slug": "work"})
	note := s.req("POST", "/api/notes", `{"title":"Note","content":{"v":1,"blocks":[{"type":"paragraph","content":[{"type":"text","text":"secret text"}]}]}}`, withCookie(ann))
	if note.Code != http.StatusCreated {
		t.Fatalf("note: %d %s", note.Code, note.Body)
	}
	noteID := decodeJSONBody[map[string]any](t, note)["id"].(string)
	rec := s.req("PUT", "/api/dashboards/"+work.ID+"/layout", `{"version":0,"sections":[{"kind":"section","width":6,"children":[
		{"kind":"note","width":2,"config":{"noteId":"`+noteID+`","size":"M","showIfPublic":false}},
		{"kind":"link","width":2,"config":{"source":"url","url":"https://example.org","title":"Link","target":"_blank","size":"M"}}]}]}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("layout: %d %s", rec.Code, rec.Body)
	}
	s.shareDirectly(t, ann, bob, work.ID, "bob@example.org")

	rec = s.req("GET", "/api/u/ann/work", "", withCookie(bob))
	if rec.Code != http.StatusOK {
		t.Fatalf("view: %d %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	if strings.Contains(body, `"kind":"note"`) || strings.Contains(body, "secret text") {
		t.Errorf("widget with showIfPublic off is visible: %s", body)
	}
	if !strings.Contains(body, `"isOwner":false`) || strings.Contains(body, `"isPublic"`) || strings.Contains(body, `"id":"`+work.ID+`"`) {
		t.Errorf("owner fields in %s", body)
	}
}
