// Command situkang runs the SiTukang API server and serves the embedded frontend.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	situkang "situkang"
	"situkang/internal/config"
	"situkang/internal/db"
	"situkang/internal/httpx"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	time.Local = cfg.Location

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create-admin":
			return runCreateAdmin()
		default:
			return fmt.Errorf("unknown command %q", os.Args[1])
		}
	}

	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	migrationFS, err := situkang.Migrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if err := db.Migrate(context.Background(), sqlDB, migrationFS); err != nil {
		return err
	}

	dist, err := situkang.StaticFS()
	if err != nil {
		return fmt.Errorf("load frontend: %w", err)
	}

	router := newRouter(sqlDB, dist)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("SiTukang listening", "address", "http://localhost:"+cfg.Port, "database", cfg.DBPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down server: %w", err)
	}
	return nil
}

func newRouter(sqlDB *sql.DB, dist fs.FS) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			if err := sqlDB.PingContext(r.Context()); err != nil {
				httpx.Error(w, http.StatusServiceUnavailable, "Database tidak dapat dihubungi.")
				return
			}
			httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	})

	spa := situkang.NewSPAHandler(dist)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			httpx.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan.")
			return
		}
		spa.ServeHTTP(w, r)
	})

	return r
}

// runCreateAdmin is filled in with the auth step; until then it must fail loudly
// instead of silently starting the server.
func runCreateAdmin() error {
	return errors.New("create-admin is not implemented yet")
}
