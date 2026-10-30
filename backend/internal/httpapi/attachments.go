package httpapi

import (
	"errors"
	"fmt"
	"mime"
	"net/http"

	"boardly/internal/attachments"
	"boardly/internal/blobs"
)

// attachmentsError maps attachment errors to API errors; other errors pass
// through.
func (a *api) attachmentsError(err error) error {
	switch {
	case errors.Is(err, attachments.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, attachments.ErrInvalidName):
		return NewError(http.StatusBadRequest, "invalid_attachment_name",
			"The file name must have 1 to 255 characters without slashes or control characters")
	case errors.Is(err, attachments.ErrUnsupported):
		return NewError(http.StatusBadRequest, "unsupported_attachment_type", "This type of file can't be attached")
	case errors.Is(err, attachments.ErrTooLarge):
		return NewError(http.StatusRequestEntityTooLarge, "attachment_too_large",
			fmt.Sprintf("Attachments can be at most %d MB", a.attachments.MaxBytes>>20))
	case errors.Is(err, attachments.ErrInUse):
		return NewError(http.StatusConflict, "attachment_in_use", "The attachment is still used in notes")
	}
	return err
}

// listAttachments returns the user's attachments and the upload limits.
func (a *api) listAttachments(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.attachments.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"attachments":  list,
		"maxSize":      a.attachments.MaxBytes,
		"allowedTypes": a.attachments.Allowed,
	})
	return nil
}

// uploadAttachment stores the request body, the raw file, as an attachment
// named by the "name" query parameter.
func (a *api) uploadAttachment(w http.ResponseWriter, r *http.Request) error {
	name, err := attachments.CleanName(r.URL.Query().Get("name"))
	if err != nil {
		return a.attachmentsError(err)
	}
	data, err := a.attachments.ReadLimited(r.Body)
	if err != nil {
		return a.attachmentsError(err)
	}
	user, _ := CurrentUser(r.Context())
	att, err := a.attachments.Create(r.Context(), user.ID, name, data)
	if err != nil {
		return a.attachmentsError(err)
	}
	WriteJSON(w, http.StatusCreated, att)
	return nil
}

// patchAttachment sets or removes the keep flag.
func (a *api) patchAttachment(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		Keep *bool `json:"keep"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	if body.Keep == nil {
		return NewError(http.StatusBadRequest, "invalid_request", "keep must be true or false")
	}
	user, _ := CurrentUser(r.Context())
	att, err := a.attachments.SetKeep(r.Context(), user.ID, r.PathValue("id"), *body.Keep)
	if err != nil {
		return a.attachmentsError(err)
	}
	WriteJSON(w, http.StatusOK, att)
	return nil
}

// deleteAttachment deletes an attachment that no note refers to.
func (a *api) deleteAttachment(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.attachments.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return a.attachmentsError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// getAttachment serves an attachment to anyone. The name in the path is only
// for the browser; the ID alone identifies the file.
func (a *api) getAttachment(w http.ResponseWriter, r *http.Request) error {
	att, f, err := a.attachments.Open(r.Context(), r.PathValue("id"))
	if err != nil {
		return a.attachmentsError(err)
	}
	defer f.Close()
	h := w.Header()
	blobs.SetHeaders(h, att.ContentType)
	h.Set("Cache-Control", uploadCacheControl)
	disposition := "attachment"
	if att.Inline {
		disposition = "inline"
	}
	h.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": att.Name}))
	http.ServeContent(w, r, "", att.CreatedAt, f)
	return nil
}
