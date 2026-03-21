package memory

import (
	"context"
	"sync"

	"github.com/mirra-ai/mirra/backend/internal/store"
)

type PersonaStore struct {
	mu      sync.RWMutex
	personas map[string]*store.Persona
	sources  map[string][]*store.Source // keyed by personaID
}

func NewPersonaStore() *PersonaStore {
	return &PersonaStore{
		personas: make(map[string]*store.Persona),
		sources:  make(map[string][]*store.Source),
	}
}

func (s *PersonaStore) Create(ctx context.Context, persona *store.Persona) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.personas[persona.ID] = persona
	return nil
}

func (s *PersonaStore) GetByID(ctx context.Context, id string) (*store.Persona, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.personas[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return p, nil
}

func (s *PersonaStore) ListByOwner(ctx context.Context, ownerID string) ([]*store.Persona, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*store.Persona
	for _, p := range s.personas {
		if p.OwnerID == ownerID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (s *PersonaStore) Update(ctx context.Context, persona *store.Persona) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.personas[persona.ID]; !ok {
		return store.ErrNotFound
	}
	s.personas[persona.ID] = persona
	return nil
}

func (s *PersonaStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.personas[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.personas, id)
	delete(s.sources, id)
	return nil
}

func (s *PersonaStore) AddSource(ctx context.Context, source *store.Source) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.personas[source.PersonaID]; !ok {
		return store.ErrNotFound
	}
	s.sources[source.PersonaID] = append(s.sources[source.PersonaID], source)
	return nil
}

func (s *PersonaStore) ListSources(ctx context.Context, personaID string) ([]*store.Source, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.sources[personaID], nil
}
