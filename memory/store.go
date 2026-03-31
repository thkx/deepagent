package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store interface {
	Get(ctx context.Context, namespace, key string) (any, bool, error)
	Put(ctx context.Context, namespace, key string, value any) error
	List(ctx context.Context, namespace string) ([]string, error)
}

type FileMemoryStore struct {
	Dir string
	mu  sync.RWMutex
}

func NewFileMemoryStore(dir string) Store {
	if dir == "" {
		dir = "./.memory"
	}
	_ = os.MkdirAll(dir, 0755)
	return &FileMemoryStore{Dir: dir}
}

func (s *FileMemoryStore) Get(ctx context.Context, namespace, key string) (any, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	path := filepath.Join(s.Dir, namespace, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (s *FileMemoryStore) Put(ctx context.Context, namespace, key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.Dir, namespace)
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, key+".json")
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *FileMemoryStore) List(ctx context.Context, namespace string) ([]string, error) {
	dir := filepath.Join(s.Dir, namespace)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var keys []string
	for _, e := range entries {
		if !e.IsDir() {
			keys = append(keys, e.Name())
		}
	}
	return keys, nil
}
