package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

type SandboxBackend struct {
	inner fs.Backend // 委托给实际的后端（InMemory 或 Disk）
}

func NewSandboxBackend(inner fs.Backend) fs.Backend {
	if inner == nil {
		inner = fs.NewInMemoryBackend()
	}
	return &SandboxBackend{inner: inner}
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
	allowedCommands := map[string]bool{
		"ls": true, "cat": true, "echo": true, "pwd": true, "whoami": true, "date": true,
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	if !allowedCommands[parts[0]] {
		return "", fmt.Errorf("command not allowed in sandbox: %s", parts[0])
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}
