package memory

import (
    "context"
    "sync"
)

// InMemoryRedisStore is a simple in-memory store that implements the Store interface.
// It's intended for local testing and does not provide persistence.
type InMemoryRedisStore struct {
    mu   sync.RWMutex
    data map[string]map[string]any // namespace -> key -> value
}

// NewInMemoryRedisStore creates an initialized in-memory redis-like store.
func NewInMemoryRedisStore() Store {
    return &InMemoryRedisStore{data: make(map[string]map[string]any)}
}

func (s *InMemoryRedisStore) Get(ctx context.Context, namespace, key string) (any, bool, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    ns, ok := s.data[namespace]
    if !ok {
        return nil, false, nil
    }
    v, ok := ns[key]
    if !ok {
        return nil, false, nil
    }
    return v, true, nil
}

func (s *InMemoryRedisStore) Put(ctx context.Context, namespace, key string, value any) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    ns, ok := s.data[namespace]
    if !ok {
        ns = make(map[string]any)
        s.data[namespace] = ns
    }
    ns[key] = value
    return nil
}

func (s *InMemoryRedisStore) List(ctx context.Context, namespace string) ([]string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    ns, ok := s.data[namespace]
    if !ok {
        return []string{}, nil
    }
    keys := make([]string, 0, len(ns))
    for k := range ns {
        keys = append(keys, k)
    }
    return keys, nil
}
