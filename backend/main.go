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
	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/config"
	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/favourites"
	"boardly/internal/httpapi"
	"boardly/internal/images"
	"boardly/internal/mail"
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

func run() error {
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
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
	authService := auth.NewService(database, userStore, mail.New(cfg.SMTP), cfg.BaseURL)
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
	favicons := images.NewFavicons(imageStore, cfg.FetchAllowPrivate)
	go favicons.Run(ctx)
	go func() {
		if err := favicons.Backfill(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "queueing missing favicons failed", "error", err)
		}
	}()
	go imageStore.RunCleanup(ctx, time.Hour)

	favouriteStore := favourites.NewStore(database)
	dashboardStore := dashboards.NewStore(database)
	widgets.Register(dashboardStore, favouriteStore, favicons)

	srv := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port),
		Handler: httpapi.New(httpapi.Deps{
			DB:                database,
			Web:               web.Handler(),
			Auth:              authService,
			Users:             userStore,
			Favourites:        favouriteStore,
			Dashboards:        dashboardStore,
			Images:            imageStore,
			Favicons:          favicons,
			AppTokens:         apptokens.NewStore(database),
			Origin:            cfg.Origin(),
			SecureCookies:     cfg.SecureCookies(),
			TrustProxyHeaders: cfg.TrustProxyHeaders,
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

func setupLogging(format string) {
	var h slog.Handler = slog.NewJSONHandler(os.Stderr, nil)
	if format == "text" {
		h = slog.NewTextHandler(os.Stderr, nil)
	}
	slog.SetDefault(slog.New(h))
}
