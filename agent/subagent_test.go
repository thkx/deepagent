package agent

import (
	"context"
	"testing"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

func TestSlugify(t *testing.T) {
	got := slugify("  Build PLAN v1.0 / Draft  ")
	if got != "build-plan-v1-0-draft" {
		t.Fatalf("unexpected slug: %q", got)
	}
}

func TestNextSubagentResultFileDedup(t *testing.T) {
	backend := fs.NewInMemoryBackend()
	if err := backend.Write(context.Background(), "subagent_task.md", "x"); err != nil {
		t.Fatalf("seed write error: %v", err)
	}

	name, err := nextSubagentResultFile(context.Background(), backend, "task")
	if err != nil {
		t.Fatalf("nextSubagentResultFile error: %v", err)
	}
	if name != "subagent_task_2.md" {
		t.Fatalf("unexpected deduped name: %q", name)
	}
}
