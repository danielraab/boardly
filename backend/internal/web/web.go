// Package web serves the built frontend that Vite writes into dist/.
package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// The markers in index.html that carry the app title. Keep them in sync with
// frontend/index.html.
const (
	titleMarker = "<title>Boardly</title>"
	metaMarker  = `<meta name="application-name" content="Boardly" />`
)

// mapStyleMarker carries the map style; keep it in sync with
// frontend/index.html.
const mapStyleMarker = `<meta name="boardly-map-style" content="https://tiles.openfreemap.org/styles/liberty" />`

// headEnd closes the head of index.html; the analytics snippet goes before it.
const headEnd = "</head>"

// Options are the instance settings the server writes into the web app.
type Options struct {
	// AppTitle is the product name in index.html and the manifests.
	AppTitle string
	// AnalyticsSnippet goes unescaped right before </head>; empty means none.
	AnalyticsSnippet string
	// MapStyleURL goes into the map style meta tag; empty keeps the built-in default.
	MapStyleURL string
	// ImprintHref and PrivacyHref are the link targets of the legal pages,
	// written into boardly-imprint and boardly-privacy meta tags; empty
	// leaves the tag out.
	ImprintHref string
	PrivacyHref string
	// ConsentService turns on the analytics consent banner and names the
	// service; empty means consent is off and the web app shows the cookie
	// notice instead. ConsentCookies and ConsentRevision go with it.
	ConsentService  string
	ConsentCookies  []string
	ConsentRevision int
}

// Handler serves the embedded frontend build with opts filled in.
func Handler(opts Options) http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return NewHandler(sub, opts)
}

// NewHandler serves files from fsys and falls back to index.html for paths
// that match no file, so client-side routes work on reload. index.html and the
// web app manifests are served with opts.AppTitle as the product name, and
// index.html carries the other options (see Options).
func NewHandler(fsys fs.FS, opts Options) http.Handler {
	titled, err := titledFiles(fsys, opts)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" && name != "index.html" && serveFile(w, r, fsys, titled, name) {
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		if !serveFile(w, r, fsys, titled, "index.html") {
			w.Header().Del("Cache-Control")
			http.Error(w, "frontend not built: run `make build` (or `pnpm build` in frontend/)", http.StatusServiceUnavailable)
		}
	})
}

// serveFile writes the named file and reports whether it existed. A file in
// titled is served with that content instead.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, titled map[string][]byte, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}
	content, ok := f.(io.ReadSeeker)
	if data, isTitled := titled[name]; isTitled {
		content = bytes.NewReader(data)
	} else if !ok {
		data, err := io.ReadAll(f)
		if err != nil {
			return false
		}
		content = strings.NewReader(string(data))
	}

	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	// Go's MIME table doesn't know web app manifests; ServeContent keeps a set type.
	// The URL isn't hashed, so a changed app title needs revalidation.
	if path.Ext(name) == ".webmanifest" {
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, name, info.ModTime(), content)
	return true
}

// titledFiles returns index.html and the root *.webmanifest files of fsys with
// the app title filled in, and index.html with the other options added, keyed
// by name. Files that don't exist are left out.
func titledFiles(fsys fs.FS, opts Options) (map[string][]byte, error) {
	appTitle, analyticsSnippet, mapStyleURL := opts.AppTitle, opts.AnalyticsSnippet, opts.MapStyleURL
	out := map[string][]byte{}

	if index, err := fs.ReadFile(fsys, "index.html"); err == nil {
		escaped := html.EscapeString(appTitle)
		index = bytes.Replace(index, []byte(titleMarker), []byte("<title>"+escaped+"</title>"), 1)
		index = bytes.Replace(index, []byte(metaMarker), []byte(`<meta name="application-name" content="`+escaped+`" />`), 1)
		if mapStyleURL != "" {
			index = bytes.Replace(index, []byte(mapStyleMarker),
				[]byte(`<meta name="boardly-map-style" content="`+html.EscapeString(mapStyleURL)+`" />`), 1)
		}
		var legal strings.Builder
		for _, m := range []struct{ name, href string }{
			{"boardly-imprint", opts.ImprintHref},
			{"boardly-privacy", opts.PrivacyHref},
		} {
			if m.href != "" {
				fmt.Fprintf(&legal, `<meta name="%s" content="%s" />`, m.name, html.EscapeString(m.href))
			}
		}
		writeConsentTags(&legal, opts)
		index = bytes.Replace(index, []byte(headEnd), []byte(legal.String()+headEnd), 1)
		if analyticsSnippet != "" {
			// The operator's HTML goes in as written; escaping would break it.
			index = bytes.Replace(index, []byte(headEnd), []byte(analyticsSnippet+headEnd), 1)
		}
		out["index.html"] = index
	}

	manifests, err := fs.Glob(fsys, "*.webmanifest")
	if err != nil {
		return nil, err
	}
	for _, name := range manifests {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		var manifest map[string]json.RawMessage
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		title, err := json.Marshal(appTitle)
		if err != nil {
			return nil, err
		}
		manifest["name"] = title
		manifest["short_name"] = title
		if out[name], err = json.MarshalIndent(manifest, "", "\t"); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// writeConsentTags writes the analytics consent settings when consent is on,
// and otherwise the cookie notice's variant: "necessary" without an analytics
// snippet and "cookieless-analytics" with one.
func writeConsentTags(b *strings.Builder, opts Options) {
	if opts.ConsentService == "" {
		variant := "necessary"
		if opts.AnalyticsSnippet != "" {
			variant = "cookieless-analytics"
		}
		fmt.Fprintf(b, `<meta name="boardly-cookie-notice" content="%s" />`, variant)
		return
	}
	fmt.Fprintf(b, `<meta name="boardly-consent-analytics" content="%s" />`, html.EscapeString(opts.ConsentService))
	if len(opts.ConsentCookies) > 0 {
		fmt.Fprintf(b, `<meta name="boardly-consent-cookies" content="%s" />`, html.EscapeString(strings.Join(opts.ConsentCookies, ",")))
	}
	fmt.Fprintf(b, `<meta name="boardly-consent-revision" content="%d" />`, opts.ConsentRevision)
}
