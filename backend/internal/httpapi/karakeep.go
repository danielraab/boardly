package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"famodash/internal/apptokens"
	"famodash/internal/trpc"
	"famodash/internal/users"
)

// karakeepPrefix is the server address that Karakeep clients are given; they
// append Karakeep's own paths (api/trpc, api/health, …) to it.
const karakeepPrefix = "/api/karakeep"

// karakeepAPI serves what the Karakeep browser extension and mobile app call:
// their tRPC procedures and two status endpoints. Nothing else of Karakeep's
// API is implemented.
func (a *api) karakeepAPI() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET "+karakeepPrefix+"/api/health", HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Famodash is working"})
		return nil
	}))
	mux.Handle("GET "+karakeepPrefix+"/api/version", HandlerFunc(func(w http.ResponseWriter, _ *http.Request) error {
		WriteJSON(w, http.StatusOK, map[string]string{"version": "Famodash"})
		return nil
	}))
	mux.Handle(karakeepPrefix+"/api/trpc/", &trpc.Server{
		Prefix: karakeepPrefix + "/api/trpc/",
		Procedures: map[string]trpc.Procedure{
			"apiKeys.validate":    {Kind: trpc.Mutation, Public: true, Handle: a.karakeepValidateKey},
			"apiKeys.exchange":    {Kind: trpc.Mutation, Public: true, Handle: karakeepExchange},
			"users.whoami":        {Kind: trpc.Query, Handle: a.karakeepWhoAmI},
			"config.clientConfig": {Kind: trpc.Query, Public: true, Handle: a.karakeepClientConfig},
		},
		Authorized: func(ctx context.Context) bool {
			_, ok := appToken(ctx)
			return ok
		},
	})
	return mux
}

// karakeepValidateKey is apiKeys.validate: the login with an API key, which
// clients send in the input before they use it as their bearer token.
func (a *api) karakeepValidateKey(ctx context.Context, input json.RawMessage) (any, error) {
	in, err := trpc.Decode[struct {
		APIKey string `json:"apiKey"`
	}](input)
	if err != nil {
		return nil, err
	}
	if _, err := a.appTokens.Authenticate(ctx, in.APIKey, apptokens.AppKarakeep); err != nil {
		if errors.Is(err, apptokens.ErrInvalidToken) {
			return nil, trpc.ErrUnauthorized
		}
		return nil, err
	}
	return map[string]bool{"success": true}, nil
}

// karakeepExchange is apiKeys.exchange, the login with email and password,
// which Famodash doesn't have.
func karakeepExchange(context.Context, json.RawMessage) (any, error) {
	return nil, trpc.NewError("UNAUTHORIZED", "Famodash has no passwords. Log in with an API key from Settings → Connected apps.")
}

// karakeepWhoAmI is users.whoami: the token's owner.
func (a *api) karakeepWhoAmI(ctx context.Context, _ json.RawMessage) (any, error) {
	token, _ := appToken(ctx)
	user, err := a.users.Get(ctx, token.UserID)
	if errors.Is(err, users.ErrNotFound) {
		return nil, trpc.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return struct {
		ID        string  `json:"id"`
		Name      *string `json:"name"`
		Email     string  `json:"email"`
		Image     *string `json:"image"`
		LocalUser bool    `json:"localUser"`
	}{user.ID, user.Name, user.Email, nil, true}, nil
}

// karakeepClientConfig is config.clientConfig. It has every field of
// Karakeep's client configuration, because the apps read nested fields without
// checks, and turns off everything Famodash doesn't offer.
func (a *api) karakeepClientConfig(context.Context, json.RawMessage) (any, error) {
	return map[string]any{
		"publicUrl":    a.origin,
		"publicApiUrl": a.origin + karakeepPrefix + "/api",
		"auth": map[string]bool{
			"disableSignups":      true,
			"disablePasswordAuth": true,
			"oauthAutoRedirect":   false,
		},
		"turnstile": nil,
		"inference": map[string]any{
			"isConfigured":            false,
			"inferredTagLang":         "english",
			"enableAutoTagging":       false,
			"enableAutoSummarization": false,
		},
		"chat":                   map[string]bool{"enabled": false},
		"search":                 map[string]bool{"semanticSearchEnabled": false},
		"stripe":                 map[string]bool{"isConfigured": false},
		"crawler":                map[string]bool{"videoDownloadOnDemand": false},
		"legal":                  map[string]any{},
		"disableNewReleaseCheck": true,
	}, nil
}
