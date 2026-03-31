package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DiskBackend struct {
	RootDir string
}

func NewDiskBackend(root string) Backend {
	if root == "" {
		root = "./deepagent_fs"
	}
	os.MkdirAll(root, 0755)
	return &DiskBackend{RootDir: root}
}

// 安全路径清理（防止目录遍历攻击）
func (b *DiskBackend) safePath(path string) (string, error) {
	if path == "" {
		path = "."
	}
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") || filepath.IsAbs(clean) && !strings.HasPrefix(clean, b.RootDir) {
		return "", fmt.Errorf("path traversal not allowed: %s", path)
	}
	full := filepath.Join(b.RootDir, clean)
	return full, nil
}

func (b *DiskBackend) List(ctx context.Context, path string) ([]string, error) {
	full, err := b.safePath(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var list []string
	for _, e := range entries {
		list = append(list, e.Name())
	}
	return list, nil
}

func (b *DiskBackend) Read(ctx context.Context, path string) (string, error) {
	full, err := b.safePath(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (b *DiskBackend) Write(ctx context.Context, path, content string) error {
	full, err := b.safePath(path)
	if err != nil {
		return err
	}
	// 原子写（先写临时文件）
	tmp := full + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, full)
}

func (b *DiskBackend) Edit(ctx context.Context, path, instructions string) error {
	full, err := b.safePath(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte{}
		} else {
			return err
		}
	}
	// 简化 LLM 辅助编辑（实际可调用 LLM 编辑）
	newContent := string(data) + "\n// Edited by instructions: " + instructions
	return b.Write(ctx, path, newContent)
}
