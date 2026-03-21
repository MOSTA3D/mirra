package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mirra-ai/mirra/backend/pkg/config"
	"github.com/mirra-ai/mirra/backend/pkg/response"
)

// Handler handles authentication endpoints.
type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// Register creates a new user account.
// TODO: persist to database
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required")
		return
	}

	// Stub: generate a user ID and return tokens
	// Real implementation will hash password, persist user, check uniqueness
	userID := uuid.NewString()
	tokens, err := h.generateTokens(userID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate tokens")
		return
	}

	response.JSON(w, http.StatusCreated, tokens)
}

// Login authenticates an existing user.
// TODO: validate against database
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required")
		return
	}

	// Stub: return tokens for any valid-looking request
	// Real implementation will look up user, verify password hash
	userID := uuid.NewString()
	tokens, err := h.generateTokens(userID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate tokens")
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

// Refresh issues a new access token from a valid refresh token.
// TODO: validate refresh token from store
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	response.Err(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Refresh endpoint coming soon")
}

func (h *Handler) generateTokens(userID string) (*tokenResponse, error) {
	expiresIn := 3600 // 1 hour

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &tokenResponse{
		AccessToken:  signed,
		RefreshToken: uuid.NewString(), // Stub — real refresh tokens need a store
		ExpiresIn:    expiresIn,
	}, nil
}
