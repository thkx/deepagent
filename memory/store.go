package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store interface {
	Get(ctx context.Context, namespace, key string) (any, bool, error)
	Put(ctx context.Context, namespace, key string, value any) error
	List(ctx context.Context, namespace string) ([]string, error)
}

// CloserStore is an optional extension for stores that hold external resources.
type CloserStore interface {
	Store
	Close() error
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
	if err := validatePathComponent(namespace, "namespace"); err != nil {
		return nil, false, err
	}
	if err := validatePathComponent(key, "key"); err != nil {
		return nil, false, err
	}
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
	if err := validatePathComponent(namespace, "namespace"); err != nil {
		return err
	}
	if err := validatePathComponent(key, "key"); err != nil {
		return err
	}
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
	if err := validatePathComponent(namespace, "namespace"); err != nil {
		return nil, err
	}
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

func (s *FileMemoryStore) Close() error {
	return nil
}

func validatePathComponent(v, field string) error {
	if v == "" {
		return nil
	}
	if filepath.IsAbs(v) {
		return fmt.Errorf("%s must not be an absolute path", field)
	}
	if strings.ContainsAny(v, `/\`) || strings.Contains(v, "..") {
		return fmt.Errorf("unsafe %s: %q", field, v)
	}
	clean := filepath.Clean(v)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("unsafe %s: %q", field, v)
	}
	return nil
}
