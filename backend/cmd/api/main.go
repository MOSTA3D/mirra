package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/mirra-ai/mirra/backend/internal/auth"
	"github.com/mirra-ai/mirra/backend/internal/jobs"
	"github.com/mirra-ai/mirra/backend/internal/persona"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/middleware"
)

func main() {
	// Load .env if present (local dev)
	_ = godotenv.Load()

	cfg := config.Load()

	router := buildRouter(cfg)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Mirra API starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func buildRouter(cfg *config.Config) http.Handler {
	handlers := middleware.Handlers{
		Auth:    auth.NewHandler(cfg),
		Persona: persona.NewHandler(cfg),
		Jobs:    jobs.NewHandler(cfg),
	}
	return middleware.NewRouter(cfg, handlers)
}
