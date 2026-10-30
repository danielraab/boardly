package importer

import (
	"encoding/json"
	"io"
	"strings"

	"boardly/internal/favourites"
	"boardly/internal/notes"
)

type karakeepExport struct {
	Bookmarks *[]karakeepBookmark `json:"bookmarks"`
	Lists     []karakeepList      `json:"lists"`
}

type karakeepList struct {
	ID       flexID    `json:"id"`
	Name     optString `json:"name"`
	ParentID flexID    `json:"parentId"`
	Type     optString `json:"type"`
}

type karakeepBookmark struct {
	CreatedAt flexTime  `json:"createdAt"`
	Title     optString `json:"title"`
	Tags      tagNames  `json:"tags"`
	Lists     []flexID  `json:"lists"`
	Content   struct {
		Type optString `json:"type"`
		URL  optString `json:"url"`
		Text optString `json:"text"`
	} `json:"content"`
	Note     optString `json:"note"`
	Summary  optString `json:"summary"`
	Archived optBool   `json:"archived"`
}

// ParseKarakeep reads a Karakeep JSON export: manual lists become folders,
// smart lists and bookmarks that are neither links nor text are counted and
// left out, and a bookmark goes into its first manual list. In every folder,
// the subfolders come first, in the order of the lists, then the links in the
// order of the bookmarks. Text bookmarks become notes, and the manual lists
// that hold notes, at any depth, become categories (see importCategory).
func ParseKarakeep(r io.Reader) (Result, error) {
	var export karakeepExport
	if err := decodeJSON(r, &export); err != nil || export.Bookmarks == nil {
		return Result{}, ErrInvalidFile
	}
	b := &builder{}

	lists := map[flexID]*karakeepList{}
	for i := range export.Lists {
		l := &export.Lists[i]
		if l.Type == "smart" {
			b.report.Skipped.SmartLists++
			continue
		}
		if l.ID != "" {
			lists[l.ID] = l
		}
	}
	var ids []flexID
	var names []string
	parents := map[flexID]flexID{}
	for _, l := range export.Lists {
		if _, ok := lists[l.ID]; ok {
			ids, names = append(ids, l.ID), append(names, string(l.Name))
			parents[l.ID] = l.ParentID
		}
	}
	folders := treeOf(&b.root, ids, names, parents)

	categories := categoryTreeOf(ids, names, parents)
	var tooLarge int

	links := map[*node][]*favourites.ImportLink{}
	for _, bm := range *export.Bookmarks {
		if bm.Content.Type == "text" {
			n, ok := karakeepNote(bm)
			if !ok {
				tooLarge++
				continue
			}
			target := categories.root
			for _, id := range bm.Lists {
				if c, ok := categories.byID[id]; ok {
					target = c
					break
				}
			}
			target.notes = append(target.notes, n)
			continue
		}
		if bm.Content.Type != "link" {
			b.report.Skipped.NotLinks++
			continue
		}
		l := b.link(entry{
			url: string(bm.Content.URL), title: string(bm.Title), tags: bm.Tags,
			note: string(bm.Note), summary: string(bm.Summary), createdAt: bm.CreatedAt.time(),
		})
		if l == nil {
			continue
		}
		target := &b.root
		for _, id := range bm.Lists {
			if f, ok := folders[id]; ok {
				target = f
				break
			}
		}
		links[target] = append(links[target], l)
	}
	addLinks(&b.root, links)
	root, _ := categories.root.importCategory()
	return Result{Links: b.tree(), Notes: notes.ImportTree{Root: root, TooLarge: tooLarge}}, nil
}

// karakeepNote converts a text bookmark. ok is false when its content is
// larger than a note can hold.
func karakeepNote(bm karakeepBookmark) (n notes.ImportNote, ok bool) {
	if title := cut(strings.TrimSpace(string(bm.Title)), notes.MaxNoteTitle, ""); title != "" {
		n.Title = &title
	}
	n.CreatedAt = bm.CreatedAt.time()
	n.Archived = bool(bm.Archived)
	n.Content = notes.FromMarkdown(string(bm.Content.Text), nil)
	for _, extra := range []struct{ prefix, text string }{{"Note: ", string(bm.Note)}, {"Summary: ", string(bm.Summary)}} {
		if t := strings.TrimSpace(extra.text); t != "" {
			n.Content.Blocks = append(n.Content.Blocks, notes.FromText(extra.prefix+t).Blocks...)
		}
	}
	var tags []string
	for _, tag := range cleanTags(bm.Tags) {
		tags = append(tags, "#"+strings.Join(strings.Fields(tag), "-"))
	}
	if len(tags) > 0 {
		n.Content.Blocks = append(n.Content.Blocks, notes.FromText(strings.Join(tags, " ")).Blocks...)
	}
	// Parsing assigns the task IDs, which count towards the size.
	raw, err := json.Marshal(n.Content)
	if err != nil {
		return n, false
	}
	if n.Content, err = notes.ParseDoc(raw); err != nil {
		return n, false
	}
	return n, true
}

// categoryTree is the category of each manual list while a file is read.
type categoryTree struct {
	root *category
	byID map[flexID]*category
}

type category struct {
	name     string
	children []*category
	notes    []notes.ImportNote
}

// categoryTreeOf builds a category for every list ID like treeOf builds
// folders: same order, parents and handling of repeated IDs and cycles.
func categoryTreeOf(ids []flexID, names []string, parents map[flexID]flexID) categoryTree {
	t := categoryTree{root: &category{}, byID: map[flexID]*category{}}
	var order []flexID
	for i, id := range ids {
		if _, dup := t.byID[id]; dup {
			continue
		}
		t.byID[id] = &category{name: cut(strings.TrimSpace(names[i]), notes.MaxCategoryTitle, "")}
		order = append(order, id)
	}
	known := func(id flexID) bool { _, ok := t.byID[id]; return ok }
	for _, id := range order {
		parent := t.root
		if p, ok := t.byID[parents[id]]; ok && !cycles(id, parents, known) {
			parent = p
		}
		parent.children = append(parent.children, t.byID[id])
	}
	return t
}

// importCategory converts c, leaving out subcategories without notes at any
// depth and replacing unnamed ones by their contents. ok reports whether c
// holds a note.
func (c *category) importCategory() (out notes.ImportCategory, ok bool) {
	out = notes.ImportCategory{Name: c.name, Notes: c.notes}
	for _, child := range c.children {
		sub, ok := child.importCategory()
		switch {
		case !ok:
		case sub.Name == "":
			out.Notes = append(out.Notes, sub.Notes...)
			out.Categories = append(out.Categories, sub.Categories...)
		default:
			out.Categories = append(out.Categories, sub)
		}
	}
	return out, len(out.Notes) > 0 || len(out.Categories) > 0
}

// treeOf adds a folder below root for every ID, with the name of the same
// index, in this order; a repeated ID is left out. A folder's parent is the
// folder of the ID in parents, or root when that ID isn't a folder or the
// parents form a cycle. It returns the folders by ID.
func treeOf(root *node, ids []flexID, names []string, parents map[flexID]flexID) map[flexID]*node {
	folders := map[flexID]*node{}
	var order []flexID
	for i, id := range ids {
		if _, dup := folders[id]; dup {
			continue
		}
		folders[id] = &node{name: folderName(names[i])}
		order = append(order, id)
	}
	known := func(id flexID) bool { _, ok := folders[id]; return ok }
	for _, id := range order {
		parent := root
		if p, ok := folders[parents[id]]; ok && !cycles(id, parents, known) {
			parent = p
		}
		parent.addFolder(folders[id])
	}
	return folders
}

// cycles reports whether following the parents from id, while they are
// known, leads back to id.
func cycles(id flexID, parents map[flexID]flexID, known func(flexID) bool) bool {
	seen := map[flexID]bool{id: true}
	for p := parents[id]; p != ""; p = parents[p] {
		if !known(p) {
			return false
		}
		if seen[p] {
			return true
		}
		seen[p] = true
	}
	return false
}

// addLinks appends each folder's links after its subfolders.
func addLinks(n *node, links map[*node][]*favourites.ImportLink) {
	for _, it := range n.items {
		if it.folder != nil {
			addLinks(it.folder, links)
		}
	}
	for _, l := range links[n] {
		n.addLink(l)
	}
}
