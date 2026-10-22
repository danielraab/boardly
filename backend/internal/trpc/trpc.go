// Package trpc serves procedures over the HTTP protocol of tRPC 11, as far as
// @trpc/client's httpBatchLink with the superjson transformer uses it: batched
// calls, inputs and outputs in superjson envelopes and tRPC's error format.
// Streaming, subscriptions and non-batched calls aren't supported.
package trpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// maxBody limits the size of a mutation's request body.
const maxBody = 1 << 20

// Kind is whether a procedure is a query (GET) or a mutation (POST).
type Kind int

const (
	Query Kind = iota
	Mutation
)

// Procedure is one tRPC procedure. Handle gets the procedure's input as raw
// JSON (nil when the call has none) and returns its output, or an error. A
// returned *Error is sent as is; any other error becomes an
// INTERNAL_SERVER_ERROR.
type Procedure struct {
	Kind Kind
	// Public procedures run without authorization.
	Public bool
	Handle func(ctx context.Context, input json.RawMessage) (any, error)
}

// Server serves the procedures at Prefix + "<name>,<name>…?batch=1".
type Server struct {
	Prefix     string
	Procedures map[string]Procedure
	// Authorized reports whether the request may call non-public procedures.
	Authorized func(ctx context.Context) bool
}

// Error is a tRPC error. Code is a tRPC error code such as "UNAUTHORIZED".
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// NewError returns a tRPC error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// ErrUnauthorized is the error of a non-public procedure without authorization.
var ErrUnauthorized = NewError("UNAUTHORIZED", "Missing or invalid API key")

// codes maps tRPC error codes to their JSON-RPC number and HTTP status, as in
// @trpc/server's rpc/codes.ts and http/getHTTPStatusCode.ts.
var codes = map[string]struct{ rpc, status int }{
	"PARSE_ERROR":           {-32700, 400},
	"BAD_REQUEST":           {-32600, 400},
	"UNAUTHORIZED":          {-32001, 401},
	"FORBIDDEN":             {-32003, 403},
	"NOT_FOUND":             {-32004, 404},
	"METHOD_NOT_SUPPORTED":  {-32005, 405},
	"CONFLICT":              {-32009, 409},
	"PAYLOAD_TOO_LARGE":     {-32013, 413},
	"TOO_MANY_REQUESTS":     {-32029, 429},
	"INTERNAL_SERVER_ERROR": {-32603, 500},
	"NOT_IMPLEMENTED":       {-32603, 501},
}

// Decode unmarshals a procedure's input into T. A missing or invalid input is
// a BAD_REQUEST error.
func Decode[T any](input json.RawMessage) (T, error) {
	var v T
	if len(input) == 0 || json.Unmarshal(input, &v) != nil {
		return v, NewError("BAD_REQUEST", "Invalid input")
	}
	return v, nil
}

type envelope struct {
	JSON json.RawMessage `json:"json,omitempty"`
}

type errorData struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"httpStatus"`
	Path       string `json:"path,omitempty"`
}

type errorShape struct {
	Message string    `json:"message"`
	Code    int       `json:"code"`
	Data    errorData `json:"data"`
}

type result struct {
	Data struct {
		JSON any            `json:"json"`
		Meta map[string]any `json:"meta,omitempty"`
	} `json:"data"`
}

// entry is one element of the response array: a result or an error.
type entry struct {
	Result *result `json:"result,omitempty"`
	Error  *struct {
		JSON errorShape `json:"json"`
	} `json:"error,omitempty"`
	status int
}

func errorEntry(e *Error, path string) entry {
	c, ok := codes[e.Code]
	if !ok {
		c = codes["INTERNAL_SERVER_ERROR"]
	}
	en := entry{status: c.status}
	en.Error = &struct {
		JSON errorShape `json:"json"`
	}{errorShape{e.Message, c.rpc, errorData{e.Code, c.status, path}}}
	return en
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	names := strings.Split(strings.TrimPrefix(r.URL.Path, s.Prefix), ",")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		write(w, []entry{errorEntry(NewError("METHOD_NOT_SUPPORTED", "Method not supported"), "")})
		return
	}
	inputs, err := readInputs(r)
	if err != nil || r.URL.Query().Get("batch") != "1" {
		write(w, []entry{errorEntry(NewError("BAD_REQUEST", "Invalid batch request"), "")})
		return
	}

	entries := make([]entry, len(names))
	for i, name := range names {
		var in envelope
		if raw, ok := inputs[strconv.Itoa(i)]; ok {
			if json.Unmarshal(raw, &in) != nil {
				entries[i] = errorEntry(NewError("BAD_REQUEST", "Invalid input"), name)
				continue
			}
		}
		entries[i] = s.call(r, name, in.JSON)
	}
	write(w, entries)
}

func (s *Server) call(r *http.Request, name string, input json.RawMessage) entry {
	p, ok := s.Procedures[name]
	if !ok {
		return errorEntry(NewError("NOT_FOUND", "No procedure found on path \""+name+"\""), name)
	}
	if (p.Kind == Query) != (r.Method == http.MethodGet) {
		return errorEntry(NewError("METHOD_NOT_SUPPORTED", "Wrong method for this procedure"), name)
	}
	if !p.Public && !s.Authorized(r.Context()) {
		return errorEntry(ErrUnauthorized, name)
	}
	out, err := p.Handle(r.Context(), input)
	if err != nil {
		var e *Error
		if !errors.As(err, &e) {
			slog.ErrorContext(r.Context(), "tRPC procedure failed", "path", name, "error", err)
			e = NewError("INTERNAL_SERVER_ERROR", "Internal server error")
		}
		return errorEntry(e, name)
	}
	res := &result{}
	res.Data.JSON = out
	res.Data.Meta = meta(out)
	return entry{Result: res, status: http.StatusOK}
}

// readInputs returns the batch's inputs keyed by call index: from the "input"
// query parameter of a query, or the body of a mutation.
func readInputs(r *http.Request) (map[string]json.RawMessage, error) {
	var raw []byte
	if r.Method == http.MethodGet {
		raw = []byte(r.URL.Query().Get("input"))
	} else {
		b, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		if err != nil {
			return nil, err
		}
		if len(b) > maxBody {
			return nil, errors.New("body too large")
		}
		raw = b
	}
	inputs := map[string]json.RawMessage{}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return inputs, nil
	}
	if err := json.Unmarshal(raw, &inputs); err != nil {
		return nil, err
	}
	return inputs, nil
}

// write sends the entries with the status they share, or 207 when they differ.
func write(w http.ResponseWriter, entries []entry) {
	status := entries[0].status
	for _, e := range entries[1:] {
		if e.status != status {
			status = http.StatusMultiStatus
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		slog.Error("writing tRPC response", "error", err)
	}
}
