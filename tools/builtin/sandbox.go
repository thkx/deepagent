package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

// ExecuteConfig holds configuration for sandbox command execution
type ExecuteConfig struct {
	AllowedCommands map[string]bool
	Timeout         time.Duration
	MaxOutputSize   int64
}

// DefaultExecuteConfig returns the default safe configuration
func DefaultExecuteConfig() ExecuteConfig {
	return ExecuteConfig{
		AllowedCommands: map[string]bool{
			"ls":      true,
			"cat":     true,
			"echo":    true,
			"pwd":     true,
			"whoami":  true,
			"date":    true,
			"grep":    true,
			"head":    true,
			"tail":    true,
			"wc":      true,
			"sort":    true,
			"uniq":    true,
			"cut":     true,
			"tr":      true,
			"sed":     true,
			"awk":     true,
			"find":    true,
			"test":    true,
			"stat":    true,
			"file":    true,
			"which":   true,
			"dirname": true,
			"basename": true,
		},
		Timeout:       30 * time.Second,
		MaxOutputSize: 1024 * 1024, // 1MB
	}
}

type SandboxBackend struct {
	inner  fs.Backend
	config ExecuteConfig
}

func NewSandboxBackend(inner fs.Backend) fs.Backend {
	if inner == nil {
		inner = fs.NewInMemoryBackend()
	}
	return &SandboxBackend{
		inner:  inner,
		config: DefaultExecuteConfig(),
	}
}

// NewSandboxBackendWithConfig creates a SandboxBackend with custom configuration
func NewSandboxBackendWithConfig(inner fs.Backend, config ExecuteConfig) fs.Backend {
	if inner == nil {
		inner = fs.NewInMemoryBackend()
	}
	return &SandboxBackend{
		inner:  inner,
		config: config,
	}
}

// === 必须实现 fs.Backend 接口的所有方法 ===

func (s *SandboxBackend) List(ctx context.Context, path string) ([]string, error) {
	return s.inner.List(ctx, path)
}

func (s *SandboxBackend) Read(ctx context.Context, path string) (string, error) {
	return s.inner.Read(ctx, path)
}

func (s *SandboxBackend) Write(ctx context.Context, path, content string) error {
	return s.inner.Write(ctx, path, content)
}

func (s *SandboxBackend) Edit(ctx context.Context, path, instructions string) error {
	return s.inner.Edit(ctx, path, instructions)
}

// === Sandbox 特有方法（供 execute 工具使用）===

func (s *SandboxBackend) Execute(ctx context.Context, command string) (string, error) {
	// 生产环境建议使用 Docker 隔离，这里提供基础安全版本
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	if !s.config.AllowedCommands[parts[0]] {
		return "", fmt.Errorf("command not allowed in sandbox: %s", parts[0])
	}

	// Apply timeout if configured
	execCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(execCtx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()

	// Truncate output if it exceeds MaxOutputSize
	if s.config.MaxOutputSize > 0 && int64(len(output)) > s.config.MaxOutputSize {
		output = output[:s.config.MaxOutputSize]
	}

	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

