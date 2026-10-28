package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"boardly/internal/notes"
)

// maxNoteBytes caps a note request: the largest document plus the other fields.
const maxNoteBytes = notes.MaxDocSize + 16<<10

// notesError maps store errors to API errors; invalid fields are reported as
// invalidCode (invalid_note or invalid_category). Other errors pass through.
func notesError(err error, invalidCode string) error {
	switch {
	case errors.Is(err, notes.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, notes.ErrInvalidDocument):
		return NewError(http.StatusBadRequest, "invalid_document", "The content is not a valid note document")
	case errors.Is(err, notes.ErrVersionConflict):
		return NewError(http.StatusConflict, "version_conflict", "The note was changed elsewhere")
	case errors.Is(err, notes.ErrInvalidParent):
		return NewError(http.StatusBadRequest, "invalid_parent", "The parent must be another category of yours outside this one")
	case errors.Is(err, notes.ErrInvalidTitle):
		return NewError(http.StatusBadRequest, invalidCode, "Titles can have at most 200 characters; category titles are required")
	case errors.Is(err, notes.ErrInvalidIcon):
		return NewError(http.StatusBadRequest, invalidCode, "The icon must be a known symbol or an image you uploaded")
	case errors.Is(err, notes.ErrInvalidColor):
		return NewError(http.StatusBadRequest, invalidCode, "Unknown colour")
	case errors.Is(err, notes.ErrInvalidCategory):
		return NewError(http.StatusBadRequest, invalidCode, "The category must be one of yours")
	}
	return err
}

// decodeFields decodes a JSON object into its raw fields. A body over max
// bytes is reported as tooLarge.
func decodeFields(r *http.Request, max int64, tooLarge error) (map[string]json.RawMessage, error) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, max)).Decode(&body); err != nil {
		if maxErr := new(http.MaxBytesError); errors.As(err, &maxErr) {
			return nil, tooLarge
		}
		return nil, NewError(http.StatusBadRequest, "invalid_request", "The request body is not a JSON object")
	}
	if body == nil {
		return nil, NewError(http.StatusBadRequest, "invalid_request", "The request body is not a JSON object")
	}
	return body, nil
}

// field decodes body[name] into dst, which must be a pointer to a pointer
// type so that null clears the field. It reports whether the field is
// present; a value of the wrong type is reported as invalid.
func field[T any](body map[string]json.RawMessage, name string, dst **T, invalid error) (bool, error) {
	raw, ok := body[name]
	if !ok {
		return false, nil
	}
	delete(body, name)
	if err := json.Unmarshal(raw, dst); err != nil {
		return true, invalid
	}
	return true, nil
}

func invalid(code, message string) error {
	return NewError(http.StatusBadRequest, code, message)
}

// unknownFields reports fields left in body after all known ones were taken.
func unknownFields(body map[string]json.RawMessage, code string) error {
	for name := range body {
		return invalid(code, "Unknown field "+name)
	}
	return nil
}

func (a *api) listNotes(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.notes.Notes(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) getNote(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	n, err := a.notes.Note(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		return notesError(err, "invalid_note")
	}
	WriteJSON(w, http.StatusOK, n)
	return nil
}

// noteFields takes the editable fields of a note from body into p.
func noteFields(body map[string]json.RawMessage, p *notes.NotePatch) error {
	bad := invalid("invalid_note", "A field has the wrong type")
	var title *string
	if ok, err := field(body, "title", &title, bad); err != nil {
		return err
	} else if ok {
		p.Title = &title
	}
	var icon *notes.Icon
	if ok, err := field(body, "icon", &icon, bad); err != nil {
		return err
	} else if ok {
		p.Icon = &icon
	}
	var color *string
	if ok, err := field(body, "color", &color, bad); err != nil {
		return err
	} else if ok {
		p.Color = &color
	}
	var category *string
	if ok, err := field(body, "categoryId", &category, bad); err != nil {
		return err
	} else if ok {
		p.CategoryID = &category
	}
	if raw, ok := body["content"]; ok {
		delete(body, "content")
		p.Content = raw
	}
	return unknownFields(body, "invalid_note")
}

func (a *api) createNote(w http.ResponseWriter, r *http.Request) error {
	body, err := decodeFields(r, maxNoteBytes, notesError(notes.ErrInvalidDocument, ""))
	if err != nil {
		return err
	}
	// text is plain text that becomes the content, for notes saved from the
	// share page.
	var text *string
	hasText, err := field(body, "text", &text, invalid("invalid_request", "text must be a string"))
	if err != nil {
		return err
	}
	if hasText && text == nil {
		return invalid("invalid_request", "text must be a string")
	}
	if _, ok := body["content"]; ok && hasText {
		return invalid("invalid_request", "Send either content or text, not both")
	}
	var p notes.NotePatch
	if err := noteFields(body, &p); err != nil {
		return err
	}
	if hasText {
		if p.Content, err = json.Marshal(notes.FromText(*text)); err != nil {
			return err
		}
	}
	n := notes.NewNote{Content: p.Content}
	if p.Title != nil {
		n.Title = *p.Title
	}
	if p.Icon != nil {
		n.Icon = *p.Icon
	}
	if p.Color != nil {
		n.Color = *p.Color
	}
	if p.CategoryID != nil {
		n.CategoryID = *p.CategoryID
	}
	user, _ := CurrentUser(r.Context())
	created, err := a.notes.CreateNote(r.Context(), user.ID, n)
	if err != nil {
		return notesError(err, "invalid_note")
	}
	WriteJSON(w, http.StatusCreated, created)
	return nil
}

func (a *api) updateNote(w http.ResponseWriter, r *http.Request) error {
	body, err := decodeFields(r, maxNoteBytes, notesError(notes.ErrInvalidDocument, ""))
	if err != nil {
		return err
	}
	var version *int
	if ok, err := field(body, "version", &version, nil); err != nil || !ok || version == nil {
		return invalid("invalid_request", "The version the change is based on is required")
	}
	var p notes.NotePatch
	if err := noteFields(body, &p); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	n, err := a.notes.UpdateNote(r.Context(), user.ID, r.PathValue("id"), *version, p)
	if err != nil {
		return notesError(err, "invalid_note")
	}
	WriteJSON(w, http.StatusOK, n)
	return nil
}

func (a *api) deleteNote(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.notes.DeleteNote(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return notesError(err, "invalid_note")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) listNoteCategories(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.notes.Categories(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

// categoryFields takes the editable fields of a category from body into p.
func categoryFields(body map[string]json.RawMessage, p *notes.CategoryPatch) error {
	bad := invalid("invalid_category", "A field has the wrong type")
	var title *string
	if ok, err := field(body, "title", &title, bad); err != nil {
		return err
	} else if ok {
		if title == nil {
			return invalid("invalid_category", "The title is required")
		}
		p.Title = title
	}
	var icon *notes.Icon
	if ok, err := field(body, "icon", &icon, bad); err != nil {
		return err
	} else if ok {
		p.Icon = &icon
	}
	var color *string
	if ok, err := field(body, "color", &color, bad); err != nil {
		return err
	} else if ok {
		p.Color = &color
	}
	var parent *string
	if ok, err := field(body, "parentId", &parent, invalid("invalid_parent", "The parent must be a category ID or null")); err != nil {
		return err
	} else if ok {
		p.ParentID = &parent
	}
	return unknownFields(body, "invalid_category")
}

func (a *api) createNoteCategory(w http.ResponseWriter, r *http.Request) error {
	body, err := decodeFields(r, maxBodyBytes, invalid("invalid_request", "The request body is too large"))
	if err != nil {
		return err
	}
	var p notes.CategoryPatch
	if err := categoryFields(body, &p); err != nil {
		return err
	}
	if p.Title == nil {
		return invalid("invalid_category", "The title is required")
	}
	n := notes.NewCategory{Title: *p.Title}
	if p.Icon != nil {
		n.Icon = *p.Icon
	}
	if p.Color != nil {
		n.Color = *p.Color
	}
	if p.ParentID != nil {
		n.ParentID = *p.ParentID
	}
	user, _ := CurrentUser(r.Context())
	c, err := a.notes.CreateCategory(r.Context(), user.ID, n)
	if err != nil {
		return notesError(err, "invalid_category")
	}
	WriteJSON(w, http.StatusCreated, c)
	return nil
}

func (a *api) updateNoteCategory(w http.ResponseWriter, r *http.Request) error {
	body, err := decodeFields(r, maxBodyBytes, invalid("invalid_request", "The request body is too large"))
	if err != nil {
		return err
	}
	var p notes.CategoryPatch
	if err := categoryFields(body, &p); err != nil {
		return err
	}
	user, _ := CurrentUser(r.Context())
	c, err := a.notes.UpdateCategory(r.Context(), user.ID, r.PathValue("id"), p)
	if err != nil {
		return notesError(err, "invalid_category")
	}
	WriteJSON(w, http.StatusOK, c)
	return nil
}

func (a *api) deleteNoteCategory(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.notes.DeleteCategory(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return notesError(err, "invalid_category")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
