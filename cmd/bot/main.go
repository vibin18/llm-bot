package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vibin/whatsapp-llm-bot/internal/adapters/primary/http"
	"github.com/vibin/whatsapp-llm-bot/internal/adapters/primary/whatsapp"
	"github.com/vibin/whatsapp-llm-bot/internal/adapters/secondary/llm"
	"github.com/vibin/whatsapp-llm-bot/internal/adapters/secondary/storage"
	"github.com/vibin/whatsapp-llm-bot/internal/adapters/secondary/webhook"
	"github.com/vibin/whatsapp-llm-bot/internal/config"
	"github.com/vibin/whatsapp-llm-bot/internal/core/services"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func main() {
	ctx := context.Background()

	// Load configuration
	configPath := getEnv("CONFIG_PATH", "config.yaml")
	configStore := config.NewFileConfigStore(configPath)

	cfg, err := configStore.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(cfg.App.LogLevel)
	logger.Info("Starting WhatsApp LLM Bot", "version", "1.0.0")

	// Create WhatsApp logger adapter
	waLogger := &whatsmeowLogger{logger: logger}

	// Initialize components
	messageRepo := storage.NewMemoryRepository()

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Ollama.Timeout)
	if err != nil {
		logger.Error("Invalid timeout format, using default 30s", "error", err)
		timeout = 30 * time.Second
	}

	llmProvider, err := llm.NewOllamaProvider(
		cfg.Ollama.URL,
		cfg.Ollama.Model,
		cfg.Ollama.Temperature,
		timeout,
	)
	if err != nil {
		logger.Error("Failed to create LLM provider", "error", err)
		os.Exit(1)
	}

	// Check LLM availability
	if !llmProvider.IsAvailable(ctx) {
		logger.Warn("LLM service is not available, but continuing anyway")
	}

	// Initialize group manager
	groupMgr := services.NewGroupService(configStore)
	if err := groupMgr.SyncWithConfig(); err != nil {
		logger.Error("Failed to sync group manager", "error", err)
		os.Exit(1)
	}

	// Initialize WhatsApp client
	waClient, err := whatsapp.NewClient(
		cfg.WhatsApp.SessionPath,
		cfg.WhatsApp.AllowedGroups,
		waLogger,
	)
	if err != nil {
		logger.Error("Failed to create WhatsApp client", "error", err)
		os.Exit(1)
	}

	// Initialize webhook client
	webhookClient := webhook.NewClient(30 * time.Second)

	// Initialize chat service
	chatService := services.NewChatService(
		llmProvider,
		messageRepo,
		waClient,
		groupMgr,
		webhookClient,
		cfg.WhatsApp.TriggerWords,
		cfg.Webhooks,
		logger,
	)

	// Start WhatsApp client
	logger.Info("Starting WhatsApp client")
	if err := waClient.Start(ctx); err != nil {
		logger.Error("Failed to start WhatsApp client", "error", err)
		os.Exit(1)
	}

	// Start chat service
	if err := chatService.Start(ctx); err != nil {
		logger.Error("Failed to start chat service", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP server
	httpHandlers := http.NewHandlers(waClient, groupMgr, configStore, logger)
	httpServer := http.NewServer(cfg.App.Port, httpHandlers, logger)

	if err := httpServer.Start(ctx); err != nil {
		logger.Error("Failed to start HTTP server", "error", err)
		os.Exit(1)
	}

	logger.Info("Bot is running", "admin_url", fmt.Sprintf("http://localhost:%d", cfg.App.Port))

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down gracefully...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping HTTP server", "error", err)
	}

	if err := waClient.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping WhatsApp client", "error", err)
	}

	logger.Info("Shutdown complete")
}

// setupLogger creates and configures the logger
func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// whatsmeowLogger adapts slog.Logger to whatsmeow's logger interface
type whatsmeowLogger struct {
	logger *slog.Logger
}

func (l *whatsmeowLogger) Errorf(msg string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(msg, args...))
}

func (l *whatsmeowLogger) Warnf(msg string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(msg, args...))
}

func (l *whatsmeowLogger) Infof(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, args...))
}

func (l *whatsmeowLogger) Debugf(msg string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(msg, args...))
}

func (l *whatsmeowLogger) Sub(module string) waLog.Logger {
	return &whatsmeowLogger{
		logger: l.logger.With("module", module),
	}
}
