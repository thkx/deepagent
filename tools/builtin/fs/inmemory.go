package fs

import (
	"context"
	"os"
	"sync"
)

type InMemoryBackend struct {
	files map[string]string
	mu    sync.RWMutex
}

func NewInMemoryBackend() Backend {
	return &InMemoryBackend{files: make(map[string]string)}
}

func (b *InMemoryBackend) List(ctx context.Context, path string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var list []string
	for k := range b.files {
		list = append(list, k)
	}
	return list, nil
}

func (b *InMemoryBackend) Read(ctx context.Context, path string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if c, ok := b.files[path]; ok {
		return c, nil
	}
	return "", os.ErrNotExist
}

func (b *InMemoryBackend) Write(ctx context.Context, path, content string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.files[path] = content
	return nil
}

func (b *InMemoryBackend) Edit(ctx context.Context, path, instructions string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c, ok := b.files[path]; ok {
		b.files[path] = c + "\n// Edited: " + instructions
	} else {
		b.files[path] = "// New file: " + instructions
	}
	return nil
}
