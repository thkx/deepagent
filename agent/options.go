package agent

import (
	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type Options struct {
	Model                  string
	Provider               string // openai | anthropic | ollama | groq
	APIKey                 string
	BaseURL                string
	LLM                    llms.ChatModel
	Tools                  []tools.Tool
	SystemPrompt           string
	Backend                fs.Backend
	Checkpointer           Checkpointer
	Memory                 memory.Store
	SkillsDir              string
	HitlConfig             InterruptConfig
	HitlApprover           ApproverFunc
	HitlAuditLogger        HITLAuditLogger
	HitlAuditIncludeArgs   bool
	HitlAuditVerifyOnStart bool
	Name                   string
	Logger                 Logger
	ExecuteConfig          *builtin.ExecuteConfig
}
