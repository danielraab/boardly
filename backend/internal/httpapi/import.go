package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"boardly/internal/favourites"
	"boardly/internal/importer"
)

// maxImportBytes is the largest import file. A Karakeep export of 168
// bookmarks has about 50 KB, so this is enough for tens of thousands.
const maxImportBytes = 10 << 20

// importFavourites imports the bookmark file in the body, sent as
// application/octet-stream like image uploads, into the folder ?folderId= of
// the user's tree and returns the report. ?format= is the file's format.
func (a *api) importFavourites(w http.ResponseWriter, r *http.Request) error {
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
	folderID := r.URL.Query().Get("folderId")
	if folderID == "" {
		return errInvalidImportFolder()
	}

	tree, ok, err := importer.Parse(importer.Format(r.URL.Query().Get("format")), bytes.NewReader(data))
	switch {
	case !ok:
		return NewError(http.StatusBadRequest, "invalid_request", "The format must be karakeep, linkwarden or netscape")
	case errors.Is(err, importer.ErrInvalidFile):
		return NewError(http.StatusBadRequest, "invalid_import_file", "The file is not a valid export of the chosen format")
	case err != nil:
		return err
	}

	user, _ := CurrentUser(r.Context())
	report, err := a.favourites.Import(r.Context(), user.ID, folderID, tree)
	if errors.Is(err, favourites.ErrNotFound) || errors.Is(err, favourites.ErrInvalidParent) {
		return errInvalidImportFolder()
	}
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, report)
	return nil
}

func errInvalidImportFolder() error {
	return NewError(http.StatusBadRequest, "invalid_folder", "The import folder must be one of your folders")
}
