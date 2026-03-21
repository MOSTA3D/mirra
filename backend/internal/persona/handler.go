package persona

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/internal/store"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	appmiddleware "github.com/mirra-ai/mirra/backend/pkg/middleware"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

const disclaimer = "This is an AI-generated approximation. Not affiliated with or endorsed by the real person."

type Handler struct {
	cfg      *config.Config
	personas store.PersonaStore
}

func NewHandler(cfg *config.Config, personas store.PersonaStore) *Handler {
	return &Handler{cfg: cfg, personas: personas}
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

// List returns all personas owned by the authenticated user.
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

// Create creates a new persona.
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
	persona := &store.Persona{
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

	if err := h.personas.Create(r.Context(), persona); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create persona")
		return
	}

	response.JSON(w, http.StatusCreated, persona)
}

// Get returns a single persona by ID (owner only).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	persona, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if persona.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	response.JSON(w, http.StatusOK, persona)
}

// Update updates a persona's metadata.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	persona, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if persona.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	var req updatePersonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) != "" {
		persona.Name = strings.TrimSpace(req.Name)
		persona.Slug = slugify(req.Name)
	}

	if req.Visibility == "private" || req.Visibility == "public" {
		persona.Visibility = req.Visibility
	}

	persona.UpdatedAt = time.Now().UTC()

	if err := h.personas.Update(r.Context(), persona); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update persona")
		return
	}

	response.JSON(w, http.StatusOK, persona)
}

// Delete removes a persona and its sources.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	persona, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if persona.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	if err := h.personas.Delete(r.Context(), id); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete persona")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Persona deleted"})
}

// AddSource attaches a data source to a persona.
func (h *Handler) AddSource(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	personaID := chi.URLParam(r, "id")

	persona, err := h.personas.GetByID(r.Context(), personaID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if persona.OwnerID != userID {
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

	source := &store.Source{
		ID:        uuid.NewString(),
		PersonaID: personaID,
		Type:      req.Type,
		Content:   strings.TrimSpace(req.Content),
		Status:    "pending",
		CreatedAt: time.Now().UTC(),
	}

	if err := h.personas.AddSource(r.Context(), source); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add source")
		return
	}

	response.JSON(w, http.StatusCreated, source)
}

// Export generates a persona export in markdown format.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.GetUserID(r)
	id := chi.URLParam(r, "id")

	persona, err := h.personas.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch persona")
		return
	}

	if persona.OwnerID != userID {
		response.Err(w, http.StatusNotFound, "PERSONA_NOT_FOUND", "Persona not found")
		return
	}

	sources, _ := h.personas.ListSources(r.Context(), id)

	md := buildMarkdownExport(persona, sources)
	response.JSON(w, http.StatusOK, map[string]string{"content": md, "format": "markdown"})
}

func buildMarkdownExport(persona *store.Persona, sources []*store.Source) string {
	md := "# " + persona.Name + " — Persona\n\n"
	md += "> **Disclaimer:** " + disclaimer + "\n\n"
	md += "## Status\n" + persona.Status + "\n\n"
	md += "## Sources\n"
	for _, s := range sources {
		md += "- [" + s.Type + "] " + s.Content + "\n"
	}
	if len(sources) == 0 {
		md += "_No sources added yet._\n"
	}
	md += "\n## Confidence Scores\n"
	for k, v := range persona.Confidence {
		md += "- " + k + ": " + fmt_float(v) + "%\n"
	}
	if len(persona.Confidence) == 0 {
		md += "_Persona not yet processed._\n"
	}
	return md
}

func fmt_float(f float64) string {
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(strings.Replace(
			strings.Replace(fmt_basic(f*100), ".000", "", 1),
			"00", "0", 1), "0 ", " ", 1), "0"), ".")
}

func fmt_basic(f float64) string {
	if f == float64(int(f)) {
		return strings.TrimRight(strings.TrimRight(
			string(rune(int(f)/100+48))+".0", "0"), ".")
	}
	return "~"
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
