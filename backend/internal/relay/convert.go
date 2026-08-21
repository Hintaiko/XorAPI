package relay

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------- 请求转换：OpenAI -> Anthropic ----------

func OpenAIToAnthropic(req *ChatRequest) (*AnthropicRequest, error) {
	out := &AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}
	if out.MaxTokens == 0 {
		out.MaxTokens = req.MaxCompletionTok
	}
	if out.MaxTokens == 0 {
		out.MaxTokens = 8192
	}
	// stop
	if len(req.Stop) > 0 {
		var one string
		if err := json.Unmarshal(req.Stop, &one); err == nil {
			out.StopSequences = []string{one}
		} else {
			var many []string
			if err := json.Unmarshal(req.Stop, &many); err == nil {
				out.StopSequences = many
			}
		}
	}
	// tools
	for _, t := range req.Tools {
		var fn OpenAIToolFn
		if err := json.Unmarshal(t.Function, &fn); err != nil {
			continue
		}
		if fn.Name == "" {
			continue
		}
		schema := fn.Parameters
		if len(schema) == 0 {
			schema = json.RawMessage(`{"type":"object"}`)
		}
		out.Tools = append(out.Tools, AnthropicTool{
			Name: fn.Name, Description: fn.Description, InputSchema: schema,
		})
	}
	if len(out.Tools) > 0 {
		switch strings.TrimSpace(string(req.ToolChoice)) {
		case `"auto"`, `{"type":"auto"}`:
			out.ToolChoice = json.RawMessage(`{"type":"auto"}`)
		case `"required"`, `{"type":"required"}`:
			out.ToolChoice = json.RawMessage(`{"type":"any"}`)
		case `"none"`:
			out.Tools = nil
		default:
			if strings.Contains(string(req.ToolChoice), `"function"`) {
				var tc struct {
					Type     string `json:"type"`
					Function struct {
						Name string `json:"name"`
					} `json:"function"`
				}
				if err := json.Unmarshal(req.ToolChoice, &tc); err == nil && tc.Function.Name != "" {
					out.ToolChoice = json.RawMessage(fmt.Sprintf(`{"type":"tool","name":%q}`, tc.Function.Name))
				}
			}
		}
	}

	// messages
	var systemParts []string
	for _, m := range req.Messages {
		switch m.Role {
		case "system", "developer":
			if t := extractText(m.Content); t != "" {
				systemParts = append(systemParts, t)
			}
		case "user":
			blocks := userBlocks(m)
			out.Messages = append(out.Messages, AnthropicMessage{Role: "user", Content: mustJSON(blocks)})
		case "assistant":
			out.Messages = append(out.Messages, AnthropicMessage{Role: "assistant", Content: mustJSON(assistantBlocks(m))})
		case "tool":
			blocks := []Block{{
				Type: "tool_result", ToolUseID: m.ToolCallID,
				Content: json.RawMessage(mustJSONStr(blockText(m.Content))),
			}}
			if n := len(out.Messages); n > 0 && out.Messages[n-1].Role == "user" {
				var prev []Block
				if err := json.Unmarshal(out.Messages[n-1].Content, &prev); err == nil {
					out.Messages[n-1].Content = mustJSON(append(prev, blocks...))
					continue
				}
			}
			out.Messages = append(out.Messages, AnthropicMessage{Role: "user", Content: mustJSON(blocks)})
		}
	}
	if len(systemParts) > 0 {
		out.System = json.RawMessage(mustJSONStr(strings.Join(systemParts, "\n\n")))
	}
	// Anthropic 要求首条消息为 user
	if len(out.Messages) == 0 || out.Messages[0].Role != "user" {
		out.Messages = append([]AnthropicMessage{{Role: "user", Content: json.RawMessage(`" "`)}}, out.Messages...)
	}
	return out, nil
}

func userBlocks(m Message) []Block {
	var blocks []Block
	if text := extractText(m.Content); text != "" {
		blocks = append(blocks, Block{Type: "text", Text: text})
	}
	for _, img := range extractImages(m.Content) {
		blocks = append(blocks, Block{Type: "image", Source: &img})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, Block{Type: "text", Text: ""})
	}
	return blocks
}

func assistantBlocks(m Message) []Block {
	var blocks []Block
	if text := extractText(m.Content); text != "" {
		blocks = append(blocks, Block{Type: "text", Text: text})
	}
	for _, tc := range m.ToolCalls {
		args := tc.Function.Arguments
		if args == "" {
			args = "{}"
		}
		if !json.Valid([]byte(args)) {
			args = string(mustJSON(map[string]string{"_raw": args}))
		}
		blocks = append(blocks, Block{Type: "tool_use", ID: tc.ID, Name: tc.Function.Name, Input: json.RawMessage(args)})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, Block{Type: "text", Text: ""})
	}
	return blocks
}

// ---------- 请求转换：Anthropic -> OpenAI ----------

func AnthropicToOpenAI(req *AnthropicRequest) *ChatRequest {
	out := &ChatRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}
	if len(req.StopSequences) > 0 {
		out.Stop = mustJSON(req.StopSequences)
	}
	for _, t := range req.Tools {
		fn := OpenAIToolFn{Name: t.Name, Description: t.Description, Parameters: t.InputSchema}
		out.Tools = append(out.Tools, OpenAITool{Type: "function", Function: mustJSON(fn)})
	}
	if len(out.Tools) > 0 {
		switch strings.TrimSpace(string(req.ToolChoice)) {
		case `{"type":"auto"}`:
			out.ToolChoice = json.RawMessage(`"auto"`)
		case `{"type":"any"}`:
			out.ToolChoice = json.RawMessage(`"required"`)
		default:
			if strings.Contains(string(req.ToolChoice), `"name"`) {
				var tc struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(req.ToolChoice, &tc); err == nil && tc.Name != "" {
					out.ToolChoice = json.RawMessage(fmt.Sprintf(`{"type":"function","function":{"name":%q}}`, tc.Name))
				}
			}
		}
	}
	// system
	if sys := blockText(req.System); sys != "" {
		out.Messages = append(out.Messages, Message{Role: "system", Content: json.RawMessage(mustJSONStr(sys))})
	}
	for _, m := range req.Messages {
		var blocks []Block
		if err := json.Unmarshal(m.Content, &blocks); err != nil {
			// string content
			out.Messages = append(out.Messages, Message{Role: m.Role, Content: json.RawMessage(mustJSONStr(blockText(m.Content)))})
			continue
		}
		// 先收集 tool_result -> role:tool；其余按角色映射
		var pendingTool []Message
		var textParts []string
		var images []ImageSource
		var toolCalls []ToolCall
		for _, b := range blocks {
			switch b.Type {
			case "text":
				textParts = append(textParts, b.Text)
			case "image":
				if b.Source != nil {
					images = append(images, *b.Source)
				}
			case "tool_use":
				input := string(b.Input)
				if input == "" {
					input = "{}"
				}
				toolCalls = append(toolCalls, ToolCall{
					ID: b.ID, Type: "function", Function: FuncCall{Name: b.Name, Arguments: input},
				})
			case "tool_result":
				res := blockText(b.Content)
				pendingTool = append(pendingTool, Message{
					Role: "tool", ToolCallID: b.ToolUseID, Content: json.RawMessage(mustJSONStr(res)),
				})
			}
		}
		if m.Role == "assistant" {
			msg := Message{Role: "assistant", Content: json.RawMessage(mustJSONStr(strings.Join(textParts, ""))), ToolCalls: toolCalls}
			if len(toolCalls) > 0 {
				msg.Content = json.RawMessage("null")
			}
			out.Messages = append(out.Messages, msg)
		} else {
			if len(images) > 0 {
				var parts []map[string]any
				if t := strings.Join(textParts, ""); t != "" {
					parts = append(parts, map[string]any{"type": "text", "text": t})
				}
				for _, img := range images {
					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": map[string]string{"url": "data:" + img.MediaType + ";base64," + img.Data},
					})
				}
				out.Messages = append(out.Messages, Message{Role: "user", Content: mustJSON(parts)})
			} else {
				out.Messages = append(out.Messages, Message{Role: "user", Content: json.RawMessage(mustJSONStr(strings.Join(textParts, "")))})
			}
			out.Messages = append(out.Messages, pendingTool...)
		}
	}
	return out
}

// ---------- 响应转换 ----------

func mapStopReasonAToO(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		if reason == "" {
			return "stop"
		}
		return "stop"
	}
}

func mapFinishOToA(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls", "function_call":
		return "tool_use"
	case "content_filter":
		return "refusal"
	default:
		return "end_turn"
	}
}

// AnthropicRespToOpenAI 非流式响应转换
func AnthropicRespToOpenAI(resp *AnthropicResponse, model string) *ChatResponse {
	out := &ChatResponse{
		ID: resp.ID, Object: "chat.completion", Created: nowUnix(), Model: model,
		Usage: &Usage{
			PromptTokens: resp.Usage.InputTokens, CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens: resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
	var sb strings.Builder
	var toolCalls []ToolCall
	for _, b := range resp.Content {
		switch b.Type {
		case "text":
			sb.WriteString(b.Text)
		case "tool_use":
			input := string(b.Input)
			if input == "" {
				input = "{}"
			}
			toolCalls = append(toolCalls, ToolCall{ID: b.ID, Type: "function", Function: FuncCall{Name: b.Name, Arguments: input}})
		}
	}
	msg := &RespMessage{Role: "assistant", Content: sb.String()}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
		if sb.Len() == 0 {
			msg.Content = nil
		}
	}
	finish := mapStopReasonAToO(resp.StopReason)
	out.Choices = []Choice{{Index: 0, Message: msg, FinishReason: &finish}}
	return out
}

// OpenAIRespToAnthropic 非流式响应转换
func OpenAIRespToAnthropic(resp *ChatResponse, model string) *AnthropicResponse {
	out := &AnthropicResponse{
		ID: resp.ID, Type: "message", Role: "assistant", Model: model,
	}
	if resp.Usage != nil {
		out.Usage = AnthropicUsage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens}
	}
	if len(resp.Choices) > 0 {
		ch := resp.Choices[0]
		if ch.Message != nil {
			if s, ok := ch.Message.Content.(string); ok && s != "" {
				out.Content = append(out.Content, Block{Type: "text", Text: s})
			}
			for _, tc := range ch.Message.ToolCalls {
				args := tc.Function.Arguments
				if args == "" {
					args = "{}"
				}
				out.Content = append(out.Content, Block{Type: "tool_use", ID: tc.ID, Name: tc.Function.Name, Input: json.RawMessage(args)})
			}
		}
		if ch.FinishReason != nil {
			out.StopReason = mapFinishOToA(*ch.FinishReason)
		}
	}
	if out.StopReason == "" {
		out.StopReason = "end_turn"
	}
	if out.Content == nil {
		out.Content = []Block{{Type: "text", Text: ""}}
	}
	return out
}

// ---------- 工具函数 ----------

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}

func mustJSONStr(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
