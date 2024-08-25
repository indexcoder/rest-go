package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
	"index-coder/rest-go/internal/config"
	"index-coder/rest-go/internal/http-server/handlers/redirect"
	"index-coder/rest-go/internal/http-server/handlers/url/delete"
	"index-coder/rest-go/internal/http-server/handlers/url/save"
	mwLogger "index-coder/rest-go/internal/http-server/middleware/logger"
	"index-coder/rest-go/internal/lib/logger/handlers/slogpretty"
	"index-coder/rest-go/internal/lib/logger/sl"
	"index-coder/rest-go/internal/storage/sqlite"
	"net/http"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cnf := config.MustLoadConfig()

	log := setupLogger(cnf.Env)

	// log.Info("starting server", slog.String("env", cnf.Env))
	// log.Debug("debug messages are enabled")
	// log.Error("this is error message")

	storage, err := sqlite.New(cnf.StoragePath)
	if err != nil {
		log.Error("Failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cnf.HTTPServer.User: cnf.HTTPServer.Password,
		}))

		r.Post("/", save.New(log, storage))
	})

	router.Get("/{alias}", redirect.New(log, storage))
	router.Delete("/url/{url}", deleteHandler.New(log, storage))

	log.Info("starting server", slog.String("address", cnf.Address))

	srv := &http.Server{
		Addr:         cnf.Address,
		Handler:      router,
		ReadTimeout:  cnf.HTTPServer.Timeout,
		WriteTimeout: cnf.HTTPServer.Timeout,
		IdleTimeout:  cnf.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}
	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {

	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
