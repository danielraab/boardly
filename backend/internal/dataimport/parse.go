// Package dataimport imports a Boardly export (openspec change
// import-boardly-export): the JSON document of the export package, or its
// ZIP archive with image and attachment files, added to a user's account
// with new IDs.
package dataimport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"boardly/internal/export"
)

// ErrInvalidFile is returned for a body that isn't a Boardly export of a
// supported version.
var ErrInvalidFile = errors.New("invalid import file")

// maxDocumentBytes is the largest boardly.json read from an archive.
const maxDocumentBytes = 50 << 20

// File is a parsed export: the document and, for an archive, its files.
type File struct {
	Doc     *export.Document
	archive *zip.Reader
}

// Parse reads a Boardly export: a ZIP archive with boardly.json, or the
// JSON document itself.
func Parse(data []byte) (*File, error) {
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
		}
		f, err := zr.Open(export.DocumentName)
		if err != nil {
			return nil, fmt.Errorf("%w: no %s in the archive", ErrInvalidFile, export.DocumentName)
		}
		defer func() { _ = f.Close() }()
		doc, err := readLimited(f, maxDocumentBytes)
		if err != nil {
			return nil, err
		}
		parsed, err := parseDocument(doc)
		if err != nil {
			return nil, err
		}
		return &File{Doc: parsed, archive: zr}, nil
	}
	parsed, err := parseDocument(data)
	if err != nil {
		return nil, err
	}
	return &File{Doc: parsed}, nil
}

func parseDocument(data []byte) (*export.Document, error) {
	var head struct {
		Format  string `json:"format"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	// Version 1 is the same document without feeds, version 2 without
	// listen records.
	if head.Format != export.Format || head.Version < 1 || head.Version > export.Version {
		return nil, fmt.Errorf("%w: format %q version %d", ErrInvalidFile, head.Format, head.Version)
	}
	var doc export.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	return &doc, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrInvalidFile, export.DocumentName, limit)
	}
	return data, nil
}

var (
	// errFileTooLarge marks an archive file over its limit.
	errFileTooLarge = errors.New("file too large")
	// errNoFile is returned for a file of a JSON export or a null path.
	errNoFile = errors.New("no such file")
)

// ReadFile returns the archive file at path, read up to limit bytes. It
// returns an error for a file that is missing, unreadable or larger than
// limit, and for any path of a JSON export, which has no files. Paths are
// looked up as archive entries (zip.Reader.Open rejects ".." and absolute
// paths), never on disk.
func (f *File) ReadFile(path *string, limit int64) ([]byte, error) {
	if f.archive == nil || path == nil {
		return nil, errNoFile
	}
	r, err := f.archive.Open(*path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errFileTooLarge
	}
	return data, nil
}
