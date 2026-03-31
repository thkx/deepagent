package agent

import (
	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type Options struct {
	Model        string
	LLM          llms.ChatModel
	Tools        []tools.Tool
	SystemPrompt string
	Backend      fs.Backend
	Checkpointer Checkpointer
	Memory       memory.Store
	SkillsDir    string
	HitlConfig   InterruptConfig
	Name         string
}
