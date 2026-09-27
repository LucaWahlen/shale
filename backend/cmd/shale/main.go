package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "time/tzdata"

	"shale/internal/config"
	"shale/internal/db"
	"shale/internal/httpapi"
	"shale/internal/repository/sqlite"
	"shale/internal/service"
	"shale/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("shale exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		return err
	}

	store := sqlite.New(conn)
	scheduleRepo := sqlite.NewScheduleRepo(store)
	eventRepo := sqlite.NewEventRepo(store)
	attendeeRepo := sqlite.NewAttendeeRepo(store)
	settingsRepo := sqlite.NewSettingsRepo(store)
	auditRepo := sqlite.NewAuditRepo(store)

	clock := func() time.Time { return time.Now() }

	auditSvc := service.NewAuditService(auditRepo, settingsRepo)
	auditSvc.Prune(ctx)
	scheduleSvc := service.NewScheduleService(scheduleRepo, eventRepo, attendeeRepo, auditSvc, clock)
	eventSvc := service.NewEventService(scheduleRepo, eventRepo, auditSvc)
	attendeeSvc := service.NewAttendeeService(scheduleRepo, eventRepo, attendeeRepo, auditSvc, clock)
	settingsSvc := service.NewSettingsService(settingsRepo)
	transferSvc := service.NewTransferService(scheduleRepo, eventRepo, attendeeRepo, settingsSvc, store, clock)

	api := httpapi.New(httpapi.Deps{
		Schedules: scheduleSvc,
		Events:    eventSvc,
		Attendees: attendeeSvc,
		Settings:  settingsSvc,
		Transfer:  transferSvc,
		Auth:      service.NewAuthService(cfg.AdminPassword, clock),
		Audit:     auditSvc,
	}, logger)

	spa, err := web.Handler()
	if err != nil {
		return err
	}
	root := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			api.Handler().ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	logger.Info("shale listening", "addr", cfg.Addr, "db_path", cfg.DBPath)

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
