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
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/internal/store/memory"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/middleware"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	stores := buildStores(cfg)
	router := buildRouter(cfg, stores)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Mirra API starting on :%s [db=%s]", cfg.Port, cfg.DBDriver)
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

// buildStores wires the correct store implementation based on DB_DRIVER config.
// memory → no setup required, data lost on restart (dev/test)
// postgres → requires DATABASE_URL (prod)
func buildStores(cfg *config.Config) store.Stores {
	switch cfg.DBDriver {
	case "postgres":
		// postgres implementation coming — falls through to memory for now
		log.Println("WARNING: postgres driver not yet implemented, falling back to memory")
		fallthrough
	default:
		log.Println("Using in-memory store")
		return store.Stores{
			Users:    memory.NewUserStore(),
			Personas: memory.NewPersonaStore(),
			Jobs:     memory.NewJobStore(),
		}
	}
}

func buildRouter(cfg *config.Config, stores store.Stores) http.Handler {
	handlers := middleware.Handlers{
		Auth:    auth.NewHandler(cfg, stores.Users),
		Persona: persona.NewHandler(cfg, stores.Personas),
		Jobs:    jobs.NewHandler(cfg, stores.Jobs),
	}
	return middleware.NewRouter(cfg, handlers)
}
