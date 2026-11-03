package httpapi

import (
	"bytes"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"boardly/internal/export"
)

// exportData downloads all data of the signed-in user (openspec change
// export-all-data): the export document as JSON, or with files=true a ZIP
// archive that also holds the image and attachment files.
func (a *api) exportData(w http.ResponseWriter, r *http.Request) error {
	withFiles := false
	switch r.URL.Query().Get("files") {
	case "", "false":
	case "true":
		withFiles = true
	default:
		return NewError(http.StatusBadRequest, "invalid_request", `files must be "true" or "false"`)
	}

	user, _ := CurrentUser(r.Context())
	now := a.export.Now()
	if ok, retry := a.exportLimiter.Allow(user.ID, now); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many exports, try again later")
	}

	doc, err := a.export.Document(r.Context(), user.ID)
	if err != nil {
		return err
	}
	name := export.FileName(doc.Profile.Slug, now, withFiles)
	header := w.Header()
	header.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	header.Set("Cache-Control", "no-store")

	if !withFiles {
		// Encoded in full first, so an error still gets an error response.
		var buf bytes.Buffer
		if err := export.WriteJSON(&buf, doc); err != nil {
			return err
		}
		header.Set("Content-Type", "application/json")
		header.Set("Content-Length", strconv.Itoa(buf.Len()))
		_, _ = buf.WriteTo(w)
		return nil
	}

	header.Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
	if err := a.export.WriteArchive(w, doc); err != nil {
		// The status is already sent. Abort the connection, so the client
		// sees a failed download rather than an archive that looks complete.
		slog.ErrorContext(r.Context(), "writing export archive", "user", user.ID, "error", err)
		panic(http.ErrAbortHandler)
	}
	return nil
}
