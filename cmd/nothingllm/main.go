package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nothingllm/internal/audit"
	"nothingllm/internal/api"
	"nothingllm/internal/auth"
	"nothingllm/internal/config"
	"nothingllm/internal/db"
	"nothingllm/internal/openai"
	"nothingllm/internal/qdrant"
	"nothingllm/internal/rag"
)

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		logger.Error("POSTGRES_URL environment variable is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Open(ctx, dsn)
	if err != nil {
		logger.Error("failed to connect database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	setupComplete, err := database.IsSetupComplete(ctx)
	if err != nil {
		logger.Error("failed to check setup status", "err", err)
		os.Exit(1)
	}

	if !setupComplete {
		runSetupMode(logger, database)
		return
	}

	cfg, err := loadConfigFromDB(ctx, database, dsn)
	if err != nil {
		logger.Error("failed to load config from database", "err", err)
		os.Exit(1)
	}

	runFullMode(logger, ctx, database, cfg)
}

func loadConfigFromDB(ctx context.Context, database *db.DB, dsn string) (*config.Config, error) {
	settings, err := database.LoadConfigFromDB(ctx)
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{
		Server:   config.ServerConfig{Addr: ":8080"},
		Database: config.DatabaseConfig{DSN: dsn},
		Qdrant: config.QdrantConfig{
			URL: settings["qdrant_url"], APIKey: settings["qdrant_api_key"], SyncInterval: 60 * time.Second,
		},
		OpenAI: config.OpenAIConfig{
			APIKey: settings["openai_api_key"], EmbeddingModel: "text-embedding-3-small",
		},
		Auth:      config.AuthConfig{SessionTTL: 24 * time.Hour, BcryptCost: 12},
		RateLimit: config.RateLimitConfig{RequestsPerMinute: 100, LoginPerMinute: 5},
	}
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		cfg.Server.Addr = addr
	}
	return cfg, cfg.Validate()
}

func runSetupMode(logger *slog.Logger, database *db.DB) {
	setupSrv := api.NewSetupServer(database)
	mux := http.NewServeMux()
	mux.Handle("/api/setup/", setupSrv.Handler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","mode":"setup"}`))
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version": Version, "git_commit": GitCommit, "build_date": BuildDate,
		})
	})
	httpSrv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		logger.Info("server started in setup mode", "addr", ":8080")
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down setup server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpSrv.Shutdown(shutdownCtx)
	logger.Info("stopped")
}

func runFullMode(logger *slog.Logger, ctx context.Context, database *db.DB, cfg *config.Config) {
	qdrantClient := qdrant.NewClient(cfg.Qdrant.URL, cfg.Qdrant.APIKey)
	openaiClient := openai.NewClient(cfg.OpenAI.APIKey, "")

	authSvc := auth.NewService(database, cfg.Auth.SessionTTL, cfg.Auth.BcryptCost)
	pipeline := rag.NewPipeline(qdrantClient, openaiClient, database)
	syncEngine := qdrant.NewSyncEngine(qdrantClient, database, cfg.Qdrant.SyncInterval)
	modelSyncer := openai.NewModelSyncer(openaiClient, database)

	go syncEngine.Run(ctx)
	go func() {
		if err := modelSyncer.Sync(ctx); err != nil {
			logger.Error("initial model sync", "err", err)
		}
	}()

	auditLogger := audit.New(database.Pool())
	go auditLogger.Run(ctx)

	srv := api.NewServer(authSvc, database, pipeline, auditLogger)
	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		if err := database.Pool().Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"not ready","reason":"db"}`, http.StatusServiceUnavailable)
			return
		}
		if err := qdrantClient.Ping(r.Context()); err != nil {
			http.Error(w, `{"status":"not ready","reason":"qdrant"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		c := r.Context()
		var users, workspaces, collections, activeSessions int
		database.Pool().QueryRow(c, `SELECT COUNT(*) FROM users`).Scan(&users)
		database.Pool().QueryRow(c, `SELECT COUNT(*) FROM workspaces`).Scan(&workspaces)
		database.Pool().QueryRow(c, `SELECT COUNT(*) FROM collections`).Scan(&collections)
		database.Pool().QueryRow(c, `SELECT COUNT(*) FROM sessions WHERE expires_at > now()`).Scan(&activeSessions)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"users": users, "workspaces": workspaces,
			"collections": collections, "active_sessions": activeSessions,
		})
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version": Version, "git_commit": GitCommit, "build_date": BuildDate,
		})
	})

	httpSrv := &http.Server{Addr: cfg.Server.Addr, Handler: mux}
	go func() {
		logger.Info("server started", "addr", cfg.Server.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}
	logger.Info("stopped")
}