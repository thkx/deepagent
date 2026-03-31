package main

import (
	"context"
	"fmt"

	"github.com/thkx/deepagent/agent"
)

func main() {
	agt, err := agent.CreateDeepAgent(agent.Options{
		Model:        "gpt-4o",
		SystemPrompt: "You are an expert deep agent.",
		SkillsDir:    "./skills",
		HitlConfig: agent.InterruptConfig{
			"execute": true, // 需要人工审批 execute 工具
		},
		Checkpointer: agent.NewFileCheckpointer("./checkpoints"),
	})
	if err != nil {
		panic(err)
	}

	result, err := agt.Invoke(context.Background(), agent.Input{
		Messages: []agent.Message{
			{Role: "user", Content: "Load skill 'research', create plan.md using write_todos, then execute a test command."},
		},
		ThreadID: "thread-002",
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("✅ DeepAgent 完成！\nFinal: %+v\n", result)
}
