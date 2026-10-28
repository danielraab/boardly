package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"boardly/internal/favourites"
	"boardly/internal/importer"
	"boardly/internal/notes"
)

// maxImportBytes is the largest import file. A Karakeep export of 168
// bookmarks has about 50 KB, so this is enough for tens of thousands.
const maxImportBytes = 10 << 20

// importReport is the response of an import: the links and folders, and the
// notes and categories, that it added and left out.
type importReport struct {
	Links                 int                 `json:"links"`
	Folders               int                 `json:"folders"`
	Notes                 int                 `json:"notes"`
	Categories            int                 `json:"categories"`
	Skipped               importReportSkipped `json:"skipped"`
	ShortenedDescriptions int                 `json:"shortenedDescriptions"`
}

type importReportSkipped struct {
	Duplicates     int `json:"duplicates"`
	DuplicateNotes int `json:"duplicateNotes"`
	InvalidURLs    int `json:"invalidUrls"`
	NotLinks       int `json:"notLinks"`
	SmartLists     int `json:"smartLists"`
	TooLargeNotes  int `json:"tooLargeNotes"`
}

// importFavourites imports the bookmark file in the body, sent as
// application/octet-stream like image uploads, into the folder ?folderId= of
// the user's tree, and its notes into the category ?categoryId= (missing or
// empty for the top level), and returns the report. ?format= is the file's
// format, and ?skipDuplicates=false imports links and notes that are already
// saved. Links and notes are written in one transaction.
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
	var categoryID *string
	if id := r.URL.Query().Get("categoryId"); id != "" {
		categoryID = &id
	}
	skipDuplicates := true
	switch r.URL.Query().Get("skipDuplicates") {
	case "", "true":
	case "false":
		skipDuplicates = false
	default:
		return NewError(http.StatusBadRequest, "invalid_request", "skipDuplicates must be true or false")
	}

	res, ok, err := importer.Parse(importer.Format(r.URL.Query().Get("format")), bytes.NewReader(data))
	switch {
	case !ok:
		return NewError(http.StatusBadRequest, "invalid_request", "The format must be karakeep, linkwarden or netscape")
	case errors.Is(err, importer.ErrInvalidFile):
		return NewError(http.StatusBadRequest, "invalid_import_file", "The file is not a valid export of the chosen format")
	case err != nil:
		return err
	}

	user, _ := CurrentUser(r.Context())
	tx, err := a.favourites.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	links, err := a.favourites.ImportTx(r.Context(), tx, user.ID, folderID, res.Links, skipDuplicates)
	if errors.Is(err, favourites.ErrNotFound) || errors.Is(err, favourites.ErrInvalidParent) {
		return errInvalidImportFolder()
	}
	if err != nil {
		return err
	}
	ns, err := a.notes.ImportTx(r.Context(), tx, user.ID, categoryID, res.Notes, skipDuplicates)
	if errors.Is(err, notes.ErrInvalidCategory) {
		return errInvalidImportCategory()
	}
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.favourites.AfterImport(links)

	WriteJSON(w, http.StatusOK, importReport{
		Links: links.Links, Folders: links.Folders, Notes: ns.Notes, Categories: ns.Categories,
		Skipped: importReportSkipped{
			Duplicates: links.Skipped.Duplicates, DuplicateNotes: ns.DuplicateNotes,
			InvalidURLs: links.Skipped.InvalidURLs, NotLinks: links.Skipped.NotLinks,
			SmartLists: links.Skipped.SmartLists, TooLargeNotes: ns.TooLargeNotes,
		},
		ShortenedDescriptions: links.ShortenedDescriptions,
	})
	return nil
}

func errInvalidImportFolder() error {
	return NewError(http.StatusBadRequest, "invalid_folder", "The import folder must be one of your folders")
}

func errInvalidImportCategory() error {
	return NewError(http.StatusBadRequest, "invalid_category", "The import category must be one of your categories")
}
