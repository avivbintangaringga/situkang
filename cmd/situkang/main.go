// Command situkang runs the SiTukang API server and serves the embedded frontend.
package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
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
	"golang.org/x/term"

	situkang "situkang"
	"situkang/internal/auth"
	"situkang/internal/clock"
	"situkang/internal/config"
	"situkang/internal/db"
	"situkang/internal/httpx"
	"situkang/internal/worker"
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

	appClock := clock.New(cfg.Location)
	authSvc := auth.NewService(sqlDB, appClock)
	workerSvc := worker.NewService(sqlDB, appClock)

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create-admin":
			return runCreateAdmin(authSvc)
		default:
			return fmt.Errorf("unknown command %q", os.Args[1])
		}
	}

	dist, err := situkang.StaticFS()
	if err != nil {
		return fmt.Errorf("load frontend: %w", err)
	}

	router := newRouter(sqlDB, authSvc, workerSvc, dist)

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

func newRouter(sqlDB *sql.DB, authSvc *auth.Service, workerSvc *worker.Service, dist fs.FS) http.Handler {
	authHandler := auth.NewHandler(authSvc)
	workerHandler := worker.NewHandler(workerSvc)

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

		api.Post("/auth/login", authHandler.Login)

		api.Group(func(protected chi.Router) {
			protected.Use(authSvc.RequireSession)
			protected.Get("/auth/me", authHandler.Me)
			protected.Post("/auth/logout", authHandler.Logout)

			protected.Get("/workers", workerHandler.List)
			protected.Post("/workers", workerHandler.Create)
			protected.Get("/workers/{id}", workerHandler.Get)
			protected.Put("/workers/{id}", workerHandler.Update)
			protected.Delete("/workers/{id}", workerHandler.Delete)
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

// runCreateAdmin prompts for credentials and stores a new admin account.
// Password input is hidden on a terminal and falls back to plain reads when
// stdin is piped.
func runCreateAdmin(svc *auth.Service) error {
	in := bufio.NewReader(os.Stdin)

	fmt.Print("Username: ")
	username, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	password, err := readPassword(in, "Password: ")
	if err != nil {
		return err
	}
	confirmation, err := readPassword(in, "Confirm password: ")
	if err != nil {
		return err
	}
	if password != confirmation {
		return errors.New("passwords do not match")
	}

	user, err := svc.CreateUser(context.Background(), strings.TrimSpace(username), password)
	if err != nil {
		return err
	}

	fmt.Printf("Admin %q created.\n", user.Username)
	return nil
}

func readPassword(in *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		secret, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(secret), nil
	}

	line, err := in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
