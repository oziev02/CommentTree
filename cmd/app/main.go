package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oziev02/CommentTree/internal/config"
	httphandler "github.com/oziev02/CommentTree/internal/delivery/http"
	"github.com/oziev02/CommentTree/internal/infrastructure/database"
	"github.com/oziev02/CommentTree/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	logger.Info("database connection established")

	repo := database.NewPostgresRepository(pool)
	commentUseCase := usecase.NewCommentUseCase(repo)

	mux := httphandler.NewRouter(commentUseCase)

	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("GET /", fs)
	mux.Handle("GET /index.html", http.RedirectHandler("/", http.StatusMovedPermanently))

	var handler http.Handler = mux
	handler = httphandler.CORSMiddleware(handler)
	handler = httphandler.LoggingMiddleware(logger, handler)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("starting server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
