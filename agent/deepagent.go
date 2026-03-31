package agent

import (
	"context"

	"github.com/thkx/deepagent/llms/openai"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func CreateDeepAgent(opts Options) (DeepAgent, error) {
	if opts.LLM == nil {
		if opts.Model == "" {
			opts.Model = "gpt-4o"
		}
		opts.LLM = openai.New("sk-your-openai-api-key-here", opts.Model)
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
		builtin.NewExecuteTool(opts.Backend),
	)

	graph := buildGraph(opts.LLM, allTools, systemPrompt, opts.Backend, opts.Checkpointer, opts.Memory, NewHumanInTheLoop(opts.HitlConfig))

	return &deepAgentImpl{
		graph:        graph,
		backend:      opts.Backend,
		checkpointer: opts.Checkpointer,
	}, nil
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
		out, err := a.Invoke(ctx, input)
		if err != nil {
			ch <- Event{Type: "error", Content: err.Error()}
			return
		}
		ch <- Event{Type: "final", Content: out}
	}()
	return ch, nil
}
