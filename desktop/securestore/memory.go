package securestore

import "sync"

// MemoryStore is an in-memory SecureStore for tests.
type MemoryStore struct {
	mu sync.RWMutex
	m  map[string]string
}

func NewMemory() *MemoryStore { return &MemoryStore{m: make(map[string]string)} }

func (s *MemoryStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
	return nil
}

func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	return nil
}
