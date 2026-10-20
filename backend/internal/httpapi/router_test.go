package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) PingContext(context.Context) error { return f.err }

var fakeWeb = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte("<html>app</html>"))
})

func do(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type = %q, want application/json", ct)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return body.Error.Code
}

func TestHealthOK(t *testing.T) {
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb})
	rec := do(t, h, http.MethodGet, "/api/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestHealthDatabaseDown(t *testing.T) {
	h := New(Deps{DB: fakeDB{err: errors.New("closed")}, Web: fakeWeb})
	rec := do(t, h, http.MethodGet, "/api/health")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if code := errorCode(t, rec); code != "unavailable" {
		t.Fatalf("code = %q", code)
	}
}

func TestUnknownAPIPath(t *testing.T) {
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb})
	rec := do(t, h, http.MethodGet, "/api/does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if code := errorCode(t, rec); code != "not_found" {
		t.Fatalf("code = %q", code)
	}
}

func TestWrongMethod(t *testing.T) {
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb})
	rec := do(t, h, http.MethodDelete, "/api/health")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if code := errorCode(t, rec); code != "method_not_allowed" {
		t.Fatalf("code = %q", code)
	}
	if allow := rec.Header().Get("Allow"); allow == "" {
		t.Fatal("missing Allow header")
	}
}

func TestNonAPIPathGoesToWeb(t *testing.T) {
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb})
	rec := do(t, h, http.MethodGet, "/profile")
	if rec.Code != http.StatusOK || rec.Body.String() != "<html>app</html>" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestHandlerFuncInternalError(t *testing.T) {
	h := HandlerFunc(func(http.ResponseWriter, *http.Request) error { return errors.New("boom") })
	rec := do(t, h, http.MethodGet, "/")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if code := errorCode(t, rec); code != "internal" {
		t.Fatalf("code = %q", code)
	}
}
