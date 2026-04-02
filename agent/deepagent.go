package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/llms/anthropic"
	"github.com/thkx/deepagent/llms/groq"
	"github.com/thkx/deepagent/llms/ollama"
	"github.com/thkx/deepagent/llms/openai"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func CreateDeepAgent(opts Options) (DeepAgent, error) {
	if opts.LLM == nil {
		llm, err := buildDefaultLLM(opts)
		if err != nil {
			return nil, err
		}
		opts.LLM = llm
	}
	if opts.Backend == nil {
		opts.Backend = fs.NewInMemoryBackend()
	}
	if opts.Checkpointer == nil {
		opts.Checkpointer = NewFileCheckpointer("")
	}
	if opts.Memory == nil {
		opts.Memory = memory.NewFileMemoryStore("")
	}
	if opts.HitlConfig == nil {
		opts.HitlConfig = make(InterruptConfig)
	}
	hitlAuditVerifyOnStart := opts.HitlAuditVerifyOnStart || strings.EqualFold(strings.TrimSpace(os.Getenv("DEEPAGENT_HITL_AUDIT_VERIFY_ON_START")), "true")
	hitlAuditIncludeArgs := opts.HitlAuditIncludeArgs || strings.EqualFold(strings.TrimSpace(os.Getenv("DEEPAGENT_HITL_AUDIT_INCLUDE_ARGS")), "true")
	if opts.HitlAuditLogger == nil {
		if path := strings.TrimSpace(os.Getenv("DEEPAGENT_HITL_AUDIT_FILE")); path != "" {
			if hitlAuditVerifyOnStart {
				if err := VerifyHITLAuditFileChain(path); err != nil && !os.IsNotExist(err) {
					return nil, fmt.Errorf("hitl audit chain verification failed: %w", err)
				}
			}
			auditLogger, err := NewFileHITLAuditLogger(path)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize hitl audit logger: %w", err)
			}
			opts.HitlAuditLogger = auditLogger
		}
	}

	skillsLoader := builtin.NewSkillsLoader(opts.SkillsDir)
	systemPrompt := opts.SystemPrompt + "\n\n" + skillsLoader.GetAllSkillsContext()

	allTools := append([]tools.Tool{}, opts.Tools...)
	allTools = append(allTools,
		builtin.NewWriteTodosTool(),
		NewTaskTool(opts), // 子代理
		fs.NewLSTool(opts.Backend),
		fs.NewReadFileTool(opts.Backend),
		fs.NewWriteFileTool(opts.Backend),
		fs.NewEditFileTool(opts.Backend),
		builtin.NewLoadSkillsTool(skillsLoader),
		builtin.NewExecuteToolWithConfig(opts.Backend, opts.ExecuteConfig),
	)

	graph := buildGraph(
		opts.LLM,
		allTools,
		systemPrompt,
		opts.Backend,
		opts.Checkpointer,
		opts.Memory,
		NewHumanInTheLoopWithApprover(opts.HitlConfig, opts.HitlApprover),
		opts.Logger,
	)
	graph.hitlAudit = opts.HitlAuditLogger
	graph.hitlAuditIncludeArgs = hitlAuditIncludeArgs

	return &deepAgentImpl{
		graph:        graph,
		backend:      opts.Backend,
		checkpointer: opts.Checkpointer,
	}, nil
}

func buildDefaultLLM(opts Options) (llms.ChatModel, error) {
	model := strings.TrimSpace(opts.Model)
	if model == "" {
		model = "gpt-4o"
	}

	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(os.Getenv("DEEPAGENT_PROVIDER")))
	}
	if provider == "" {
		provider = inferProviderFromModel(model)
	}

	switch provider {
	case "", "openai":
		apiKey := strings.TrimSpace(opts.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is required when provider is openai")
		}
		baseURL := strings.TrimSpace(opts.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
		}
		if baseURL != "" {
			return openai.NewWithBaseURL(apiKey, model, baseURL), nil
		}
		return openai.New(apiKey, model), nil
	case "anthropic":
		apiKey := strings.TrimSpace(opts.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY"))
		}
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required when provider is anthropic")
		}
		baseURL := strings.TrimSpace(opts.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL"))
		}
		return anthropic.New(apiKey, model, baseURL), nil
	case "groq":
		apiKey := strings.TrimSpace(opts.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
		}
		if apiKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY is required when provider is groq")
		}
		baseURL := strings.TrimSpace(opts.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("GROQ_BASE_URL"))
		}
		return groq.New(apiKey, model, baseURL), nil
	case "ollama":
		baseURL := strings.TrimSpace(opts.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))
		}
		return ollama.New(model, baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (expected one of openai, anthropic, groq, ollama)", provider)
	}
}

func inferProviderFromModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "claude"):
		return "anthropic"
	case strings.HasPrefix(m, "groq/"):
		return "groq"
	case strings.HasPrefix(m, "ollama/"):
		return "ollama"
	default:
		return "openai"
	}
}

type deepAgentImpl struct {
	graph        *Graph
	backend      fs.Backend
	checkpointer Checkpointer
}

func (a *deepAgentImpl) Invoke(ctx context.Context, input Input) (Output, error) {
	return a.graph.Run(ctx, input)
}

func (a *deepAgentImpl) Stream(ctx context.Context, input Input) (<-chan Event, error) {
	ch := make(chan Event, 32)
	go func() {
		defer close(ch)
		_, err := a.graph.runWithEventSink(ctx, input, func(event Event) {
			ch <- event
		})
		if err != nil {
			// error event is emitted from graph defer
			return
		}
	}()
	return ch, nil
}
