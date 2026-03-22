// Package verification handles email verification codes.
// In production, codes are sent via email. For now, they are logged to console
// so you can test without an email provider. Swap sendCode() for real SMTP later.
package verification

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"
)

const (
	codeLength  = 6
	codeTTL     = 15 * time.Minute
	maxAttempts = 5
)

type verificationEntry struct {
	Code      string
	Email     string
	ExpiresAt time.Time
	Attempts  int
	Verified  bool
}

// Store holds pending verification codes in memory.
// Replace with Redis in the Postgres phase.
type Store struct {
	mu      sync.Mutex
	entries map[string]*verificationEntry // keyed by email
}

func NewStore() *Store {
	return &Store{entries: make(map[string]*verificationEntry)}
}

// Issue generates and "sends" a verification code for the given email.
// Returns the code (for testing). In production, only send via email.
func (s *Store) Issue(ctx context.Context, email string) (string, error) {
	code, err := generateCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}

	s.mu.Lock()
	s.entries[email] = &verificationEntry{
		Code:      code,
		Email:     email,
		ExpiresAt: time.Now().Add(codeTTL),
		Attempts:  0,
		Verified:  false,
	}
	s.mu.Unlock()

	// Send the code — currently logs to console (swap for real email)
	sendCode(email, code)

	return code, nil
}

// Verify checks the submitted code for the given email.
func (s *Store) Verify(ctx context.Context, email, code string) error {
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

	if entry.Attempts >= maxAttempts {
		return ErrTooManyAttempts
	}

	entry.Attempts++

	if entry.Code != code {
		return ErrInvalidCode
	}

	entry.Verified = true
	delete(s.entries, email) // consume the code
	return nil
}

// IsVerified checks if an email was recently verified (for registration flow).
func (s *Store) IsVerified(email string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[email]
	return ok && entry.Verified
}

func generateCode() (string, error) {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

// sendCode delivers the code. Currently logs — replace with SMTP/SendGrid/etc.
func sendCode(email, code string) {
	log.Printf("📧 VERIFICATION CODE for %s: %s (valid 15 min)", email, code)
	// TODO: integrate email provider
	// Example with SendGrid:
	// sendgrid.Send(email, "Mirra verification", fmt.Sprintf("Your code: %s", code))
}

// Sentinel errors
var (
	ErrNoCode          = &VerificationError{"NO_CODE", "No verification code found for this email"}
	ErrExpired         = &VerificationError{"CODE_EXPIRED", "Verification code has expired"}
	ErrInvalidCode     = &VerificationError{"INVALID_CODE", "Incorrect verification code"}
	ErrTooManyAttempts = &VerificationError{"TOO_MANY_ATTEMPTS", "Too many incorrect attempts. Request a new code."}
)

type VerificationError struct {
	Code    string
	Message string
}

func (e *VerificationError) Error() string { return e.Message }
