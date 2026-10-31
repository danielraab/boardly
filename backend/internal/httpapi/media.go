package httpapi

import (
	"errors"
	"net/http"

	"boardly/internal/media"
	"boardly/internal/notes"
)

// maxMediaBytes caps a media request.
const maxMediaBytes = 16 << 10

// mediaError maps store and URL check errors to API errors. Other errors
// pass through.
func mediaError(err error) error {
	switch {
	case errors.Is(err, media.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, media.ErrInvalidKind):
		return invalid("invalid_kind", "Unknown media source kind")
	case errors.Is(err, media.ErrInvalidTitle):
		return invalid("invalid_title", "The title must have 1 to 200 characters")
	case errors.Is(err, media.ErrInvalidIcon):
		return invalid("invalid_icon", "The icon must be a known symbol or an image you uploaded")
	case errors.Is(err, media.ErrInvalidOrder):
		return invalid("invalid_order", "The order must list each of your media sources once")
	case errors.Is(err, media.ErrInvalidURL):
		return invalid("invalid_url", "The URL must be an absolute http or https URL")
	case errors.Is(err, media.ErrNotHTTPS):
		return invalid("stream_not_https", "The stream isn't available over https")
	case errors.Is(err, media.ErrUnreachable):
		return invalid("stream_unreachable", "The URL doesn't answer with an audio stream")
	}
	return err
}

func (a *api) listMedia(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.media.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) createMedia(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	body, err := decodeFields(r, maxMediaBytes, invalid("invalid_request", "The request is too large"))
	if err != nil {
		return err
	}
	bad := invalid("invalid_request", "A field has the wrong type")
	var n media.NewSource
	for name, dst := range map[string]*string{"kind": &n.Kind, "title": &n.Title, "url": &n.URL, "logoUrl": &n.LogoURL} {
		var v *string
		if _, err := field(body, name, &v, bad); err != nil {
			return err
		} else if v != nil {
			*dst = *v
		}
	}
	if _, err := field(body, "icon", &n.Icon, bad); err != nil {
		return err
	}
	if err := unknownFields(body, "invalid_request"); err != nil {
		return err
	}
	src, err := a.media.Create(r.Context(), user.ID, n)
	if err != nil {
		return mediaError(err)
	}
	WriteJSON(w, http.StatusCreated, src)
	return nil
}

func (a *api) updateMedia(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	body, err := decodeFields(r, maxMediaBytes, invalid("invalid_request", "The request is too large"))
	if err != nil {
		return err
	}
	bad := invalid("invalid_request", "A field has the wrong type")
	var p media.Patch
	for name, dst := range map[string]**string{"title": &p.Title, "url": &p.URL} {
		var v *string
		if ok, err := field(body, name, &v, bad); err != nil {
			return err
		} else if ok && v == nil {
			return bad
		} else if ok {
			*dst = v
		}
	}
	var icon *notes.Icon
	if ok, err := field(body, "icon", &icon, bad); err != nil {
		return err
	} else if ok {
		p.Icon = &icon
	}
	if err := unknownFields(body, "invalid_request"); err != nil {
		return err
	}
	src, err := a.media.Update(r.Context(), user.ID, r.PathValue("id"), p)
	if err != nil {
		return mediaError(err)
	}
	WriteJSON(w, http.StatusOK, src)
	return nil
}

func (a *api) reorderMedia(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	var ids []string
	if err := decodeJSON(r, &ids); err != nil {
		return err
	}
	if err := a.media.Reorder(r.Context(), user.ID, ids); err != nil {
		return mediaError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) deleteMedia(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.media.Delete(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return mediaError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
