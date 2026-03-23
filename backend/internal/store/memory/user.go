package memory

import (
	"context"
	"sync"

	"github.com/mirra-ai/mirra/backend/internal/store"
)

type UserStore struct {
	mu      sync.RWMutex
	byID    map[string]*store.User
	byEmail map[string]*store.User
}

func NewUserStore() *UserStore {
	return &UserStore{
		byID:    make(map[string]*store.User),
		byEmail: make(map[string]*store.User),
	}
}

func (s *UserStore) Create(ctx context.Context, user *store.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.byEmail[user.Email]; exists {
		return store.ErrAlreadyExists
	}

	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*store.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*store.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byEmail[email]
	if !ok {
		return nil, store.ErrNotFound
	}
	return user, nil
}

func (s *UserStore) Update(ctx context.Context, user *store.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byID[user.ID]; !ok {
		return store.ErrNotFound
	}
	s.byID[user.ID] = user
	s.byEmail[user.Email] = user
	return nil
}
