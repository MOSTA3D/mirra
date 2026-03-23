package verification

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const resetTTL = 15 * time.Minute

type resetEntry struct {
	Code      string
	Email     string
	ExpiresAt time.Time
	Attempts  int
	Used      bool
}

// ResetStore holds pending password reset codes.
type ResetStore struct {
	mu      sync.Mutex
	entries map[string]*resetEntry // keyed by email
}

func NewResetStore() *ResetStore {
	return &ResetStore{entries: make(map[string]*resetEntry)}
}

// Issue generates a reset code and "sends" it to the email.
func (s *ResetStore) Issue(ctx context.Context, email string) error {
	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("failed to generate reset code: %w", err)
	}

	s.mu.Lock()
	s.entries[email] = &resetEntry{
		Code:      code,
		Email:     email,
		ExpiresAt: time.Now().Add(resetTTL),
	}
	s.mu.Unlock()

	// Log to console — replace with real email provider
	log.Printf("🔑 PASSWORD RESET CODE for %s: %s (valid 15 min)", email, code)
	return nil
}

// Verify checks the code without consuming it (call Consume after password update).
func (s *ResetStore) Verify(ctx context.Context, email, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[email]
	if !ok {
		return ErrNoCode
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(s.entries, email)
		return ErrExpired
	}
	if entry.Used {
		return ErrNoCode
	}
	if entry.Attempts >= maxAttempts {
		return ErrTooManyAttempts
	}

	entry.Attempts++
	if entry.Code != code {
		return ErrInvalidCode
	}

	return nil
}

// Consume marks the code as used — call after successfully updating password.
func (s *ResetStore) Consume(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, email)
}
