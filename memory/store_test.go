package memory

import (
	"context"
	"testing"
)

func TestFileMemoryStoreRejectsTraversalOnPut(t *testing.T) {
	store := NewFileMemoryStore(t.TempDir())
	if err := store.Put(context.Background(), "../ns", "k", "v"); err == nil {
		t.Fatalf("expected traversal error on namespace")
	}
	if err := store.Put(context.Background(), "ns", "../k", "v"); err == nil {
		t.Fatalf("expected traversal error on key")
	}
}

func TestFileMemoryStoreRejectsTraversalOnGetAndList(t *testing.T) {
	store := NewFileMemoryStore(t.TempDir())
	if _, _, err := store.Get(context.Background(), "../ns", "k"); err == nil {
		t.Fatalf("expected traversal error on get namespace")
	}
	if _, _, err := store.Get(context.Background(), "ns", "../k"); err == nil {
		t.Fatalf("expected traversal error on get key")
	}
	if _, err := store.List(context.Background(), "../ns"); err == nil {
		t.Fatalf("expected traversal error on list namespace")
	}
}
