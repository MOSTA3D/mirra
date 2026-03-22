package persona

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/pipeline"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	appmiddleware "github.com/mirra-ai/mirra/backend/pkg/middleware"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

const disclaimer = "This is an AI-generated approximation. Not affiliated with or endorsed by the real person."

type Handler struct {
	cfg      *config.Config
	personas store.PersonaStore
	runner   *pipeline.Runner
}

func NewHandler(cfg *config.Config, personas store.PersonaStore, runner *pipeline.Runner) *Handler {
	return &Handler{cfg: cfg, personas: personas, runner: runner}
}

type createPersonaRequest struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type addSourceRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type updatePersonaRequest struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type processRequest struct{}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	personas, err := h.personas.ListByOwner(r.Context(), userID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch personas")
		return
	}
	if personas == nil {
		personas = []*store.Persona{}
	}
	response.JSON(w, http.StatusOK, personas)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)

	var req createPersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")
		return
	}

	if req.Visibility == "" {
		req.Visibility = "private"
	}

	if req.Visibility != "private" && req.Visibility != "public" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Visibility must be private or public")
		return
	}

	now := time.Now().UTC()
	p := &store.Persona{
		ID:         uuid.NewString(),
		OwnerID:    userID,
		Name:       strings.TrimSpace(req.Name),
		Slug:       slugify(req.Name),
		Visibility: req.Visibility,
		Status:     "draft",
		Disclaimer: disclaimer,
		Confidence: map[string]float64{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.personas.Create(r.Context(), p); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create persona")
		return
	}

	response.JSON(w, http.StatusCreated, p)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	response.JSON(w, http.StatusOK, p)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	var req updatePersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) != "" {
		p.Name = strings.TrimSpace(req.Name)
		p.Slug = slugify(req.Name)
	}
	if req.Visibility == "private" || req.Visibility == "public" {
		p.Visibility = req.Visibility
	}
	p.UpdatedAt = time.Now().UTC()

	if err := h.personas.Update(r.Context(), p); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update persona")
		return
	}

	response.JSON(w, http.StatusOK, p)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	if err := h.personas.Delete(r.Context(), id); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete persona")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Persona deleted"})
}

func (h *Handler) AddSource(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	personaID := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), personaID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	var req addSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	validTypes := map[string]bool{"url": true, "pdf": true, "text": true}
	if !validTypes[req.Type] {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Type must be one of: url, pdf, text")
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Content is required")
		return
	}

	src := &store.Source{
		ID:        uuid.NewString(),
		PersonaID: personaID,
		Type:      req.Type,
		Content:   strings.TrimSpace(req.Content),
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	if err := h.personas.AddSource(r.Context(), src); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add source")
		return
	}

	response.JSON(w, http.StatusCreated, src)
}

// Process triggers the distillation pipeline for a persona.
// It runs asynchronously and immediately returns the updated persona status.
func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	personaID := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), personaID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	if p.Status == "processing" {
		response.Err(w, http.StatusConflict, "ALREADY_PROCESSING", "Persona is already being processed")
		return
	}

	sources, err := h.personas.ListSources(r.Context(), personaID)
	if err != nil || len(sources) == 0 {
		response.Err(w, http.StatusBadRequest, "NO_SOURCES", "Add at least one source before processing")
		return
	}

	// Launch pipeline async
	go h.runner.Run(r.Context(), personaID, p.Name, sources)

	response.JSON(w, http.StatusAccepted, map[string]string{
		"message": "Processing started",
		"status":  "processing",
	})
}

// Export generates the markdown export for a ready persona.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	p, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if p.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	sources, _ := h.personas.ListSources(r.Context(), id)

	// If persona is ready, re-run export from stored data
	// If not ready, return basic export
	if p.Status == "ready" && len(sources) > 0 {
		ingestor := pipeline.NewIngestor()
		extractor := pipeline.NewExtractor()
		distiller := pipeline.NewDistiller()
		exporter := pipeline.NewExporter()

		var chunks []*pipeline.Chunk
		for _, src := range sources {
			chunks = append(chunks, ingestor.Ingest(src.Type, src.Content))
		}
		signals := extractor.Extract(chunks)
		profile := distiller.Distill(p.Name, signals)
		md := exporter.ToMarkdown(profile)

		response.JSON(w, http.StatusOK, map[string]string{"content": md, "format": "markdown"})
		return
	}

	// Basic export for draft personas
	md := buildBasicExport(p, sources)
	response.JSON(w, http.StatusOK, map[string]string{"content": md, "format": "markdown"})
}

func buildBasicExport(p *store.Persona, sources []*store.Source) string {
	var b strings.Builder
	b.WriteString("# " + p.Name + " — Persona\n\n")
	b.WriteString("> **Disclaimer:** " + disclaimer + "\n\n")
	b.WriteString("**Status:** " + p.Status + " — process the persona to get the full export.\n\n")
	b.WriteString("## Sources\n")
	for _, s := range sources {
		b.WriteString("- [" + s.Type + "] " + s.Content + "\n")
	}
	return b.String()
}

func slugify(name string) string {
	var slug strings.Builder
	for _, c := range strings.ToLower(name) {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			slug.WriteRune(c)
		case c == ' ' || c == '-' || c == '_':
			slug.WriteRune('-')
		}
	}
	return strings.Trim(slug.String(), "-")
}
