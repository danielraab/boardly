package httpapi

import (
	"net/http"

	"boardly/internal/notes"
)

// legalPage is the content of a legal page in one language.
type legalPage struct {
	Page string `json:"page"`
	// Language is that of the file the content came from; nil for the
	// fallback <page>.md.
	Language *string   `json:"language"`
	Doc      notes.Doc `json:"doc"`
}

// getLegalPage serves a legal page that comes from files, in the language
// of the lang query parameter or its fallback. It needs no session.
func (a *api) getLegalPage(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("page")
	doc, lang, ok := a.legal.Lookup(name, r.URL.Query().Get("lang"))
	if !ok {
		return NewError(http.StatusNotFound, "not_found", "Legal page not found")
	}
	body := legalPage{Page: name, Doc: doc}
	if lang != "" {
		body.Language = &lang
	}
	WriteJSON(w, http.StatusOK, body)
	return nil
}
