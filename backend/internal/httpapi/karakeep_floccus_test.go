package httpapi

import (
	"net/http"
	"reflect"
	"testing"
)

// TestKarakeepFloccusFlow replays the calls of Floccus' Karakeep adapter
// (floccusaddon/floccus src/lib/adapters/Karakeep.ts) for a sync that
// creates, edits, moves and removes bookmarks and folders.
func TestKarakeepFloccusFlow(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	c := s.karakeepRESTClient(t, ann)
	listFolder := map[string]any{"icon": "📔", "type": "manual"}
	folder := func(name string, parent string) kkList {
		t.Helper()
		body := map[string]any{"name": name}
		for k, v := range listFolder {
			body[k] = v
		}
		if parent != "" {
			body["parentId"] = parent
		}
		return kkJSON[kkList](c, "POST", "/lists", body, http.StatusCreated)
	}
	tagNames := func(id string) []string {
		t.Helper()
		names := []string{}
		for _, tag := range kkJSON[kkBookmark](c, "GET", "/bookmarks/"+id+"?includeContent=false", nil, http.StatusOK).Tags {
			names = append(names, tag.Name)
		}
		return names
	}
	// removeBookmark: out of its list, then deleted when it's in no list.
	removeBookmark := func(id, parent string) {
		t.Helper()
		kkJSON[any](c, "DELETE", "/lists/"+parent+"/bookmarks/"+id, nil, http.StatusNoContent)
		if lists := kkJSON[kkLists](c, "GET", "/bookmarks/"+id+"/lists", nil, http.StatusOK); len(lists.Lists) == 0 {
			kkJSON[any](c, "DELETE", "/bookmarks/"+id, nil, http.StatusNoContent)
		}
	}

	// Tree load: no lists yet, so Floccus creates its root list.
	if got := kkJSON[kkLists](c, "GET", "/lists", nil, http.StatusOK); len(got.Lists) != 0 {
		t.Fatalf("lists = %+v", got)
	}
	root := folder("Floccus", "")
	if got := kkJSON[kkPage](c, "GET", "/lists/"+root.ID+"/bookmarks?includeContent=false&", nil, http.StatusOK); len(got.Bookmarks) != 0 {
		t.Fatalf("root bookmarks = %+v", got)
	}

	// createFolder, createBookmark (create, add to list, attach tags).
	bar := folder("Bookmarks bar", root.ID)
	b := kkJSON[kkBookmark](c, "POST", "/bookmarks", map[string]any{"type": "link", "url": "https://example.org/docs", "title": "Docs"}, http.StatusCreated)
	kkJSON[any](c, "PUT", "/lists/"+bar.ID+"/bookmarks/"+b.ID, nil, http.StatusNoContent)
	kkJSON[map[string][]string](c, "POST", "/bookmarks/"+b.ID+"/tags", map[string]any{"tags": []map[string]string{{"tagName": "dev"}}}, http.StatusOK)
	if got := s.children(t, bar.ID); !reflect.DeepEqual(got, []string{"Docs"}) {
		t.Fatalf("bar = %v", got)
	}

	// updateBookmark: title, tag diff, then a move whose add arrives before
	// its remove.
	kkJSON[kkBookmark](c, "PATCH", "/bookmarks/"+b.ID, map[string]any{"url": "https://example.org/docs", "title": "Docs (new)"}, http.StatusOK)
	if got := tagNames(b.ID); !reflect.DeepEqual(got, []string{"dev"}) {
		t.Fatalf("tags = %v", got)
	}
	kkJSON[map[string][]string](c, "POST", "/bookmarks/"+b.ID+"/tags", map[string]any{"tags": []map[string]string{{"tagName": "go"}}}, http.StatusOK)
	kkJSON[map[string][]string](c, "DELETE", "/bookmarks/"+b.ID+"/tags", map[string]any{"tags": []map[string]string{{"tagName": "dev"}}}, http.StatusOK)
	if got := tagNames(b.ID); !reflect.DeepEqual(got, []string{"go"}) {
		t.Errorf("tags after diff = %v", got)
	}
	kkJSON[any](c, "PUT", "/lists/"+root.ID+"/bookmarks/"+b.ID, nil, http.StatusNoContent)
	kkJSON[any](c, "DELETE", "/lists/"+bar.ID+"/bookmarks/"+b.ID, nil, http.StatusNoContent)
	if got := s.children(t, root.ID); !reflect.DeepEqual(got, []string{"Docs (new)", "Bookmarks bar"}) {
		t.Errorf("root after move (add first) = %v", got)
	}
	// And the other way round: remove first, then add.
	kkJSON[any](c, "DELETE", "/lists/"+root.ID+"/bookmarks/"+b.ID, nil, http.StatusNoContent)
	kkJSON[any](c, "PUT", "/lists/"+bar.ID+"/bookmarks/"+b.ID, nil, http.StatusNoContent)
	if got := s.children(t, bar.ID); !reflect.DeepEqual(got, []string{"Docs (new)"}) {
		t.Errorf("bar after move (remove first) = %v", got)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus"}) {
		t.Errorf("token root = %v", got)
	}

	// updateFolder: rename and move to the top.
	kkJSON[kkList](c, "PATCH", "/lists/"+bar.ID, map[string]any{"name": "Toolbar", "parentId": nil}, http.StatusOK)
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus", "Toolbar"}) {
		t.Errorf("token root after folder move = %v", got)
	}

	// removeBookmark.
	removeBookmark(b.ID, bar.ID)
	if rec := c.do("GET", "/bookmarks/"+b.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("removed bookmark: %d", rec.Code)
	}

	// removeFolder: bookmarks and sublists first, then the list.
	other := folder("Other", root.ID)
	sub := folder("Sub", other.ID)
	inner := kkJSON[kkBookmark](c, "POST", "/bookmarks", map[string]any{"type": "link", "url": "https://example.org/inner", "title": "Inner"}, http.StatusCreated)
	kkJSON[any](c, "PUT", "/lists/"+sub.ID+"/bookmarks/"+inner.ID, nil, http.StatusNoContent)
	for _, bm := range kkJSON[kkPage](c, "GET", "/lists/"+sub.ID+"/bookmarks?includeContent=false&", nil, http.StatusOK).Bookmarks {
		removeBookmark(bm.ID, sub.ID)
	}
	kkJSON[any](c, "DELETE", "/lists/"+sub.ID, nil, http.StatusNoContent)
	kkJSON[any](c, "DELETE", "/lists/"+other.ID, nil, http.StatusNoContent)
	if got := s.children(t, root.ID); len(got) != 0 {
		t.Errorf("Floccus after removing Other = %v", got)
	}
	if got := s.children(t, c.rootID); !reflect.DeepEqual(got, []string{"Floccus", "Toolbar"}) {
		t.Errorf("token root at the end = %v", got)
	}
}
