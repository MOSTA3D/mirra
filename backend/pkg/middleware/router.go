package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mirra-ai/mirra/backend/pkg/config"
)

// Handlers groups all route handlers to avoid circular imports.
type Handlers struct {
	Auth    AuthRoutes
	Persona PersonaRoutes
	Jobs    JobRoutes
}

type AuthRoutes interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	Refresh(w http.ResponseWriter, r *http.Request)
	SendVerificationCode(w http.ResponseWriter, r *http.Request)
	VerifyCode(w http.ResponseWriter, r *http.Request)
}

type PersonaRoutes interface {
	List(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
	AddSource(w http.ResponseWriter, r *http.Request)
	Process(w http.ResponseWriter, r *http.Request)
	Export(w http.ResponseWriter, r *http.Request)
}

type JobRoutes interface {
	List(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
}

// NewRouter constructs the full application router with all middleware and routes.
func NewRouter(cfg *config.Config, h Handlers) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware)

	// Health check — no auth required
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"mirra-api"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Auth.Register)
			r.Post("/login", h.Auth.Login)
			r.Post("/refresh", h.Auth.Refresh)
			r.Post("/send-code", h.Auth.SendVerificationCode)
			r.Post("/verify-code", h.Auth.VerifyCode)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(cfg))

			r.Route("/personas", func(r chi.Router) {
				r.Get("/", h.Persona.List)
				r.Post("/", h.Persona.Create)
				r.Get("/{id}", h.Persona.Get)
				r.Put("/{id}", h.Persona.Update)
				r.Delete("/{id}", h.Persona.Delete)
				r.Post("/{id}/sources", h.Persona.AddSource)
				r.Post("/{id}/process", h.Persona.Process)
				r.Post("/{id}/export", h.Persona.Export)
			})

			r.Route("/jobs", func(r chi.Router) {
				r.Get("/", h.Jobs.List)
				r.Get("/{id}", h.Jobs.Get)
			})
		})
	})

	return r
}

// corsMiddleware sets CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
