package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"boardly/internal/jobs"
)

// Previews fetches the description and preview image of links' pages and
// converts preview images, as jobs of the persistent queue. A link's preview
// belongs to that link only.
type Previews struct {
	Images *Store
	Jobs   *jobs.Queue
	// Client is the guarded client, shared with the favicon fetcher.
	Client    *http.Client
	Converter Converter
	// MaxBytes is the largest image accepted before conversion.
	MaxBytes int64

	rawDir string
}

// rawDirName is the directory below the images directory that holds raw
// images waiting for their conversion.
const rawDirName = "raw"

// NewPreviews returns a preview fetcher that keeps raw images waiting for
// their conversion in the store's raw directory, creating it.
func NewPreviews(store *Store, queue *jobs.Queue, client *http.Client, maxBytes int64, conv Converter) (*Previews, error) {
	dir := filepath.Join(store.Dir, rawDirName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("creating raw image directory: %w", err)
	}
	return &Previews{Images: store, Jobs: queue, Client: client, Converter: conv, MaxBytes: maxBytes, rawDir: dir}, nil
}

// Run starts the workers of the preview jobs until ctx is done. One worker
// converts, because decoding a large image takes much memory.
func (p *Previews) Run(ctx context.Context) {
	p.Jobs.Run(ctx, jobs.KindPreviewFetch, 2, p.fetch)
	p.Jobs.Run(ctx, jobs.KindPreviewConvert, 1, p.convert)
}

// Replace queues data, an image a connected app sent for a link, to become
// that link's preview, replacing the current one. It checks the limits
// first: ErrTooLarge, ErrUnsupported or ErrTooManyPixels.
func (p *Previews) Replace(ctx context.Context, userID, nodeID, rawURL string, data []byte) error {
	if int64(len(data)) > p.MaxBytes {
		return ErrTooLarge
	}
	if _, err := PreviewType(data); err != nil {
		return err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return ErrUnsupported
	}
	if int64(cfg.Width)*int64(cfg.Height) > p.Converter.MaxPixels {
		return ErrTooManyPixels
	}
	return p.queueConvert(ctx, jobs.PreviewConvertPayload{UserID: userID, NodeID: nodeID, URL: rawURL, Replace: true}, data)
}

// queueConvert writes data as a raw file and queues its conversion.
func (p *Previews) queueConvert(ctx context.Context, job jobs.PreviewConvertPayload, data []byte) error {
	name := uuid.NewString()
	if err := writeAtomic(p.rawDir, name, data); err != nil {
		return err
	}
	job.File = name
	if err := p.Jobs.Enqueue(ctx, p.Images.DB, jobs.KindPreviewConvert, "", job); err != nil {
		_ = os.Remove(filepath.Join(p.rawDir, name))
		return err
	}
	return nil
}

// rawPath is the file of a raw image; names are UUIDs, anything else maps to
// a name that can't exist.
func (p *Previews) rawPath(name string) string {
	if u, err := uuid.Parse(name); err != nil || u.String() != name {
		return filepath.Join(p.rawDir, "invalid")
	}
	return filepath.Join(p.rawDir, name)
}

// fetch is the preview_fetch job: it loads the link's page, stores its
// description when the link has none, and downloads its preview image for
// conversion when the link has no preview. Failures that are the site's, not
// the server's, are not errors.
func (p *Previews) fetch(ctx context.Context, job jobs.Job) error {
	var pl jobs.PreviewFetchPayload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return err
	}
	meta, err := p.loadMeta(ctx, pl.URL)
	if err != nil || (meta.Description == "" && meta.ImageURL == "") {
		return siteError(err)
	}
	if meta.Description != "" {
		if _, err := p.Images.DB.ExecContext(ctx,
			`UPDATE favourite_nodes SET description = ?
			 WHERE id = ? AND user_id = ? AND kind = 'link' AND url = ? AND description IS NULL`,
			meta.Description, pl.NodeID, pl.UserID, pl.URL,
		); err != nil {
			return err
		}
	}
	if meta.ImageURL == "" {
		return nil
	}
	// Skip the download when the link already has a preview or changed.
	var open bool
	if err := p.Images.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM favourite_nodes
		 WHERE id = ? AND user_id = ? AND url = ? AND preview_image_id IS NULL)`,
		pl.NodeID, pl.UserID, pl.URL,
	).Scan(&open); err != nil || !open {
		return err
	}
	data, err := p.download(ctx, meta.ImageURL)
	if err != nil {
		return siteError(err)
	}
	return p.queueConvert(ctx, jobs.PreviewConvertPayload{UserID: pl.UserID, NodeID: pl.NodeID, URL: pl.URL}, data)
}

// siteError drops errors that only say something about the linked site.
func siteError(err error) error {
	if err == nil || errors.Is(err, ErrBlocked) || errors.Is(err, ErrTooLarge) || errors.Is(err, ErrUnsupported) ||
		errors.Is(err, errNotFetched) {
		return nil
	}
	return err
}

// errNotFetched is a page or image that the site didn't deliver.
var errNotFetched = errors.New("not fetched")

func (p *Previews) loadMeta(ctx context.Context, pageURL string) (PageMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	resp, err := p.get(ctx, pageURL, "text/html")
	if err != nil {
		return PageMeta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return PageMeta{}, errNotFetched
	}
	return ParseMeta(io.LimitReader(resp.Body, MaxPageSize), resp.Request.URL), nil
}

func (p *Previews) download(ctx context.Context, imageURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	resp, err := p.get(ctx, imageURL, "image/*")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errNotFetched
	}
	data, err := ReadLimit(resp.Body, p.MaxBytes)
	if err != nil {
		return nil, err
	}
	if _, err := PreviewType(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (p *Previews) get(ctx context.Context, rawURL, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "Boardly preview fetcher")
	return p.Client.Do(req)
}

// convert is the preview_convert job: it converts the raw image and stores it
// as the link's preview when the link still has the job's URL, and, unless
// the job replaces, still has no preview. The raw file is removed when the
// job is done; when the server stops during the job, it stays for the rerun.
func (p *Previews) convert(ctx context.Context, job jobs.Job) error {
	var pl jobs.PreviewConvertPayload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return err
	}
	path := p.rawPath(pl.File)
	defer func() {
		if ctx.Err() == nil {
			_ = os.Remove(path)
		}
	}()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	out, err := p.Converter.Convert(data)
	if errors.Is(err, ErrUnsupported) || errors.Is(err, ErrTooManyPixels) {
		return nil
	}
	if err != nil {
		return err
	}
	img, err := p.Images.Save(ctx, SourcePreview, pl.UserID, out)
	if err != nil {
		return err
	}
	query := `UPDATE favourite_nodes SET preview_image_id = ?
		WHERE id = ? AND user_id = ? AND kind = 'link' AND url = ? AND preview_image_id IS NULL`
	if pl.Replace {
		query = `UPDATE favourite_nodes SET preview_image_id = ?
		WHERE id = ? AND user_id = ? AND kind = 'link' AND url = ?`
	}
	res, err := p.Images.DB.ExecContext(ctx, query, img.ID, pl.NodeID, pl.UserID, pl.URL)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	if delErr := p.Images.Delete(context.WithoutCancel(ctx), img.ID); delErr != nil && err == nil {
		err = delErr
	}
	return err
}

// writeAtomic writes data to dir/name through a temporary file, so readers
// never see half a file.
func writeAtomic(dir, name string, data []byte) error {
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}
