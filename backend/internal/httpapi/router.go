// Package httpapi wires the REST API and the web app into one HTTP handler.
package httpapi

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"

	"boardly/internal/accounts"
	"boardly/internal/apptokens"
	"boardly/internal/attachments"
	"boardly/internal/auth"
	"boardly/internal/dashboards"
	"boardly/internal/dataimport"
	"boardly/internal/export"
	"boardly/internal/favourites"
	"boardly/internal/feeds"
	"boardly/internal/groups"
	"boardly/internal/images"
	"boardly/internal/invitations"
	"boardly/internal/legal"
	"boardly/internal/media"
	"boardly/internal/notes"
	"boardly/internal/offline"
	"boardly/internal/ratelimit"
	"boardly/internal/shares"
	"boardly/internal/users"
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Deps are the dependencies of the HTTP handler.
type Deps struct {
	DB  Pinger
	Web http.Handler // serves the web app for every non-API path

	Auth       *auth.Service
	Users      *users.Store
	Favourites *favourites.Store
	Dashboards *dashboards.Store
	Notes      *notes.Store
	Media      *media.Store
	Feeds      *feeds.Store
	// Groups, Shares and Invitations let users give others access.
	Groups      *groups.Store
	Shares      *shares.Store
	Invitations *invitations.Store
	// Directory searches radio-browser.info for the "Add station" dialog.
	Directory *media.Directory
	Images    *images.Store
	// Attachments stores the files attached to notes.
	Attachments *attachments.Store
	Favicons    *images.Favicons
	// Previews takes preview images that connected apps send.
	Previews *images.Previews
	// Offline holds the offline copies of links.
	Offline   *offline.Store
	AppTokens *apptokens.Store
	// Accounts schedules and cancels account deletions.
	Accounts *accounts.Service
	// Export builds the download of all data of a user.
	Export *export.Service
	// DataImport imports a Boardly export into a user's account.
	DataImport *dataimport.Service
	// ExportsPerHour and ExportsPerDay limit exports per user; 0 means no limit.
	ExportsPerHour int
	ExportsPerDay  int
	// Origin is the web app's origin (scheme://host[:port]); state-changing
	// requests with another Origin header are rejected.
	Origin string
	// SecureCookies sets the Secure attribute on the session cookie.
	SecureCookies bool
	// TrustProxyHeaders takes the client IP from X-Forwarded-For.
	TrustProxyHeaders bool
	// LogRequests picks which requests get a log line: "all", "errors" or "none".
	LogRequests string
	// Legal holds the imprint and privacy policy; nil means none.
	Legal *legal.Pages
}

// New returns the root handler: /api/* goes to the API, everything else to the web app.
func New(d Deps) http.Handler {
	a := &api{auth: d.Auth, users: d.Users, favourites: d.Favourites, dashboards: d.Dashboards, notes: d.Notes, media: d.Media, feeds: d.Feeds, groups: d.Groups, shares: d.Shares, invitations: d.Invitations, invitationLimiter: ratelimit.NewWindow(invitationsPerHour, 0), feedLimiter: ratelimit.New(feedAddsPerHour, feedAddBurst, 10_000), directory: d.Directory, images: d.Images, attachments: d.Attachments, favicons: d.Favicons, previews: d.Previews, offline: d.Offline, offlineLimiter: ratelimit.New(offlineRefreshesPerHour, offlineRefreshBurst, 10_000), refreshLimiter: ratelimit.New(refreshesPerHour, refreshBurst, 10_000), searchLimiter: ratelimit.New(searchesPerHour, searchBurst, 10_000), appTokens: d.AppTokens, accounts: d.Accounts, export: d.Export, dataImport: d.DataImport, exportLimiter: ratelimit.NewWindow(d.ExportsPerHour, d.ExportsPerDay), origin: d.Origin, secureCookies: d.SecureCookies, trustProxy: d.TrustProxyHeaders, legal: d.Legal}
	// Logo links of station searches are signed with a key that lives as long
	// as the process; they are only used while the dialog is open.
	a.logoKey = make([]byte, 32)
	if _, err := rand.Read(a.logoKey); err != nil {
		panic(err)
	}

	mux := a.mainAPI(d.DB)

	root := http.NewServeMux()
	root.Handle("/api/", jsonMuxErrors(mux.ServeMux, d.Origin))
	root.Handle(karakeepPrefix+"/", a.appAPI(apptokens.AppKarakeep, a.karakeepAPI()))
	root.Handle(linkwardenPrefix+"/", a.appAPI(apptokens.AppLinkwarden, a.linkwardenAPI()))
	root.Handle(memosPrefix+"/", a.appAPI(apptokens.AppMemos, a.memosAPI()))
	root.Handle("/", d.Web)
	return logRequests(d.LogRequests, root)
}

// mainAPI registers the routes of Boardly's own API under /api.
func (a *api) mainAPI(db Pinger) *routeMux {
	mux := newRouteMux()
	mux.Handle("GET /api/health", health(db))
	mux.Handle("GET /api/openapi.yaml", openAPIDoc("boardly.yaml"))
	mux.Handle("GET /api/legal/{page}", HandlerFunc(a.getLegalPage))
	mux.Handle("POST /api/auth/magic-link", HandlerFunc(a.requestLink))
	mux.Handle("POST /api/auth/magic-link/verify", HandlerFunc(a.verifyLink))
	mux.Handle("POST /api/auth/magic-link/code", HandlerFunc(a.redeemCode))
	mux.Handle("POST /api/auth/logout", HandlerFunc(a.logout))
	mux.Handle("GET /api/me", a.requireUser(a.getMe))
	mux.Handle("PATCH /api/me", a.requireUser(a.patchMe))
	mux.Handle("DELETE /api/me", a.requireUser(a.deleteMe))
	mux.Handle("POST /api/me/restore", a.requireUser(a.restoreMe))
	mux.Handle("GET /api/me/export", a.requireUser(a.exportData))
	mux.Handle("POST /api/import", a.requireUser(a.importData))
	mux.Handle("POST /api/auth/passkey/login/options", HandlerFunc(a.passkeyLoginOptions))
	mux.Handle("POST /api/auth/passkey/login/verify", HandlerFunc(a.passkeyLoginVerify))
	mux.Handle("GET /api/passkeys", a.requireUser(a.listPasskeys))
	mux.Handle("POST /api/passkeys/registration/options", a.requireUser(a.passkeyRegistrationOptions))
	mux.Handle("POST /api/passkeys/registration/verify", a.requireUser(a.passkeyRegistrationVerify))
	mux.Handle("DELETE /api/passkeys/{id}", a.requireUser(a.deletePasskey))
	mux.Handle("GET /api/favourites", a.requireUser(a.getFavourites))
	mux.Handle("POST /api/favourites", a.requireUser(a.createFavourite))
	mux.Handle("PATCH /api/favourites/{id}", a.requireUser(a.updateFavourite))
	mux.Handle("POST /api/favourites/bulk", a.requireUser(a.bulkFavourites))
	mux.Handle("POST /api/favourites/{id}/move", a.requireUser(a.moveFavourite))
	mux.Handle("DELETE /api/favourites/{id}", a.requireUser(a.deleteFavourite))
	mux.Handle("GET /api/favourites/{id}/offline", a.requireUser(a.getOfflineCopy))
	mux.Handle("POST /api/favourites/{id}/offline", a.requireUser(a.refreshOfflineCopy))
	mux.Handle("GET /api/favourites/{id}/offline/article", a.requireUser(a.getOfflineArticle))
	mux.Handle("GET /api/favourites/{id}/offline/screenshot", a.requireUser(a.getOfflineScreenshot))
	mux.Handle("POST /api/favourites/import", a.requireUser(a.importFavourites))
	mux.Handle("GET /api/tags", a.requireUser(a.suggestTags))
	mux.Handle("PATCH /api/tags/{id}", a.requireUser(a.renameTag))
	mux.Handle("DELETE /api/tags/{id}", a.requireUser(a.deleteTag))
	mux.Handle("GET /api/dashboards", a.requireUser(a.listDashboards))
	mux.Handle("POST /api/dashboards", a.requireUser(a.createDashboard))
	mux.Handle("PATCH /api/dashboards/{id}", a.requireUser(a.updateDashboard))
	mux.Handle("DELETE /api/dashboards/{id}", a.requireUser(a.deleteDashboard))
	mux.Handle("PUT /api/dashboards/{id}/layout", a.requireUser(a.saveLayout))
	mux.Handle("GET /api/notes", a.requireUser(a.listNotes))
	mux.Handle("POST /api/notes", a.requireUser(a.createNote))
	mux.Handle("GET /api/notes/{id}", a.requireUser(a.getNote))
	mux.Handle("PATCH /api/notes/{id}", a.requireUser(a.updateNote))
	mux.Handle("DELETE /api/notes/{id}", a.requireUser(a.deleteNote))
	mux.Handle("GET /api/note-categories", a.requireUser(a.listNoteCategories))
	mux.Handle("POST /api/note-categories", a.requireUser(a.createNoteCategory))
	mux.Handle("PATCH /api/note-categories/{id}", a.requireUser(a.updateNoteCategory))
	mux.Handle("DELETE /api/note-categories/{id}", a.requireUser(a.deleteNoteCategory))
	mux.Handle("GET /api/feeds", a.requireUser(a.listFeeds))
	mux.Handle("POST /api/feeds", a.requireUser(a.createFeed))
	mux.Handle("PUT /api/feeds/order", a.requireUser(a.reorderFeeds))
	mux.Handle("PATCH /api/feeds/{id}", a.requireUser(a.updateFeed))
	mux.Handle("DELETE /api/feeds/{id}", a.requireUser(a.deleteFeed))
	mux.Handle("GET /api/feed-items", a.requireUser(a.listFeedItems))
	mux.Handle("PATCH /api/feed-items/{id}", a.requireUser(a.updateFeedItem))
	mux.Handle("GET /api/media", a.requireUser(a.listMedia))
	mux.Handle("POST /api/media", a.requireUser(a.createMedia))
	mux.Handle("PUT /api/media/order", a.requireUser(a.reorderMedia))
	mux.Handle("GET /api/media/search", a.requireUser(a.searchStations))
	mux.Handle("GET /api/media/search/logo", a.requireUser(a.searchLogo))
	mux.Handle("PATCH /api/media/{id}", a.requireUser(a.updateMedia))
	mux.Handle("DELETE /api/media/{id}", a.requireUser(a.deleteMedia))
	mux.Handle("GET /api/u/{userSlug}/{dashboardSlug}", a.optionalUser(a.viewDashboard))
	mux.Handle("GET /api/groups", a.requireUser(a.listGroups))
	mux.Handle("POST /api/groups", a.requireUser(a.createGroup))
	mux.Handle("GET /api/groups/{id}", a.requireUser(a.getGroup))
	mux.Handle("PATCH /api/groups/{id}", a.requireUser(a.updateGroup))
	mux.Handle("DELETE /api/groups/{id}", a.requireUser(a.deleteGroup))
	mux.Handle("PATCH /api/groups/{id}/members/{userId}", a.requireUser(a.updateGroupMember))
	mux.Handle("DELETE /api/groups/{id}/members/{userId}", a.requireUser(a.removeGroupMember))
	mux.Handle("POST /api/groups/{id}/invitations", a.requireUser(a.inviteToGroup))
	mux.Handle("DELETE /api/groups/{id}/invitations/{invitationId}", a.requireUser(a.revokeGroupInvitation))
	mux.Handle("GET /api/shares/{type}/{id}", a.requireUser(a.listShares))
	mux.Handle("POST /api/shares/{type}/{id}", a.requireUser(a.createShare))
	mux.Handle("DELETE /api/shares/{type}/{id}/{shareId}", a.requireUser(a.deleteShare))
	mux.Handle("POST /api/shares/dashboard/{id}/invitations", a.requireUser(a.inviteToDashboard))
	mux.Handle("DELETE /api/shares/dashboard/{id}/invitations/{invitationId}", a.requireUser(a.revokeDashboardInvitation))
	mux.Handle("GET /api/shared-with-me", a.requireUser(a.sharedWithMe))
	mux.Handle("GET /api/invitations", a.requireUser(a.listInvitations))
	mux.Handle("POST /api/invitations/preview", a.requireUser(a.previewInvitation))
	mux.Handle("POST /api/invitations/accept", a.requireUser(a.acceptInvitation))
	mux.Handle("GET /api/images/{id}", HandlerFunc(a.getImage))
	mux.Handle("POST /api/images", a.requireUser(a.uploadImage))
	mux.Handle("GET /api/attachments", a.requireUser(a.listAttachments))
	mux.Handle("POST /api/attachments", a.requireUser(a.uploadAttachment))
	mux.Handle("PATCH /api/attachments/{id}", a.requireUser(a.patchAttachment))
	mux.Handle("DELETE /api/attachments/{id}", a.requireUser(a.deleteAttachment))
	mux.Handle("GET /api/attachments/{id}", HandlerFunc(a.getAttachment))
	mux.Handle("GET /api/attachments/{id}/{name}", HandlerFunc(a.getAttachment))
	mux.Handle("POST /api/favicons", a.requireUser(a.getFavicon))
	mux.Handle("POST /api/favicons/refresh", a.requireUser(a.refreshFavicon))
	mux.Handle("GET /api/app-tokens", a.requireUser(a.listAppTokens))
	mux.Handle("POST /api/app-tokens", a.requireUser(a.createAppToken))
	mux.Handle("PATCH /api/app-tokens/{id}", a.requireUser(a.updateAppToken))
	mux.Handle("POST /api/app-tokens/{id}/revoke", a.requireUser(a.revokeAppToken))
	mux.Handle("DELETE /api/app-tokens/{id}", a.requireUser(a.deleteAppToken))

	return mux
}

func health(db Pinger) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := db.PingContext(r.Context()); err != nil {
			return NewError(http.StatusServiceUnavailable, "unavailable", "Database is not reachable")
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return nil
	}
}

// jsonMuxErrors turns the plain-text 404 and 405 responses that http.ServeMux
// writes for unmatched requests into the API's JSON error format. Matched
// requests pass the cross-site request check first.
func jsonMuxErrors(mux *http.ServeMux, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			if err := checkCSRF(r, origin); err != nil {
				WriteError(w, err)
				return
			}
			mux.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(&muxErrorWriter{ResponseWriter: w}, r)
	})
}

type muxErrorWriter struct {
	http.ResponseWriter
	replaced bool
}

func (m *muxErrorWriter) WriteHeader(status int) {
	if !strings.HasPrefix(m.Header().Get("Content-Type"), "text/plain") {
		m.ResponseWriter.WriteHeader(status)
		return
	}
	m.replaced = true
	m.Header().Del("Content-Length")
	switch status {
	case http.StatusMethodNotAllowed:
		WriteError(m.ResponseWriter, NewError(status, "method_not_allowed", "Method not allowed"))
	case http.StatusNotFound:
		WriteError(m.ResponseWriter, NewError(status, "not_found", "Not found"))
	default:
		WriteError(m.ResponseWriter, NewError(status, "error", http.StatusText(status)))
	}
}

func (m *muxErrorWriter) Write(b []byte) (int, error) {
	if m.replaced {
		return len(b), nil
	}
	return m.ResponseWriter.Write(b)
}
