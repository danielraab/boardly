package importer

import (
	"io"

	"boardly/internal/favourites"
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
	} `json:"content"`
	Note    optString `json:"note"`
	Summary optString `json:"summary"`
}

// ParseKarakeep reads a Karakeep JSON export: manual lists become folders,
// smart lists and bookmarks that aren't links are counted and left out, and
// a bookmark goes into its first manual list. In every folder, the subfolders
// come first, in the order of the lists, then the links in the order of the
// bookmarks.
func ParseKarakeep(r io.Reader) (favourites.ImportTree, error) {
	var export karakeepExport
	if err := decodeJSON(r, &export); err != nil || export.Bookmarks == nil {
		return favourites.ImportTree{}, ErrInvalidFile
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

	links := map[*node][]*favourites.ImportLink{}
	for _, bm := range *export.Bookmarks {
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
	return b.tree(), nil
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
	for _, id := range order {
		parent := root
		if p, ok := folders[parents[id]]; ok && !cycles(id, parents, folders) {
			parent = p
		}
		parent.addFolder(folders[id])
	}
	return folders
}

// cycles reports whether following the parents from id leads back to id.
func cycles(id flexID, parents map[flexID]flexID, folders map[flexID]*node) bool {
	seen := map[flexID]bool{id: true}
	for p := parents[id]; p != ""; p = parents[p] {
		if _, ok := folders[p]; !ok {
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
