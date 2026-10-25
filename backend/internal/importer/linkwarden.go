package importer

import (
	"fmt"
	"io"
	"strings"

	"boardly/internal/favourites"
)

type linkwardenExport struct {
	Collections *[]linkwardenCollection `json:"collections"`
}

type linkwardenCollection struct {
	ID       flexID           `json:"id"`
	Name     optString        `json:"name"`
	ParentID flexID           `json:"parentId"`
	Links    []linkwardenLink `json:"links"`
}

type linkwardenLink struct {
	Name        optString `json:"name"`
	URL         optString `json:"url"`
	Description optString `json:"description"`
	CreatedAt   flexTime  `json:"createdAt"`
	Tags        tagNames  `json:"tags"`
}

// ParseLinkwarden reads a Linkwarden JSON export (its migration backup):
// collections become folders and their links go into them. Links without a
// URL, such as uploaded files, are counted and left out. In every folder, the
// subfolders come first, in the order of the collections, then the links in
// the order of the file.
func ParseLinkwarden(r io.Reader) (favourites.ImportTree, error) {
	var export linkwardenExport
	if err := decodeJSON(r, &export); err != nil || export.Collections == nil {
		return favourites.ImportTree{}, ErrInvalidFile
	}
	b := &builder{}
	collections := *export.Collections

	// A collection without an ID can't be anyone's parent; it still gets a
	// folder of its own, under a key no real ID has.
	keys := make([]flexID, len(collections))
	names := make([]string, len(collections))
	parents := map[flexID]flexID{}
	for i, c := range collections {
		keys[i], names[i] = c.ID, string(c.Name)
		if c.ID == "" {
			keys[i] = flexID(fmt.Sprintf("\x00%d", i))
		}
		parents[keys[i]] = c.ParentID
	}
	folders := treeOf(&b.root, keys, names, parents)

	links := map[*node][]*favourites.ImportLink{}
	for ci, c := range collections {
		folder := folders[keys[ci]]
		for _, ln := range c.Links {
			if strings.TrimSpace(string(ln.URL)) == "" {
				b.report.Skipped.NotLinks++
				continue
			}
			l := b.link(entry{
				url: string(ln.URL), title: string(ln.Name), tags: ln.Tags,
				description: string(ln.Description), createdAt: ln.CreatedAt.time(),
			})
			if l != nil {
				links[folder] = append(links[folder], l)
			}
		}
	}
	addLinks(&b.root, links)
	return b.tree(), nil
}
