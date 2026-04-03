package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/thkx/deepagent/llms"
)

const (
	defaultBaseURL    = "https://api.anthropic.com"
	defaultAPIVersion = "2023-06-01"
	defaultMaxTokens  = 2048
)

type Anthropic struct {
	apiKey     string
	model      string
	baseURL    string
	version    string
	maxTokens  int
	httpClient *http.Client
}

func New(apiKey, model, baseURL string) *Anthropic {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Anthropic{
		apiKey:     apiKey,
		model:      model,
		baseURL:    strings.TrimRight(baseURL, "/"),
		version:    defaultAPIVersion,
		maxTokens:  defaultMaxTokens,
		httpClient: &http.Client{},
	}
}

type requestMessage struct {
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
}

type requestTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
}

type messageRequest struct {
	Model     string           `json:"model"`
	System    string           `json:"system,omitempty"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []requestMessage `json:"messages"`
	Tools     []requestTool    `json:"tools,omitempty"`
}

type responseContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type messageResponse struct {
	Content []responseContentBlock `json:"content"`
}

func (a *Anthropic) Invoke(ctx context.Context, messages []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	if strings.TrimSpace(a.apiKey) == "" {
		return "", nil, fmt.Errorf("anthropic api key is required")
	}
	reqBody := messageRequest{
		Model:     a.model,
		MaxTokens: a.maxTokens,
	}
	for _, m := range messages {
		switch m.Role {
		case "system":
			if reqBody.System == "" {
				reqBody.System = m.Content
			} else {
				reqBody.System += "\n" + m.Content
			}
		case "assistant", "user":
			reqBody.Messages = append(reqBody.Messages, requestMessage{
				Role: m.Role,
				Content: []map[string]any{
					{"type": "text", "text": m.Content},
				},
			})
		case "tool":
			if block, ok := parseToolResultBlock(m.Content); ok {
				reqBody.Messages = append(reqBody.Messages, requestMessage{
					Role:    "user",
					Content: []map[string]any{block},
				})
			} else {
				// Fallback for legacy tool message format.
				reqBody.Messages = append(reqBody.Messages, requestMessage{
					Role: "user",
					Content: []map[string]any{
						{"type": "text", "text": "Tool result: " + m.Content},
					},
				})
			}
		}
	}
	for _, t := range tools {
		schema := llms.GenerateParametersSchema(t)
		reqBody.Tools = append(reqBody.Tools, requestTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, err
	}
	endpoint := a.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", a.version)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("anthropic api error: %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed messageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", nil, err
	}

	toolCalls := make([]llms.ToolCall, 0)
	textParts := make([]string, 0)
	for _, block := range parsed.Content {
		switch block.Type {
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, llms.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(args),
			})
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		}
	}
	if len(toolCalls) > 0 {
		return "", toolCalls, nil
	}
	return strings.Join(textParts, "\n"), nil, nil
}

func parseToolResultBlock(content string) (map[string]any, bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, false
	}
	toolUseID, _ := payload["tool_call_id"].(string)
	if strings.TrimSpace(toolUseID) == "" {
		return nil, false
	}

	ok, _ := payload["ok"].(bool)
	resultContent := ""
	if ok {
		if data, exists := payload["data"]; exists {
			b, _ := json.Marshal(data)
			resultContent = string(b)
		} else {
			resultContent = "ok"
		}
	} else {
		if e, _ := payload["error"].(string); strings.TrimSpace(e) != "" {
			resultContent = e
		} else {
			resultContent = "tool execution failed"
		}
	}

	return map[string]any{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
		"content": []map[string]any{
			{"type": "text", "text": resultContent},
		},
	}, true
}
