package httpapi

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// openAPIRoutes are the OpenAPI documents with the mux whose routes each one
// describes, and the prefix its paths are relative to (its server URL).
var openAPIRoutes = []struct {
	file   string
	prefix string
	mux    func(a *api) *routeMux
}{
	{"boardly.yaml", "/api", func(a *api) *routeMux { return a.mainAPI(fakeDB{}) }},
	{"karakeep.yaml", karakeepPrefix, (*api).karakeepAPI},
	{"linkwarden.yaml", linkwardenPrefix, (*api).linkwardenAPI},
	{"memos.yaml", memosPrefix, (*api).memosAPI},
}

// openAPIPrefixRoutes maps routes registered without a method to the
// operations that document them. Other routes without a method are the
// catch-alls that answer 404 and are not documented.
var openAPIPrefixRoutes = map[string][]string{
	karakeepPrefix + "/api/trpc/": {"get /api/trpc/{procedure}", "post /api/trpc/{procedure}"},
}

// openAPIAliases maps documented operations that are handled inside another
// route to that route, per document.
var openAPIAliases = map[string]map[string]string{
	// Memos puts the custom method into the last path segment, which the
	// users/{id} route tells apart.
	"memos.yaml": {"get /api/v1/users/{id}:getStats": "get /api/v1/users/{id}"},
}

var openAPIMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

func loadOpenAPIDoc(t *testing.T, file string) map[string]any {
	t.Helper()
	b, err := openAPIDocs.ReadFile("openapi/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return doc
}

// registeredOperations returns the routes of mux as "method path" with the
// prefix removed.
func registeredOperations(t *testing.T, mux *routeMux, prefix string) []string {
	t.Helper()
	var ops []string
	for _, p := range mux.patterns {
		method, path, ok := strings.Cut(p, " ")
		if !ok {
			ops = append(ops, openAPIPrefixRoutes[p]...)
			continue
		}
		rel, ok := strings.CutPrefix(path, prefix)
		if !ok {
			t.Errorf("route %q is outside %s", p, prefix)
			continue
		}
		ops = append(ops, strings.ToLower(method)+" "+rel)
	}
	return ops
}

// documentedOperations returns the operations of doc as "method path".
func documentedOperations(doc map[string]any) []string {
	var ops []string
	paths, _ := doc["paths"].(map[string]any)
	for path, item := range paths {
		item, _ := item.(map[string]any)
		for _, m := range openAPIMethods {
			if _, ok := item[m]; ok {
				ops = append(ops, m+" "+path)
			}
		}
	}
	return ops
}

func TestOpenAPIDocumentsMatchRoutes(t *testing.T) {
	for _, r := range openAPIRoutes {
		t.Run(r.file, func(t *testing.T) {
			registered := registeredOperations(t, r.mux(&api{}), r.prefix)
			documented := map[string]bool{}
			for _, op := range documentedOperations(loadOpenAPIDoc(t, r.file)) {
				if route, ok := openAPIAliases[r.file][op]; ok {
					op = route
				}
				documented[op] = true
			}
			for _, op := range registered {
				if !documented[op] {
					t.Errorf("route %q is not documented in %s", op, r.file)
				}
				delete(documented, op)
			}
			for _, op := range slices.Sorted(maps.Keys(documented)) {
				t.Errorf("%s documents %q, which is not a registered route", r.file, op)
			}
		})
	}
}

func TestOpenAPIKarakeepProcedures(t *testing.T) {
	doc := loadOpenAPIDoc(t, "karakeep.yaml")
	var documented []string
	paths, _ := doc["paths"].(map[string]any)
	item, _ := paths["/api/trpc/{procedure}"].(map[string]any)
	params, _ := item["parameters"].([]any)
	for _, p := range params {
		p, _ := p.(map[string]any)
		if p["name"] != "procedure" {
			continue
		}
		schema, _ := p["schema"].(map[string]any)
		enum, _ := schema["enum"].([]any)
		for _, v := range enum {
			s, _ := v.(string)
			documented = append(documented, s)
		}
	}
	slices.Sort(documented)
	implemented := slices.Sorted(maps.Keys((&api{}).karakeepProcedures()))
	for _, p := range implemented {
		if !slices.Contains(documented, p) {
			t.Errorf("tRPC procedure %q is not in the procedure enum", p)
		}
	}
	for _, p := range documented {
		if !slices.Contains(implemented, p) {
			t.Errorf("the procedure enum lists %q, which is not implemented", p)
		}
	}
}

func TestOpenAPIDocumentsAreValid(t *testing.T) {
	for _, r := range openAPIRoutes {
		t.Run(r.file, func(t *testing.T) {
			doc := loadOpenAPIDoc(t, r.file)
			if v, _ := doc["openapi"].(string); !strings.HasPrefix(v, "3.1.") {
				t.Errorf("openapi = %q, want 3.1.x", v)
			}
			checkOpenAPIRefs(t, doc, doc)
		})
	}
}

// checkOpenAPIRefs reports every local $ref below v that doesn't resolve in doc.
func checkOpenAPIRefs(t *testing.T, doc map[string]any, v any) {
	t.Helper()
	switch v := v.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
			var node any = doc
			for _, part := range strings.Split(ref[2:], "/") {
				part = strings.NewReplacer("~1", "/", "~0", "~").Replace(part)
				m, _ := node.(map[string]any)
				node = m[part]
			}
			if node == nil {
				t.Errorf("$ref %q doesn't resolve", ref)
			}
		}
		for _, child := range v {
			checkOpenAPIRefs(t, doc, child)
		}
	case []any:
		for _, child := range v {
			checkOpenAPIRefs(t, doc, child)
		}
	}
}

func TestServeOpenAPIDocuments(t *testing.T) {
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb})
	for _, c := range []struct{ path, file string }{
		{"/api/openapi.yaml", "boardly.yaml"},
		{"/api/karakeep/openapi.yaml", "karakeep.yaml"},
		{"/api/linkwarden/openapi.yaml", "linkwarden.yaml"},
		{"/api/memos/openapi.yaml", "memos.yaml"},
	} {
		rec := do(t, h, http.MethodGet, c.path)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", c.path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/yaml" {
			t.Errorf("%s: content type = %q", c.path, ct)
		}
		want, _ := openAPIDocs.ReadFile("openapi/" + c.file)
		if rec.Body.String() != string(want) {
			t.Errorf("%s: body isn't %s", c.path, c.file)
		}
		if c.file != "boardly.yaml" && rec.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: no CORS header", c.path)
		}
	}
}
