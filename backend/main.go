// Command boardly runs the Boardly server: the REST API and the embedded web app.
// `boardly healthcheck` checks a running server and exits 0 if it is healthy.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"boardly/internal/aaguid"
	"boardly/internal/accounts"
	"boardly/internal/apptokens"
	"boardly/internal/attachments"
	"boardly/internal/auth"
	"boardly/internal/blobs"
	"boardly/internal/config"
	"boardly/internal/consent"
	"boardly/internal/dashboards"
	"boardly/internal/dataimport"
	"boardly/internal/db"
	"boardly/internal/export"
	"boardly/internal/favourites"
	"boardly/internal/feeds"
	"boardly/internal/httpapi"
	"boardly/internal/images"
	"boardly/internal/jobs"
	"boardly/internal/legal"
	"boardly/internal/mail"
	"boardly/internal/media"
	"boardly/internal/notes"
	"boardly/internal/offline"
	"boardly/internal/users"
	"boardly/internal/web"
	"boardly/internal/widgets"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(os.LookupEnv); err != nil {
			fmt.Fprintln(os.Stderr, "unhealthy:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// loadConfig reads the configuration and the legal pages it names, and checks
// the analytics consent settings against both. Any failure stops the server
// before anything else starts.
func loadConfig(lookupEnv func(string) (string, bool)) (config.Config, *legal.Pages, error) {
	cfg, err := config.Load(lookupEnv)
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("configuration: %w", err)
	}
	pages, err := legal.Load(legal.Sources{Dir: cfg.Legal.Dir, ImprintURL: cfg.Legal.ImprintURL, PrivacyURL: cfg.Legal.PrivacyURL})
	if err != nil {
		return config.Config{}, nil, fmt.Errorf("configuration: %w", err)
	}
	if err := consent.Check(cfg.Consent, cfg.AnalyticsSnippet, pages.Href(legal.Privacy)); err != nil {
		return config.Config{}, nil, fmt.Errorf("configuration: %w", err)
	}
	return cfg, pages, nil
}

func run() error {
	cfg, legalPages, err := loadConfig(os.LookupEnv)
	if err != nil {
		return err
	}
	setupLogging(cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := db.Migrate(ctx, database, db.Migrations); err != nil {
		return err
	}

	userStore := users.NewStore(database)
	mailer := mail.New(cfg.SMTP)
	authService := auth.NewService(database, userStore, mailer, cfg.BaseURL)
	authService.AppTitle = cfg.AppTitle
	authService.RegistrationClosed = !cfg.AllowRegistration
	if authService.WebAuthn, err = auth.NewWebAuthn(cfg.WebAuthnRPID, cfg.Origin()); err != nil {
		return err
	}
	if cfg.AAGUIDListURL != "" {
		providers := aaguid.New(cfg.AAGUIDListURL, filepath.Join(filepath.Dir(cfg.DBPath), "aaguids.json"))
		authService.Providers = providers
		go providers.Run(ctx)
	}
	go authService.RunJanitor(ctx, time.Hour)

	imageStore, err := images.NewStore(database, cfg.DataDir)
	if err != nil {
		return err
	}
	queue := jobs.New(database)
	favicons := images.NewFavicons(imageStore, cfg.FetchAllowPrivate)
	favicons.Jobs = queue
	previews, err := images.NewPreviews(imageStore, queue, favicons.Client, cfg.Previews.MaxBytes,
		images.Converter{MaxPixels: cfg.Previews.MaxPixels, Width: cfg.Previews.Width})
	if err != nil {
		return err
	}
	offlineStore, err := offline.NewStore(database, cfg.DataDir)
	if err != nil {
		return err
	}
	capturer := &offline.Capturer{
		Store: offlineStore, Jobs: queue, Client: favicons.Client, MaxBytes: cfg.Offline.MaxBytes,
		Converter: images.Converter{MaxPixels: cfg.Previews.MaxPixels, Width: offline.ScreenshotWidth},
		Previews:  previews,
	}
	if cfg.Offline.BrowserURL != "" {
		capturer.Browser = offline.NewBrowser(cfg.Offline.BrowserURL, images.GuardedTransport(cfg.FetchAllowPrivate))
	}
	if err := startJobs(ctx, queue, favicons, previews, capturer); err != nil {
		return err
	}
	go imageStore.RunCleanup(ctx, time.Hour)
	go offlineStore.RunCleanup(ctx, time.Hour)

	attachmentStore, err := attachments.NewStore(database, cfg.DataDir,
		cfg.Attachments.MaxBytes, cfg.Attachments.Types, cfg.Attachments.DeleteAfterDays)
	if err != nil {
		return err
	}
	go attachmentStore.RunCleanup(ctx, time.Hour)

	accountService := &accounts.Service{
		DB: database, Users: userStore, Now: time.Now, Days: cfg.AccountDeletionDays,
		Mailer: mailer, AppTitle: cfg.AppTitle, Link: cfg.BaseURL.String() + "/",
		ImageFiles: blobs.Dir(imageStore.Dir), AttachmentFiles: attachmentStore.Files, OfflineFiles: offlineStore.Files,
	}
	go accountService.RunSweep(ctx, time.Hour)

	favouriteStore := favourites.NewStore(database)
	favouriteStore.Jobs = queue
	dashboardStore := dashboards.NewStore(database)
	noteStore := notes.NewStore(database)
	mediaStore := media.NewStore(database, media.NewChecker(cfg.FetchAllowPrivate).Check, favicons.FetchLogo)
	feedStore := feeds.NewStore(database, feeds.NewFetcher(cfg.FetchAllowPrivate), favicons,
		time.Duration(cfg.Feeds.RefreshMinutes)*time.Minute, cfg.Feeds.LatestItems)
	go feedStore.Run(ctx, time.Minute, time.Hour)
	widgets.Register(dashboardStore, favouriteStore, noteStore, mediaStore, feedStore, favicons)

	srv := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port),
		Handler: httpapi.New(httpapi.Deps{
			DB: database,
			Web: web.Handler(web.Options{
				AppTitle: cfg.AppTitle, AnalyticsSnippet: cfg.AnalyticsSnippet, MapStyleURL: cfg.MapStyleURL,
				ImprintHref: legalPages.Href(legal.Imprint), PrivacyHref: legalPages.Href(legal.Privacy),
				ConsentService: cfg.Consent.Service, ConsentCookies: cfg.Consent.Cookies,
				ConsentRevision: consent.Revision(cfg.AnalyticsSnippet, cfg.Consent.Service, cfg.Consent.Cookies),
			}),
			Auth:        authService,
			Legal:       legalPages,
			Users:       userStore,
			Favourites:  favouriteStore,
			Dashboards:  dashboardStore,
			Notes:       noteStore,
			Media:       mediaStore,
			Feeds:       feedStore,
			Images:      imageStore,
			Attachments: attachmentStore,
			Favicons:    favicons,
			Directory:   media.NewDirectory(cfg.FetchAllowPrivate),
			Previews:    previews,
			Offline:     offlineStore,
			AppTokens:   apptokens.NewStore(database),
			Accounts:    accountService,
			Export: &export.Service{
				DB: database, ImageFiles: blobs.Dir(imageStore.Dir), AttachmentFiles: attachmentStore.Files,
				OfflineFiles: offlineStore.Files, Now: time.Now,
			},
			DataImport: &dataimport.Service{
				DB: database, Favourites: favouriteStore, Notes: noteStore, Dashboards: dashboardStore,
				Media: mediaStore, Feeds: feedStore, Images: imageStore, Attachments: attachmentStore,
				OfflineFiles: offlineStore.Files, OfflineMaxBytes: cfg.Offline.MaxBytes,
				Screenshots: images.Converter{MaxPixels: cfg.Previews.MaxPixels, Width: offline.ScreenshotWidth},
			},
			ExportsPerHour:    cfg.Exports.PerHour,
			ExportsPerDay:     cfg.Exports.PerDay,
			Origin:            cfg.Origin(),
			SecureCookies:     cfg.SecureCookies(),
			TrustProxyHeaders: cfg.TrustProxyHeaders,
			LogRequests:       cfg.LogRequests,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "db", cfg.DBPath, "data_dir", cfg.DataDir, "base_url", cfg.BaseURL.String(), "webauthn_rp_id", cfg.WebAuthnRPID)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("shutdown complete")
	return nil
}

// startJobs queues the jobs that were running when the server stopped again
// and starts the workers of every job kind.
func startJobs(ctx context.Context, queue *jobs.Queue, favicons *images.Favicons, previews *images.Previews, capturer *offline.Capturer) error {
	if err := queue.Recover(ctx); err != nil {
		return fmt.Errorf("recovering background jobs: %w", err)
	}
	favicons.Run(ctx)
	previews.Run(ctx)
	capturer.Run(ctx)
	return nil
}

func setupLogging(format string) {
	var h slog.Handler = slog.NewJSONHandler(os.Stderr, nil)
	if format == "text" {
		h = slog.NewTextHandler(os.Stderr, nil)
	}
	slog.SetDefault(slog.New(h))
}
