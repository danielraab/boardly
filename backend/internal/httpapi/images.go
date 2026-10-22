package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"famodash/internal/images"
)

// Cache lifetimes of served images. An upload's bytes never change; a
// favicon's are replaced under the same ID when it is refreshed.
const (
	uploadCacheControl  = "public, max-age=31536000, immutable"
	faviconCacheControl = "public, max-age=86400"
)

// imagesError maps image errors to API errors; other errors pass through.
func imagesError(err error) error {
	switch {
	case errors.Is(err, images.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Image not found")
	case errors.Is(err, images.ErrUnsupported):
		return NewError(http.StatusBadRequest, "unsupported_image", "Images must be PNG, JPEG, WebP, GIF or ICO files")
	case errors.Is(err, images.ErrTooLarge):
		return NewError(http.StatusRequestEntityTooLarge, "image_too_large", "Images can be at most 1 MB")
	case errors.Is(err, images.ErrInvalidURL):
		return NewError(http.StatusBadRequest, "invalid_url", "The URL must be an http or https address")
	case errors.Is(err, images.ErrBlocked):
		return NewError(http.StatusBadRequest, "blocked_address", "Favicons aren't fetched from private addresses")
	case errors.Is(err, images.ErrNoFavicon):
		return NewError(http.StatusNotFound, "favicon_not_found", "No favicon was found for this address")
	}
	return err
}

// getImage serves an image to anyone. Its ID is the only access check.
func (a *api) getImage(w http.ResponseWriter, r *http.Request) error {
	img, f, err := a.images.Open(r.Context(), r.PathValue("id"))
	if err != nil {
		return imagesError(err)
	}
	defer f.Close()
	h := w.Header()
	h.Set("Content-Type", img.ContentType)
	h.Set("Content-Length", strconv.FormatInt(img.Size, 10))
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
	if img.Source == images.SourceUpload {
		h.Set("Cache-Control", uploadCacheControl)
	} else {
		h.Set("Cache-Control", faviconCacheControl)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
	return nil
}

// uploadImage stores the request body, the raw file, as the user's image.
func (a *api) uploadImage(w http.ResponseWriter, r *http.Request) error {
	data, err := images.ReadLimited(r.Body)
	if err != nil {
		return imagesError(err)
	}
	user, _ := CurrentUser(r.Context())
	img, err := a.images.Save(r.Context(), images.SourceUpload, user.ID, data)
	if err != nil {
		return imagesError(err)
	}
	WriteJSON(w, http.StatusCreated, map[string]string{"id": img.ID})
	return nil
}

// getFavicon returns the favicon of a URL's origin, fetching it if needed.
func (a *api) getFavicon(w http.ResponseWriter, r *http.Request) error {
	var body struct {
		URL string `json:"url"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return err
	}
	id, err := a.favicons.Get(r.Context(), body.URL)
	if err != nil {
		return imagesError(err)
	}
	WriteJSON(w, http.StatusOK, map[string]string{"id": id})
	return nil
}
