package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thkx/deepagent/tools"
)

type SkillsLoader struct {
	SkillsDir string
	skills    map[string]string // name -> content
}

func NewSkillsLoader(dir string) *SkillsLoader {
	if dir == "" {
		dir = "./skills"
	}
	_ = os.MkdirAll(dir, 0755)
	loader := &SkillsLoader{SkillsDir: dir, skills: make(map[string]string)}
	loader.loadAll()
	return loader
}

func (l *SkillsLoader) loadAll() {
	files, _ := filepath.Glob(filepath.Join(l.SkillsDir, "*.md"))
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		content, _ := os.ReadFile(f)
		l.skills[name] = string(content)
	}
}

func NewLoadSkillsTool(loader *SkillsLoader) tools.Tool {
	return tools.NewTool("load_skill",
		"Load a skill from the skills directory by name.",
		func(ctx context.Context, args map[string]any) (any, error) {
			name, ok := args["name"].(string)
			if !ok || name == "" {
				return nil, fmt.Errorf("skill name is required")
			}
			if content, ok := loader.skills[name]; ok {
				return map[string]any{"name": name, "content": content}, nil
			}
			return nil, fmt.Errorf("skill %s not found", name)
		},
	)
}

// GetAllSkillsContext 返回所有 skills 拼接的上下文（供 system prompt 使用）
func (l *SkillsLoader) GetAllSkillsContext() string {
	var sb strings.Builder
	for name, content := range l.skills {
		sb.WriteString(fmt.Sprintf("=== Skill: %s ===\n%s\n\n", name, content))
	}
	return sb.String()
}
