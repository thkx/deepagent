package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Checkpointer interface {
	Save(threadID string, state State) error
	Load(threadID string) (State, bool, error)
}

// FileCheckpointer：生产级文件持久化（简单、可替换为 Redis/Postgres）
type FileCheckpointer struct {
	Dir string
	mu  sync.Mutex
}

func NewFileCheckpointer(dir string) Checkpointer {
	if dir == "" {
		dir = "./.checkpoints"
	}
	os.MkdirAll(dir, 0755)
	return &FileCheckpointer{Dir: dir}
}

func (c *FileCheckpointer) Save(threadID string, state State) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := validatePathComponent(threadID, "thread_id"); err != nil {
		return err
	}

	path := filepath.Join(c.Dir, threadID+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *FileCheckpointer) Load(threadID string) (State, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := validatePathComponent(threadID, "thread_id"); err != nil {
		return State{}, false, err
	}

	path := filepath.Join(c.Dir, threadID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, false, nil
		}
		return State{}, false, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, false, err
	}
	return state, true, nil
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
