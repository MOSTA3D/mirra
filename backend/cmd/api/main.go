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
	"github.com/mirra-ai/mirra/backend/internal/llm"
	"github.com/mirra-ai/mirra/backend/internal/persona"
	"github.com/mirra-ai/mirra/backend/internal/pipeline"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/internal/store/memory"
	"github.com/mirra-ai/mirra/backend/internal/verification"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/middleware"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	stores := buildStores(cfg)
	llmProvider := buildLLM(cfg)
	runner := pipeline.NewRunner(stores, llmProvider)
	vs := verification.NewStore()
	router := buildRouter(cfg, stores, runner, vs)

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

func buildStores(cfg *config.Config) store.Stores {
	switch cfg.DBDriver {
	case "postgres":
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

func buildLLM(cfg *config.Config) llm.Provider {
	if cfg.GroqAPIKey != "" {
		log.Printf("LLM: Groq provider active (llama-3.1-8b-instant)")
		return llm.NewGroqProvider(cfg.GroqAPIKey)
	}
	log.Printf("LLM: no provider configured — using rule-based distillation")
	return nil
}

func buildRouter(cfg *config.Config, stores store.Stores, runner *pipeline.Runner, vs *verification.Store) http.Handler {
	handlers := middleware.Handlers{
		Auth:    auth.NewHandler(cfg, stores.Users, vs),
		Persona: persona.NewHandler(cfg, stores.Personas, runner),
		Jobs:    jobs.NewHandler(cfg, stores.Jobs),
	}
	return middleware.NewRouter(cfg, handlers)
}
