package httpapi

import (
	"errors"
	"io"
	"net/http"

	"boardly/internal/dataimport"
)

// importData imports a Boardly export (openspec change import-boardly-export):
// the JSON document or the ZIP archive in the body, sent as
// application/octet-stream like the bookmark import. ?parts= lists the parts
// to import; ?folderId=, ?categoryId= and ?skipDuplicates= work as for the
// bookmark import.
func (a *api) importData(w http.ResponseWriter, r *http.Request) error {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxImportBytes))
	if tooLarge := new(http.MaxBytesError); errors.As(err, &tooLarge) {
		return NewError(http.StatusRequestEntityTooLarge, "file_too_large", "The import file can have at most 10 MB")
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return NewError(http.StatusBadRequest, "invalid_request", "The import file is missing")
	}
	query := r.URL.Query()
	parts, ok := dataimport.ParseParts(query.Get("parts"))
	if !ok {
		return NewError(http.StatusBadRequest, "invalid_request", "parts must list favourites, dashboards, notes, media or feeds")
	}
	opts := dataimport.Options{Parts: parts, FolderID: query.Get("folderId"), SkipDuplicates: true}
	if parts.Favourites && opts.FolderID == "" {
		return errInvalidImportFolder()
	}
	if id := query.Get("categoryId"); id != "" {
		opts.CategoryID = &id
	}
	switch query.Get("skipDuplicates") {
	case "", "true":
	case "false":
		opts.SkipDuplicates = false
	default:
		return NewError(http.StatusBadRequest, "invalid_request", "skipDuplicates must be true or false")
	}

	user, _ := CurrentUser(r.Context())
	report, err := a.dataImport.Import(r.Context(), user.ID, data, opts)
	switch {
	case errors.Is(err, dataimport.ErrInvalidFile):
		return NewError(http.StatusBadRequest, "invalid_import_file", "The file is not a valid Boardly export")
	case errors.Is(err, dataimport.ErrInvalidFolder):
		return errInvalidImportFolder()
	case errors.Is(err, dataimport.ErrInvalidCategory):
		return errInvalidImportCategory()
	case err != nil:
		return err
	}
	WriteJSON(w, http.StatusOK, report)
	return nil
}
