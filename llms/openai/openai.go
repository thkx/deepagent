package openai

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
	"github.com/thkx/deepagent/llms"
)

type OpenAI struct {
	client *openai.Client
	model  string
}

func New(apiKey, model string) *OpenAI {
	return &OpenAI{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (o *OpenAI) Invoke(ctx context.Context, msgs []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	var openaiMsgs []openai.ChatCompletionMessage
	for _, m := range msgs {
		openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Build request with tools if provided
	req := openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: openaiMsgs,
	}

	// Convert tools to OpenAI format if provided
	if len(tools) > 0 {
		var toolDefs []openai.Tool
		for _, tool := range tools {
			parametersSchema := llms.GenerateParametersSchema(tool)
			toolDef := openai.Tool{
				Type: "function",
				Function: &openai.FunctionDefinition{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  parametersSchema,
				},
			}
			toolDefs = append(toolDefs, toolDef)
		}

		req.Tools = toolDefs
		if len(toolDefs) > 0 {
			req.ToolChoice = "auto"
		}
	}

	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", nil, err
	}

	choice := resp.Choices[0].Message
	if len(choice.ToolCalls) > 0 {
		var calls []llms.ToolCall
		for _, tc := range choice.ToolCalls {
			calls = append(calls, llms.ToolCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
		return "", calls, nil
	}
	return choice.Content, nil, nil
}
