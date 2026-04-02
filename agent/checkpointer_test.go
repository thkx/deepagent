package agent

import "testing"

func TestFileCheckpointerRejectsTraversalThreadIDOnSave(t *testing.T) {
	cp := NewFileCheckpointer(t.TempDir())
	err := cp.Save("../evil", State{})
	if err == nil {
		t.Fatalf("expected save error for traversal thread id")
	}
}

func TestFileCheckpointerRejectsTraversalThreadIDOnLoad(t *testing.T) {
	cp := NewFileCheckpointer(t.TempDir())
	_, _, err := cp.Load("../evil")
	if err == nil {
		t.Fatalf("expected load error for traversal thread id")
	}
}
