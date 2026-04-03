package fs

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestInMemoryReadMissingFileReturnsNotExist(t *testing.T) {
	b := NewInMemoryBackend()
	_, err := b.Read(context.Background(), "missing.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got: %v", err)
	}
}
