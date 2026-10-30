package httpapi

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"boardly/internal/apptokens"
	"boardly/internal/attachments"
	"boardly/internal/notes"
)

// karakeepRESTError is an error of Karakeep's REST endpoints, which the apps
// read as {"error": message}.
type karakeepRESTError struct {
	status  int
	message string
}

func (e *karakeepRESTError) Error() string { return e.message }

// karakeepREST serves h to requests with a Karakeep token and writes its
// errors in Karakeep's REST format.
func karakeepREST(h func(w http.ResponseWriter, r *http.Request, token apptokens.Token) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		if token, ok := appToken(r.Context()); ok {
			err = h(w, r, token)
		} else {
			err = &karakeepRESTError{http.StatusUnauthorized, "Unauthorized"}
		}
		if err == nil {
			return
		}
		var e *karakeepRESTError
		if !errors.As(err, &e) {
			slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
			e = &karakeepRESTError{http.StatusInternalServerError, "Internal server error"}
		}
		WriteJSON(w, e.status, map[string]string{"error": e.message})
	})
}

// karakeepAttachmentError maps attachment errors to Karakeep REST errors;
// other errors pass through.
func (a *api) karakeepAttachmentError(err error) error {
	switch {
	case errors.Is(err, attachments.ErrNotFound):
		return &karakeepRESTError{http.StatusNotFound, "Not found"}
	case errors.Is(err, attachments.ErrInvalidName):
		return &karakeepRESTError{http.StatusBadRequest, "The file name must have 1 to 255 characters without slashes or control characters"}
	case errors.Is(err, attachments.ErrUnsupported):
		return &karakeepRESTError{http.StatusBadRequest, "This type of file can't be attached"}
	case errors.Is(err, attachments.ErrTooLarge), errors.As(err, new(*http.MaxBytesError)):
		return &karakeepRESTError{http.StatusRequestEntityTooLarge, fmt.Sprintf("Attachments can be at most %d MB", a.attachments.MaxBytes>>20)}
	}
	return err
}

// karakeepUploadAsset stores the "file" part of a multipart upload as an
// attachment, for an asset bookmark that the app creates next.
func (a *api) karakeepUploadAsset(w http.ResponseWriter, r *http.Request, token apptokens.Token) error {
	if karakeepScope(token).Notes.Mode == notes.ScopeNone {
		return &karakeepRESTError{http.StatusBadRequest, "This connected app can't save notes"}
	}
	noFile := &karakeepRESTError{http.StatusBadRequest, "The upload needs a file"}
	r.Body = http.MaxBytesReader(nil, r.Body, a.attachments.MaxBytes+64<<10)
	mr, err := r.MultipartReader()
	if err != nil {
		return noFile
	}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return noFile
		}
		if err != nil {
			if mapped := a.karakeepAttachmentError(err); mapped != err {
				return mapped
			}
			return noFile
		}
		if part.FormName() != "file" {
			continue
		}
		data, err := a.attachments.ReadLimited(part)
		if err != nil {
			return a.karakeepAttachmentError(err)
		}
		att, err := a.attachments.Create(r.Context(), token.UserID, part.FileName(), data)
		if err != nil {
			return a.karakeepAttachmentError(err)
		}
		WriteJSON(w, http.StatusOK, map[string]any{
			"assetId": att.ID, "contentType": att.ContentType, "size": att.Size, "fileName": att.Name,
		})
		return nil
	}
}

// karakeepAsset serves one of the owner's attachments that a note in the
// token's note scope refers to, or that no note refers to, such as a file
// just uploaded.
func (a *api) karakeepAsset(w http.ResponseWriter, r *http.Request, token apptokens.Token) error {
	id := r.PathValue("id")
	att, err := a.attachments.Get(r.Context(), token.UserID, id)
	if err != nil {
		return a.karakeepAttachmentError(err)
	}
	if len(att.NoteIDs) > 0 {
		ns, err := a.notes.ScopedNotes(r.Context(), karakeepScope(token).Notes)
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(ns, func(n notes.Note) bool { return slices.Contains(att.NoteIDs, n.ID) }) {
			return a.karakeepAttachmentError(attachments.ErrNotFound)
		}
	}
	return a.karakeepAttachmentError(a.serveAttachment(w, r, id))
}
