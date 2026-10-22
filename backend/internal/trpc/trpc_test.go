package trpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type authKey struct{}

func newServer() *Server {
	return &Server{
		Prefix: "/trpc/",
		Procedures: map[string]Procedure{
			"greet": {Kind: Query, Public: true, Handle: func(_ context.Context, in json.RawMessage) (any, error) {
				v, err := Decode[struct {
					Name string `json:"name"`
				}](in)
				if err != nil {
					return nil, err
				}
				return map[string]string{"hello": v.Name}, nil
			}},
			"secret": {Kind: Query, Handle: func(context.Context, json.RawMessage) (any, error) {
				return "s3cret", nil
			}},
			"save": {Kind: Mutation, Public: true, Handle: func(_ context.Context, in json.RawMessage) (any, error) {
				return map[string]any{"saved": in == nil}, nil
			}},
			"fail": {Kind: Mutation, Public: true, Handle: func(context.Context, json.RawMessage) (any, error) {
				return nil, errors.New("database exploded")
			}},
			"conflict": {Kind: Mutation, Public: true, Handle: func(context.Context, json.RawMessage) (any, error) {
				return nil, NewError("CONFLICT", "Already there")
			}},
		},
		Authorized: func(ctx context.Context) bool { return ctx.Value(authKey{}) != nil },
	}
}

func do(t *testing.T, s *Server, method, target, body string, authorized bool) (int, string) {
	t.Helper()
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if authorized {
		r = r.WithContext(context.WithValue(r.Context(), authKey{}, true))
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type = %q", ct)
	}
	return rec.Code, strings.TrimSpace(rec.Body.String())
}

func query(names, input string) string {
	return "/trpc/" + names + "?batch=1&input=" + url.QueryEscape(input)
}

func TestWireFormat(t *testing.T) {
	s := newServer()
	for name, tc := range map[string]struct {
		method, target, body string
		authorized           bool
		status               int
		want                 string
	}{
		"query": {
			"GET", query("greet", `{"0":{"json":{"name":"Ann"}}}`), "", false,
			200, `[{"result":{"data":{"json":{"hello":"Ann"}}}}]`,
		},
		"batch": {
			"GET", query("greet,secret", `{"0":{"json":{"name":"Ann"}}}`), "", true,
			200, `[{"result":{"data":{"json":{"hello":"Ann"}}}},{"result":{"data":{"json":"s3cret"}}}]`,
		},
		"mutation without input": {
			"POST", "/trpc/save?batch=1", `{}`, false,
			200, `[{"result":{"data":{"json":{"saved":true}}}}]`,
		},
		"superjson meta is ignored": {
			"POST", "/trpc/save?batch=1", `{"0":{"meta":{"values":["undefined"]}}}`, false,
			200, `[{"result":{"data":{"json":{"saved":true}}}}]`,
		},
		"unauthorized": {
			"GET", query("secret", `{}`), "", false,
			401, `[{"error":{"json":{"message":"Missing or invalid API key","code":-32001,"data":{"code":"UNAUTHORIZED","httpStatus":401,"path":"secret"}}}}]`,
		},
		"unknown procedure": {
			"GET", query("nope", `{}`), "", true,
			404, `[{"error":{"json":{"message":"No procedure found on path \"nope\"","code":-32004,"data":{"code":"NOT_FOUND","httpStatus":404,"path":"nope"}}}}]`,
		},
		"mixed statuses": {
			"GET", query("greet,nope", `{"0":{"json":{"name":"Ann"}}}`), "", false,
			207, `[{"result":{"data":{"json":{"hello":"Ann"}}}},{"error":{"json":{"message":"No procedure found on path \"nope\"","code":-32004,"data":{"code":"NOT_FOUND","httpStatus":404,"path":"nope"}}}}]`,
		},
		"mutation with GET": {
			"GET", query("save", `{}`), "", false,
			405, `[{"error":{"json":{"message":"Wrong method for this procedure","code":-32005,"data":{"code":"METHOD_NOT_SUPPORTED","httpStatus":405,"path":"save"}}}}]`,
		},
		"query with POST": {
			"POST", "/trpc/greet?batch=1", `{}`, false,
			405, `[{"error":{"json":{"message":"Wrong method for this procedure","code":-32005,"data":{"code":"METHOD_NOT_SUPPORTED","httpStatus":405,"path":"greet"}}}}]`,
		},
		"invalid input": {
			"GET", query("greet", `{"0":{"json":"Ann"}}`), "", false,
			400, `[{"error":{"json":{"message":"Invalid input","code":-32600,"data":{"code":"BAD_REQUEST","httpStatus":400,"path":"greet"}}}}]`,
		},
		"tRPC error from handler": {
			"POST", "/trpc/conflict?batch=1", `{}`, false,
			409, `[{"error":{"json":{"message":"Already there","code":-32009,"data":{"code":"CONFLICT","httpStatus":409,"path":"conflict"}}}}]`,
		},
		"other error from handler": {
			"POST", "/trpc/fail?batch=1", `{}`, false,
			500, `[{"error":{"json":{"message":"Internal server error","code":-32603,"data":{"code":"INTERNAL_SERVER_ERROR","httpStatus":500,"path":"fail"}}}}]`,
		},
		"not batched": {
			"GET", "/trpc/greet?input=" + url.QueryEscape(`{"json":{"name":"Ann"}}`), "", false,
			400, `[{"error":{"json":{"message":"Invalid batch request","code":-32600,"data":{"code":"BAD_REQUEST","httpStatus":400}}}}]`,
		},
		"invalid JSON": {
			"POST", "/trpc/save?batch=1", `{"0":`, false,
			400, `[{"error":{"json":{"message":"Invalid batch request","code":-32600,"data":{"code":"BAD_REQUEST","httpStatus":400}}}}]`,
		},
		"other method": {
			"PUT", "/trpc/save?batch=1", `{}`, false,
			405, `[{"error":{"json":{"message":"Method not supported","code":-32005,"data":{"code":"METHOD_NOT_SUPPORTED","httpStatus":405}}}}]`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := do(t, s, tc.method, tc.target, tc.body, tc.authorized)
			if status != tc.status || body != tc.want {
				t.Errorf("got %d %s\nwant %d %s", status, body, tc.status, tc.want)
			}
		})
	}
}
