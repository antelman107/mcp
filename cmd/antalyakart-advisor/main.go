package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/antelman107/mcp/internal/advisor"
	"github.com/antelman107/mcp/internal/antalyakart"
)

func main() {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	apiKey := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY is required")
	}

	historyPath, err := advisor.DefaultHistoryPath()
	if err != nil {
		log.Fatalf("history path: %v", err)
	}
	store, err := advisor.Open(historyPath)
	if err != nil {
		log.Fatalf("load history: %v", err)
	}

	transit := antalyakart.NewClient(
		envOrDefault("ANTALYAKART_BASE_URL", antalyakart.DefaultBaseURL),
		envOrDefault("ANTALYAKART_REGION", antalyakart.DefaultRegion),
		envOrDefault("ANTALYAKART_LANG", antalyakart.DefaultLang),
		envOrDefault("ANTALYAKART_AUTH_TYPE", antalyakart.DefaultAuthType),
	)
	agent, err := advisor.NewAgent(context.Background(), apiKey, os.Getenv("GEMINI_MODEL"), transit)
	if err != nil {
		log.Fatalf("agent: %s", redact(err.Error(), token, apiKey))
	}

	webhookSecret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
	telegram := advisor.NewTelegramClient(token, webhookSecret)
	bot := advisor.NewBot(
		store,
		agent,
		telegram,
		envOrDefault("BOT_PATH", "/antalyakart-advisor"),
		webhookSecret,
		[]string{token, apiKey, webhookSecret},
	)

	addr := envOrDefault("BOT_ADDR", "127.0.0.1:8091")
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           bot.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	if webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_URL")); webhookURL != "" {
		if err := telegram.SetWebhook(context.Background(), webhookURL); err != nil {
			log.Fatalf("setWebhook: %s", redact(err.Error(), token, apiKey))
		}
		log.Printf("telegram webhook registered")
	}

	log.Printf("antalyakart advisor listening on %s, history %s", addr, historyPath)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func redact(message, token, apiKey string) string {
	if token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
	}
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[REDACTED]")
	}
	return message
}
