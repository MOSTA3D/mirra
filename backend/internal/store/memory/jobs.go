package memory

import (
	"context"
	"sync"

	"github.com/mirra-ai/mirra/backend/internal/store"
)

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*store.Job
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*store.Job),
	}
}

func (s *JobStore) Create(ctx context.Context, job *store.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = job
	return nil
}

func (s *JobStore) GetByID(ctx context.Context, id string) (*store.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return job, nil
}

func (s *JobStore) ListByOwner(ctx context.Context, ownerID string) ([]*store.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*store.Job
	for _, j := range s.jobs {
		if j.OwnerID == ownerID {
			result = append(result, j)
		}
	}
	return result, nil
}

func (s *JobStore) Update(ctx context.Context, job *store.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[job.ID]; !ok {
		return store.ErrNotFound
	}
	s.jobs[job.ID] = job
	return nil
}
