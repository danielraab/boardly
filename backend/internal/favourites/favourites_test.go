package favourites

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"boardly/internal/apptokens"
	"boardly/internal/dbtest"
	"boardly/internal/users"
)

type fixture struct {
	t     *testing.T
	s     *Store
	ctx   context.Context
	user  string
	other string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	ctx := context.Background()
	u := users.NewStore(database)
	ann, err := u.EnsureByEmail(ctx, database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.EnsureByEmail(ctx, database, "bob@example.org")
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, s: NewStore(database), ctx: ctx, user: ann.ID, other: bob.ID}
}

func (f *fixture) folder(parentID, name string) string {
	f.t.Helper()
	n, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: parentID, Kind: KindFolder, Title: ptr(name)})
	if err != nil {
		f.t.Fatalf("create folder %q: %v", name, err)
	}
	return n.ID
}

func (f *fixture) link(parentID, title string, tags ...string) string {
	f.t.Helper()
	n, err := f.s.Create(f.ctx, f.user, NewNode{ParentID: parentID, Kind: KindLink, Title: ptr(title), URL: ptr("https://example.org/" + title), Tags: tags})
	if err != nil {
		f.t.Fatalf("create link %q: %v", title, err)
	}
	return n.ID
}

func (f *fixture) tree() *Node {
	f.t.Helper()
	root, err := f.s.Tree(f.ctx, f.user)
	if err != nil {
		f.t.Fatal(err)
	}
	return root
}

// titles returns the titles of the children of the node with the given ID
// (the root for ""), and checks that their stored positions are 0..n-1.
func (f *fixture) titles(id string) []string {
	f.t.Helper()
	n := find(f.tree(), id)
	if n == nil {
		f.t.Fatalf("node %s not in tree", id)
	}
	rows, err := f.s.DB.QueryContext(f.ctx, `SELECT position FROM favourite_nodes WHERE parent_id = ? ORDER BY position`, n.ID)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	for want := 0; rows.Next(); want++ {
		var pos int
		if err := rows.Scan(&pos); err != nil {
			f.t.Fatal(err)
		}
		if pos != want {
			f.t.Fatalf("positions in %s are not contiguous: got %d, want %d", id, pos, want)
		}
	}
	out := []string{}
	for _, c := range n.Children {
		out = append(out, *c.Title)
	}
	return out
}

func find(n *Node, id string) *Node {
	if id == "" || n.ID == id {
		return n
	}
	for _, c := range n.Children {
		if found := find(c, id); found != nil {
			return found
		}
	}
	return nil
}

func (f *fixture) tagCount() int {
	f.t.Helper()
	var n int
	if err := f.s.DB.QueryRow(`SELECT COUNT(*) FROM tags WHERE user_id = ?`, f.user).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func TestEnsureRootOnceConcurrently(t *testing.T) {
	f := newFixture(t)
	ids := make([]string, 8)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Go(func() {
			id, err := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)
			if err != nil {
				t.Error(err)
			}
			ids[i] = id
		})
	}
	wg.Wait()
	for _, id := range ids {
		if id == "" || id != ids[0] {
			t.Fatalf("roots differ: %v", ids)
		}
	}
}

func TestTreeIsNestedAndOrdered(t *testing.T) {
	f := newFixture(t)
	if root := f.tree(); root.Kind != KindFolder || root.Title != nil || len(root.Children) != 0 {
		t.Fatalf("new tree = %+v, want an empty root without title", root)
	}

	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	f.link(work, "Docs", "reading")
	f.link("", "News")
	f.link(tools, "Git")

	if got := f.titles(""); !reflect.DeepEqual(got, []string{"Work", "News"}) {
		t.Fatalf("root = %q", got)
	}
	if got := f.titles(work); !reflect.DeepEqual(got, []string{"Tools", "Docs"}) {
		t.Fatalf("Work = %q", got)
	}
	docs := find(f.tree(), work).Children[1]
	if docs.Kind != KindLink || *docs.URL != "https://example.org/Docs" || !reflect.DeepEqual(docs.Tags, []string{"reading"}) {
		t.Fatalf("Docs = %+v", docs)
	}
}

func TestSubtree(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	f.link(tools, "Git", "code")
	f.folder(tools, "Empty")
	f.link(work, "Docs")
	f.link("", "News")

	sub, err := f.s.Subtree(f.ctx, f.user, work)
	if err != nil {
		t.Fatal(err)
	}
	var shape func(n *Node) string
	shape = func(n *Node) string {
		out := *n.Title
		if n.Kind == KindFolder {
			out += "("
			for i, c := range n.Children {
				if i > 0 {
					out += ","
				}
				out += shape(c)
			}
			out += ")"
		}
		return out
	}
	if got := shape(sub); got != "Work(Tools(Git,Empty()),Docs)" {
		t.Fatalf("subtree = %s", got)
	}
	if git := sub.Children[0].Children[0]; *git.URL != "https://example.org/Git" || git.Tags != nil {
		t.Fatalf("Git = %+v", git)
	}

	root := f.tree().ID
	if sub, err := f.s.Subtree(f.ctx, f.user, root); err != nil || len(sub.Children) != 2 {
		t.Fatalf("root subtree = %+v, %v", sub, err)
	}

	docs := find(f.tree(), work).Children[1].ID
	for name, r := range map[string]struct{ user, id string }{
		"link":           {f.user, docs},
		"unknown":        {f.user, "nope"},
		"another user's": {f.other, work},
	} {
		if _, err := f.s.Subtree(f.ctx, r.user, r.id); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestCreateValidation(t *testing.T) {
	f := newFixture(t)
	link := f.link("", "Docs")
	foreign, err := f.s.EnsureRoot(f.ctx, f.s.DB, f.other)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		n    NewNode
		want error
	}{
		{"folder without name", NewNode{Kind: KindFolder}, ErrInvalidTitle},
		{"folder with url", NewNode{Kind: KindFolder, Title: ptr("F"), URL: ptr("https://example.org")}, ErrInvalidField},
		{"folder with tags", NewNode{Kind: KindFolder, Title: ptr("F"), Tags: []string{"t"}}, ErrInvalidField},
		{"link without url", NewNode{Kind: KindLink}, ErrInvalidURL},
		{"link with bad url", NewNode{Kind: KindLink, URL: ptr("javascript:alert(1)")}, ErrInvalidURL},
		{"link with bad tag", NewNode{Kind: KindLink, URL: ptr("https://example.org"), Tags: []string{" "}}, ErrInvalidTag},
		{"unknown kind", NewNode{Kind: "note"}, ErrInvalidKind},
		{"parent is a link", NewNode{ParentID: link, Kind: KindFolder, Title: ptr("F")}, ErrInvalidParent},
		{"foreign parent", NewNode{ParentID: foreign, Kind: KindFolder, Title: ptr("F")}, ErrNotFound},
		{"missing parent", NewNode{ParentID: "missing", Kind: KindFolder, Title: ptr("F")}, ErrNotFound},
	}
	for _, tt := range tests {
		if _, err := f.s.Create(f.ctx, f.user, tt.n); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
}

func TestTagsAreReusedIgnoringCase(t *testing.T) {
	f := newFixture(t)
	f.link("", "A", "Reading", "Ärger")
	n, err := f.s.Create(f.ctx, f.user, NewNode{Kind: KindLink, URL: ptr("https://example.org/b"), Tags: []string{"reading", "ärger", "new"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(n.Tags, []string{"new", "Reading", "Ärger"}) {
		t.Fatalf("tags = %q, want the stored spellings sorted by name", n.Tags)
	}
	if got := f.tagCount(); got != 3 {
		t.Fatalf("tags = %d, want 3", got)
	}
}

func TestUpdate(t *testing.T) {
	f := newFixture(t)
	work := f.folder("", "Work")
	docs := f.link(work, "Docs", "reading")
	other := f.link(work, "Other", "reading")
	root, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)

	n, err := f.s.Update(f.ctx, f.user, work, Patch{Title: ptr(" Office ")})
	if err != nil || *n.Title != "Office" {
		t.Fatalf("rename = %+v, %v", n, err)
	}

	n, err = f.s.Update(f.ctx, f.user, docs, Patch{Title: ptr("Documentation"), URL: ptr("https://example.org/doc")})
	if err != nil || *n.Title != "Documentation" || *n.URL != "https://example.org/doc" || !reflect.DeepEqual(n.Tags, []string{"reading"}) {
		t.Fatalf("edit link = %+v, %v", n, err)
	}

	n, err = f.s.Update(f.ctx, f.user, docs, Patch{Title: ptr("  ")})
	if err != nil || n.Title != nil {
		t.Fatalf("clear title = %+v, %v", n, err)
	}

	// Removing a tag from one link keeps it on the other; removing it from the last link deletes it.
	if n, err = f.s.Update(f.ctx, f.user, docs, Patch{Tags: &[]string{}}); err != nil || n.Tags != nil {
		t.Fatalf("remove tags = %+v, %v", n, err)
	}
	if f.tagCount() != 1 {
		t.Fatal("tag removed while another link still has it")
	}
	if _, err := f.s.Update(f.ctx, f.user, other, Patch{Tags: &[]string{"old"}}); err != nil {
		t.Fatal(err)
	}
	if tags, _ := f.s.SuggestTags(f.ctx, f.user, ""); !reflect.DeepEqual(tags, []Tag{{ID: tags[0].ID, Name: "old"}}) {
		t.Fatalf("tags = %+v, want only old", tags)
	}

	for name, tt := range map[string]struct {
		id   string
		p    Patch
		want error
	}{
		"root":           {root, Patch{Title: ptr("Root")}, ErrRootImmutable},
		"url on folder":  {work, Patch{URL: ptr("https://example.org")}, ErrInvalidField},
		"tags on folder": {work, Patch{Tags: &[]string{"t"}}, ErrInvalidField},
		"empty folder":   {work, Patch{Title: ptr(" ")}, ErrInvalidTitle},
		"bad url":        {docs, Patch{URL: ptr("example")}, ErrInvalidURL},
		"other user's":   {"missing", Patch{}, ErrNotFound},
	} {
		if _, err := f.s.Update(f.ctx, f.user, tt.id, tt.p); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tt.want)
		}
	}
	if _, err := f.s.Update(f.ctx, f.other, docs, Patch{Title: ptr("x")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("foreign update: err = %v, want ErrNotFound", err)
	}
}

func TestMove(t *testing.T) {
	f := newFixture(t)
	a := f.link("", "A")
	f.link("", "B")
	c := f.link("", "C")
	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	root, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)

	if err := f.s.Move(f.ctx, f.user, c, root, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"C", "A", "B", "Work"}) {
		t.Fatalf("after moving C first: %q", got)
	}
	if err := f.s.Move(f.ctx, f.user, c, root, 2); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"A", "B", "C", "Work"}) {
		t.Fatalf("after moving C to index 2: %q", got)
	}

	// Into another folder, with a position beyond the end.
	if err := f.s.Move(f.ctx, f.user, a, work, 99); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"B", "C", "Work"}) {
		t.Fatalf("root after moving A out: %q", got)
	}
	if got := f.titles(work); !reflect.DeepEqual(got, []string{"Tools", "A"}) {
		t.Fatalf("Work after moving A in: %q", got)
	}

	// A folder moves with its contents.
	old := f.folder("", "Old")
	if err := f.s.Move(f.ctx, f.user, work, old, 0); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(work); !reflect.DeepEqual(got, []string{"Tools", "A"}) {
		t.Fatalf("Work after moving it into Old: %q", got)
	}

	for name, tt := range map[string]struct {
		id, parent string
		want       error
	}{
		"into itself":       {work, work, ErrInvalidMove},
		"into a descendant": {work, tools, ErrInvalidMove},
		"onto a link":       {c, a, ErrInvalidParent},
		"the root":          {root, work, ErrRootImmutable},
		"missing target":    {c, "missing", ErrNotFound},
	} {
		if err := f.s.Move(f.ctx, f.user, tt.id, tt.parent, 0); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tt.want)
		}
	}
	otherRoot, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.other)
	if err := f.s.Move(f.ctx, f.user, c, otherRoot, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("into another user's tree: err = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	f := newFixture(t)
	f.link("", "A")
	work := f.folder("", "Work")
	f.link("", "B")
	tools := f.folder(work, "Tools")
	f.link(tools, "Git", "dev")
	docs := f.link(work, "Docs", "reading")
	root, _ := f.s.EnsureRoot(f.ctx, f.s.DB, f.user)

	if err := f.s.Delete(f.ctx, f.user, work, DeleteKeepNone); !errors.Is(err, ErrFolderNotEmpty) {
		t.Fatalf("non-empty folder without mode: err = %v", err)
	}
	if err := f.s.Delete(f.ctx, f.user, root, DeleteWithChildren); !errors.Is(err, ErrRootImmutable) {
		t.Fatalf("root: err = %v", err)
	}
	if err := f.s.Delete(f.ctx, f.other, docs, DeleteKeepNone); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign: err = %v", err)
	}

	// Keep the contents: they take the folder's place.
	if err := f.s.Delete(f.ctx, f.user, work, DeleteMoveChildren); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"A", "Tools", "Docs", "B"}) {
		t.Fatalf("root = %q, want A, Tools, Docs, B", got)
	}

	// Delete with contents: the subtree and its now unused tags disappear.
	if err := f.s.Delete(f.ctx, f.user, tools, DeleteWithChildren); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(""); !reflect.DeepEqual(got, []string{"A", "Docs", "B"}) {
		t.Fatalf("root = %q, want A, Docs, B", got)
	}
	if tags, _ := f.s.SuggestTags(f.ctx, f.user, ""); len(tags) != 1 || tags[0].Name != "reading" {
		t.Fatalf("tags = %+v, want only reading", tags)
	}

	if err := f.s.Delete(f.ctx, f.user, docs, DeleteKeepNone); err != nil {
		t.Fatal(err)
	}
	if f.tagCount() != 0 {
		t.Fatal("tag kept after deleting its last link")
	}
	if err := f.s.Delete(f.ctx, f.user, docs, DeleteKeepNone); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}
}

func TestSuggestTags(t *testing.T) {
	f := newFixture(t)
	f.link("", "A", "reading", "Recipes", "tools", "100%", "a_b", "ab")

	names := func(prefix string) []string {
		tags, err := f.s.SuggestTags(f.ctx, f.user, prefix)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, tag := range tags {
			out = append(out, tag.Name)
		}
		return out
	}
	if got := names("RE"); !reflect.DeepEqual(got, []string{"reading", "Recipes"}) {
		t.Fatalf("RE = %q", got)
	}
	if got := names("100%"); !reflect.DeepEqual(got, []string{"100%"}) {
		t.Fatalf("100%% = %q", got)
	}
	if got := names("a_"); !reflect.DeepEqual(got, []string{"a_b"}) {
		t.Fatalf("a_ = %q, want the underscore matched literally", got)
	}
	if got := names("%"); len(got) != 0 {
		t.Fatalf("%% = %q, want none", got)
	}

	var many []string
	for i := range 12 {
		many = append(many, string(rune('a'+i))+"x")
	}
	f.link("", "B", many...)
	if got := names(""); len(got) != MaxTagSuggestions {
		t.Fatalf("got %d suggestions, want %d", len(got), MaxTagSuggestions)
	}

	otherTags, err := f.s.SuggestTags(f.ctx, f.other, "")
	if err != nil || len(otherTags) != 0 {
		t.Fatalf("other user's tags = %+v, %v", otherTags, err)
	}
}

func TestDeleteRevokesAppTokens(t *testing.T) {
	f := newFixture(t)
	tokens := apptokens.NewStore(f.s.DB)
	token := func(root string) string {
		t.Helper()
		tok, _, err := tokens.Create(f.ctx, f.user, apptokens.NewToken{Name: "t", App: apptokens.AppKarakeep, RootID: root})
		if err != nil {
			t.Fatal(err)
		}
		return tok.ID
	}
	status := func(id string) (apptokens.Status, *string) {
		t.Helper()
		list, err := tokens.List(f.ctx, f.user)
		if err != nil {
			t.Fatal(err)
		}
		for _, tok := range list {
			if tok.ID == id {
				return tok.Status, tok.RootID
			}
		}
		t.Fatalf("token %s not listed", id)
		return "", nil
	}

	// Keep the contents: only tokens rooted at the deleted folder are revoked.
	work := f.folder("", "Work")
	tools := f.folder(work, "Tools")
	atWork, atTools := token(work), token(tools)
	if err := f.s.Delete(f.ctx, f.user, work, DeleteMoveChildren); err != nil {
		t.Fatal(err)
	}
	if s, root := status(atWork); s != apptokens.StatusRevoked || root != nil {
		t.Errorf("token at deleted folder: %s, root %v; want revoked without root", s, root)
	}
	if s, root := status(atTools); s != apptokens.StatusActive || root == nil || *root != tools {
		t.Errorf("token at moved folder: %s, root %v; want active at Tools", s, root)
	}

	// Delete with contents: tokens anywhere in the subtree are revoked, but not
	// expired ones, and tokens outside stay active.
	reading := f.folder("", "Reading")
	later := f.folder(reading, "Later")
	other := f.folder("", "Other")
	atReading, atLater, expired, outside := token(reading), token(later), token(later), token(other)
	if _, err := f.s.DB.Exec(`UPDATE app_tokens SET expires_at = '2000-01-01T00:00:00.000Z' WHERE id = ?`, expired); err != nil {
		t.Fatal(err)
	}
	if err := f.s.Delete(f.ctx, f.user, reading, DeleteWithChildren); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		id   string
		want apptokens.Status
	}{
		"at deleted folder": {atReading, apptokens.StatusRevoked},
		"below it":          {atLater, apptokens.StatusRevoked},
		"expired below it":  {expired, apptokens.StatusExpired},
		"outside":           {outside, apptokens.StatusActive},
	} {
		if s, _ := status(tc.id); s != tc.want {
			t.Errorf("%s: status %s, want %s", name, s, tc.want)
		}
	}
}
