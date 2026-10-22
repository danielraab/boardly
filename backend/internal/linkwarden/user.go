package linkwarden

import (
	"context"
	"slices"
	"strings"

	"famodash/internal/favourites"
	"famodash/internal/images"
	"famodash/internal/users"
)

// Version is the Linkwarden version whose API Famodash follows. The extension
// turns on paginated tags and tag search from 2.14.
const Version = "v2.16.3"

// DashboardSection is a section of the mobile app's dashboard.
type DashboardSection struct {
	ID           int    `json:"id"`
	UserID       int    `json:"userId"`
	CollectionID *int64 `json:"collectionId"`
	Type         string `json:"type"`
	Order        int    `json:"order"`
	CreatedAt    Time   `json:"createdAt"`
	UpdatedAt    Time   `json:"updatedAt"`
}

// Subscription is a user's subscription; Famodash has no billing.
type Subscription struct {
	Active   bool `json:"active"`
	Quantity int  `json:"quantity"`
}

// ParentSubscription is the subscription of a team owner; Famodash has none.
type ParentSubscription struct {
	Active bool     `json:"active"`
	User   struct{} `json:"user"`
}

// User is Linkwarden's GET /api/v1/users/me response.
type User struct {
	ID                      int                `json:"id"`
	UUID                    string             `json:"uuid"`
	Name                    *string            `json:"name"`
	Username                string             `json:"username"`
	Email                   string             `json:"email"`
	EmailVerified           Time               `json:"emailVerified"`
	UnverifiedNewEmail      *string            `json:"unverifiedNewEmail"`
	Image                   *string            `json:"image"`
	Locale                  string             `json:"locale"`
	ParentSubscriptionID    *int               `json:"parentSubscriptionId"`
	CollectionOrder         []int64            `json:"collectionOrder"`
	LinksRouteTo            string             `json:"linksRouteTo"`
	AiTaggingMethod         string             `json:"aiTaggingMethod"`
	AiPredefinedTags        []string           `json:"aiPredefinedTags"`
	AiTagExistingLinks      bool               `json:"aiTagExistingLinks"`
	Theme                   string             `json:"theme"`
	DismissedAnnouncementID *string            `json:"dismissedAnnouncementId"`
	ReadableFontFamily      string             `json:"readableFontFamily"`
	ReadableFontSize        string             `json:"readableFontSize"`
	ReadableLineHeight      string             `json:"readableLineHeight"`
	ReadableLineWidth       string             `json:"readableLineWidth"`
	PreventDuplicateLinks   bool               `json:"preventDuplicateLinks"`
	ArchiveAsScreenshot     bool               `json:"archiveAsScreenshot"`
	ArchiveAsMonolith       bool               `json:"archiveAsMonolith"`
	ArchiveAsPDF            bool               `json:"archiveAsPDF"`
	ArchiveAsReadable       bool               `json:"archiveAsReadable"`
	ArchiveAsWaybackMachine bool               `json:"archiveAsWaybackMachine"`
	IsPrivate               bool               `json:"isPrivate"`
	ReferredBy              *string            `json:"referredBy"`
	DashboardSections       []DashboardSection `json:"dashboardSections"`
	LastPickedAt            *Time              `json:"lastPickedAt"`
	AcceptPromotionalEmails bool               `json:"acceptPromotionalEmails"`
	TrialEndEmailSent       bool               `json:"trialEndEmailSent"`
	CreatedAt               Time               `json:"createdAt"`
	UpdatedAt               Time               `json:"updatedAt"`
	Subscription            Subscription       `json:"subscription"`
	ParentSubscription      ParentSubscription `json:"parentSubscription"`
	HasPassword             bool               `json:"hasPassword"`
	HasOAuthAccount         bool               `json:"hasOAuthAccount"`
}

// Me is GET /api/v1/users/me for the token's owner. Tapping a link opens the
// page (linksRouteTo ORIGINAL), because Famodash stores no page content.
func Me(u users.User) User {
	created := Time(u.CreatedAt)
	sections := []DashboardSection{
		{ID: 1, UserID: userID, Type: "STATS", Order: 0, CreatedAt: created, UpdatedAt: created},
		{ID: 2, UserID: userID, Type: "RECENT_LINKS", Order: 1, CreatedAt: created, UpdatedAt: created},
	}
	return User{
		ID: userID, UUID: u.ID, Name: u.Name, Username: u.Slug, Email: u.Email, EmailVerified: created,
		Locale: "en", CollectionOrder: []int64{}, LinksRouteTo: "ORIGINAL", AiTaggingMethod: "DISABLED",
		AiPredefinedTags: []string{}, Theme: "auto",
		ReadableFontFamily: "sans-serif", ReadableFontSize: "20px", ReadableLineHeight: "1.8", ReadableLineWidth: "normal",
		IsPrivate: true, DashboardSections: sections, CreatedAt: created, UpdatedAt: Time(u.UpdatedAt),
	}
}

// Config is GET /api/v1/config: every field of Linkwarden's configuration,
// with registration off and no billing, so the app never asks for a
// subscription.
func Config() map[string]any {
	config := map[string]any{"DISABLE_REGISTRATION": true, "INSTANCE_VERSION": Version}
	for _, key := range []string{
		"ADMIN", "RSS_POLLING_INTERVAL_MINUTES", "EMAIL_PROVIDER", "MAX_FILE_BUFFER", "USER_CONTENT_DOMAIN",
		"AI_ENABLED", "STRIPE_ENABLED", "STRIPE_BILLING_PORTAL_URL", "TRIAL_PERIOD_DAYS", "REQUIRE_CC", "DEMO",
		"DEMO_USERNAME", "DEMO_PASSWORD", "GOOGLE_ENABLED", "MOBILE_APP_REDIRECT_ENABLED",
	} {
		config[key] = nil
	}
	return config
}

// dashboardLinks is how many recent links the dashboard shows, as in Linkwarden.
const dashboardLinks = 16

// Dashboard is the data of GET /api/v2/dashboard.
type Dashboard struct {
	Links               []Link           `json:"links"`
	CollectionLinks     map[int64][]Link `json:"collectionLinks"`
	NumberOfPinnedLinks int              `json:"numberOfPinnedLinks"`
	NumberOfTags        int              `json:"numberOfTags"`
}

// Dashboard is GET /api/v2/dashboard: the newest links and the number of tags.
func (s *Service) Dashboard(ctx context.Context, sc favourites.Scope) (Dashboard, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return Dashboard{}, err
	}
	recent := t.links()
	sortLinks(recent, SortNewest)
	if len(recent) > dashboardLinks {
		recent = recent[:dashboardLinks]
	}
	links, err := s.mapLinks(ctx, t, recent)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{Links: links, CollectionLinks: map[int64][]Link{}, NumberOfTags: len(t.tags())}, nil
}

// Preview is GET /api/v1/archives/{id}?preview=true: the favicon of the
// link's origin when the app can show it, else ErrPreviewNotFound.
func (s *Service) Preview(ctx context.Context, sc favourites.Scope, num int64) (images.FaviconImage, error) {
	t, err := s.load(ctx, sc)
	if err != nil {
		return images.FaviconImage{}, err
	}
	n, err := t.link(num)
	if err != nil {
		return images.FaviconImage{}, err
	}
	origin, err := images.Origin(*n.URL)
	if err != nil {
		return images.FaviconImage{}, ErrPreviewNotFound
	}
	icons, err := s.Favicons.LookupImages(ctx, []string{origin})
	if err != nil {
		return images.FaviconImage{}, err
	}
	img, ok := icons[origin]
	if !ok || !slices.Contains(previewTypes, strings.ToLower(img.ContentType)) {
		return images.FaviconImage{}, ErrPreviewNotFound
	}
	return img, nil
}
