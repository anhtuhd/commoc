package main

import (
	"context"
	"github.com/anhtuhd/commoc/backend/internal/shop"
	"github.com/anhtuhd/commoc/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, e := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if e != nil {
		panic("Invalid database configuration")
	}
	cfg.MaxConns = 8
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "15000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "10000"
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		panic("Database unavailable")
	}
	defer pool.Close()
	if _, e = pool.Exec(ctx, migrations.SQL); e != nil {
		slog.Error("migration_failed", "error", e)
		os.Exit(1)
	}
	a := &shop.App{DB: &shop.Store{Pool: pool}, Site: env("SITE_URL", "https://demo.pishare.cloud"), AdminUser: os.Getenv("ADMIN_USER"), AdminEmail: os.Getenv("ADMIN_EMAIL"), PasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"), SessionSecret: os.Getenv("SESSION_SECRET"), SePaySecret: os.Getenv("SEPAY_WEBHOOK_SECRET"), Mode: env("SEPAY_MODE", "test"), AutoMatch: os.Getenv("SEPAY_AUTO_MATCH") == "true", BridgeURL: os.Getenv("ZALO_BRIDGE_URL"), BridgeSecret: os.Getenv("ZALO_BRIDGE_SECRET"), MediaDir: env("MEDIA_DIR", "/data/media")}
	if len(a.SessionSecret) < 32 || a.AdminUser == "" || a.PasswordHash == "" {
		panic("Admin/session environment required")
	}
	if a.Mode != "live" && a.Mode != "test" {
		panic("Invalid SePay mode")
	}
	if a.Mode == "test" && a.Site == "https://pishare.cloud" {
		panic("Test mode cannot target production")
	}
	if len(os.Args) == 3 && os.Args[1] == "import" {
		if e := a.ImportFile(ctx, os.Args[2]); e != nil {
			slog.Error("import_failed", "error", e)
			os.Exit(1)
		}
		slog.Info("import_complete")
		return
	}
	if e = a.Initialize(ctx); e != nil {
		panic(e)
	}
	server := &http.Server{Addr: env("LISTEN_ADDR", ":8080"), Handler: a.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tickCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
				if e := a.Tick(tickCtx); e != nil {
					slog.Warn("worker_tick_failed")
				}
				cancel()
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("listening", "address", server.Addr)
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		panic(e)
	}
}
