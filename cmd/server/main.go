package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wallet_transfer_service/internal/config"
	"wallet_transfer_service/internal/handler"
	"wallet_transfer_service/internal/middleware"
	"wallet_transfer_service/internal/repository/postgres"
	"wallet_transfer_service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

	cfg := config.Load()
	logger := config.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	var store *postgres.Store
	deps := handler.Deps{}
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s, err := postgres.NewStore(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("database connection failed", slog.Any("error", err))
			if cfg.Env == "production" {
				os.Exit(1)
			}
		} else {
			store = s
			deps.DBPing = func() error {
				return store.Ping(context.Background())
			}
			deps.TransferService = service.NewTransferService(store)
			logger.Info("database connected")
		}
	} else {
		logger.Warn("DATABASE_URL not set; health check is liveness-only")
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.AccessLog(logger))
	router.Use(gin.Recovery())

	handler.RegisterRoutes(router, deps)

	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server listening", slog.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-signalChan
	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	if store != nil {
		store.Close()
	}

	logger.Info("server stopped")
}
