package relay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type StreamStats struct {
	Usage     Usage
	TextBytes int
}

func (s *StreamStats) EstimateIfNeeded() {
	if s.Usage.PromptTokens == 0 && s.Usage.CompletionTokens == 0 && s.TextBytes > 0 {
		s.Usage.PromptTokens = 0
		s.Usage.CompletionTokens = (s.TextBytes + 3) / 4
		s.Usage.TotalTokens = s.Usage.CompletionTokens
	}
	s.Usage.TotalTokens = s.Usage.PromptTokens + s.Usage.CompletionTokens
}

func newScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 4*1024*1024)
	return sc
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func writeSSE(w http.ResponseWriter, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	flush(w)
}

func writeAnthropicEvent(w http.ResponseWriter, event string, payload any) {
	b, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	flush(w)
}

// anthropic SSE 事件载荷
type ssePayload struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	Delta        json.RawMessage `json:"delta"`
	ContentBlock *Block          `json:"content_block"`
	Message      *struct {
		ID    string         `json:"id"`
		Model string         `json:"model"`
		Usage *AnthropicUsage `json:"usage"`
	} `json:"message"`
	Usage *AnthropicUsage `json:"usage"`
}

type sseDelta struct {
	Type        string          `json:"type"`
	Text        string          `json:"text"`
	PartialJSON string          `json:"partial_json"`
	StopReason  string          `json:"stop_reason"`
	Usage       *AnthropicUsage `json:"usage"`
}

// RelayAnthropicSSEToOpenAI 上游 Anthropic 流 -> 客户端 OpenAI 流
func RelayAnthropicSSEToOpenAI(w http.ResponseWriter, src io.Reader, model string) (*StreamStats, error) {
	stats := &StreamStats{}
	sc := newScanner(src)
	blockKind := map[int]string{} // block index -> kind
	toolIdx := map[int]int{}      // block index -> tool call index
	nextTool := 0
	finish := ""
	id := "chatcmpl-" + randHex(12)

	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var p ssePayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			continue
		}
		switch p.Type {
		case "message_start":
			if p.Message != nil {
				if p.Message.Usage != nil {
					stats.Usage.PromptTokens = p.Message.Usage.InputTokens
				}
			}
			chunk := ChatChunk{ID: id, Object: "chat.completion.chunk", Created: nowUnix(), Model: model,
				Choices: []Choice{{Index: 0, Delta: &RespMessage{Role: "assistant", Content: ""}, FinishReason: nil}}}
			writeSSE(w, string(mustJSON(chunk)))
		case "content_block_start":
			if p.ContentBlock != nil {
				blockKind[p.Index] = p.ContentBlock.Type
				if p.ContentBlock.Type == "tool_use" {
					idx := nextTool
					toolIdx[p.Index] = idx
					nextTool++
					chunk := ChatChunk{ID: id, Object: "chat.completion.chunk", Created: nowUnix(), Model: model,
						Choices: []Choice{{Index: 0, Delta: &RespMessage{ToolCalls: []ToolCall{{
							ID: p.ContentBlock.ID, Type: "function", Index: &idx,
							Function: FuncCall{Name: p.ContentBlock.Name, Arguments: ""},
						}}}, FinishReason: nil}}}
					writeSSE(w, string(mustJSON(chunk)))
				}
			}
		case "content_block_delta":
			var d sseDelta
			if err := json.Unmarshal(p.Delta, &d); err != nil {
				continue
			}
			if d.Type == "text_delta" && d.Text != "" {
				stats.TextBytes += len(d.Text)
				chunk := ChatChunk{ID: id, Object: "chat.completion.chunk", Created: nowUnix(), Model: model,
					Choices: []Choice{{Index: 0, Delta: &RespMessage{Content: d.Text}, FinishReason: nil}}}
				writeSSE(w, string(mustJSON(chunk)))
			} else if d.Type == "input_json_delta" && d.PartialJSON != "" {
				idx := toolIdx[p.Index]
				chunk := ChatChunk{ID: id, Object: "chat.completion.chunk", Created: nowUnix(), Model: model,
					Choices: []Choice{{Index: 0, Delta: &RespMessage{ToolCalls: []ToolCall{{
						Index: &idx, Function: FuncCall{Arguments: d.PartialJSON},
					}}}, FinishReason: nil}}}
				writeSSE(w, string(mustJSON(chunk)))
			}
		case "message_delta":
			var d sseDelta
			if err := json.Unmarshal(p.Delta, &d); err == nil {
				if d.StopReason != "" {
					finish = mapStopReasonAToO(d.StopReason)
				}
				if d.Usage != nil {
					stats.Usage.CompletionTokens = d.Usage.OutputTokens
				}
			}
		case "message_stop":
			if finish == "" {
				finish = "stop"
			}
			stats.Usage.TotalTokens = stats.Usage.PromptTokens + stats.Usage.CompletionTokens
			chunk := ChatChunk{ID: id, Object: "chat.completion.chunk", Created: nowUnix(), Model: model,
				Choices:  []Choice{{Index: 0, Delta: &RespMessage{}, FinishReason: &finish}},
				Usage:    &Usage{PromptTokens: stats.Usage.PromptTokens, CompletionTokens: stats.Usage.CompletionTokens, TotalTokens: stats.Usage.TotalTokens},
			}
			writeSSE(w, string(mustJSON(chunk)))
			writeSSE(w, "[DONE]")
			return stats, nil
		case "error":
			return stats, fmt.Errorf("上游流式错误: %s", data)
		}
	}
	return stats, sc.Err()
}

// RelayOpenAISSEToAnthropic 上游 OpenAI 流 -> 客户端 Anthropic 流
func RelayOpenAISSEToAnthropic(w http.ResponseWriter, src io.Reader, model string) (*StreamStats, error) {
	stats := &StreamStats{}
	sc := newScanner(src)
	started := false
	curBlock := -1
	curKind := ""
	toolBlock := map[int]int{}  // tool index -> block index
	nextBlock := 0
	stopReason := "end_turn"
	id := "msg_" + randHex(12)

	finishUp := func() {
		if curBlock >= 0 {
			writeAnthropicEvent(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": curBlock})
		}
	}
	startText := func() {
		writeAnthropicEvent(w, "content_block_start", map[string]any{
			"type": "content_block_start", "index": nextBlock,
			"content_block": map[string]any{"type": "text", "text": ""},
		})
		curBlock = nextBlock
		curKind = "text"
		nextBlock++
	}

	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			finishUp()
			writeAnthropicEvent(w, "message_delta", map[string]any{
				"type":  "message_delta",
				"delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil},
				"usage": map[string]any{"output_tokens": stats.Usage.CompletionTokens},
			})
			writeAnthropicEvent(w, "message_stop", map[string]string{"type": "message_stop"})
			return stats, nil
		}
		var chunk ChatChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if !started {
			started = true
			writeAnthropicEvent(w, "message_start", map[string]any{
				"type": "message_start",
				"message": map[string]any{
					"id": id, "type": "message", "role": "assistant", "model": model,
					"content": []any{}, "stop_reason": nil, "stop_sequence": nil,
					"usage": map[string]any{"input_tokens": 0, "output_tokens": 0},
				},
			})
		}
		if chunk.Usage != nil && (chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0) {
			stats.Usage.PromptTokens = chunk.Usage.PromptTokens
			stats.Usage.CompletionTokens = chunk.Usage.CompletionTokens
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			stopReason = mapFinishOToA(*ch.FinishReason)
		}
		if ch.Delta == nil {
			continue
		}
		if s, ok := ch.Delta.Content.(string); ok && s != "" {
			stats.TextBytes += len(s)
			if curKind != "text" || curBlock < 0 {
				if curBlock >= 0 {
					writeAnthropicEvent(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": curBlock})
				}
				startText()
			}
			writeAnthropicEvent(w, "content_block_delta", map[string]any{
				"type": "content_block_delta", "index": curBlock,
				"delta": map[string]any{"type": "text_delta", "text": s},
			})
		}
		for _, tc := range ch.Delta.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			if _, seen := toolBlock[idx]; !seen {
				finishUp()
				writeAnthropicEvent(w, "content_block_start", map[string]any{
					"type": "content_block_start", "index": nextBlock,
					"content_block": map[string]any{"type": "tool_use", "id": tc.ID, "name": tc.Function.Name, "input": map[string]any{}},
				})
				toolBlock[idx] = nextBlock
				curBlock = nextBlock
				curKind = "tool_use"
				nextBlock++
			}
			if tc.Function.Arguments != "" {
				writeAnthropicEvent(w, "content_block_delta", map[string]any{
					"type": "content_block_delta", "index": toolBlock[idx],
					"delta": map[string]any{"type": "input_json_delta", "partial_json": tc.Function.Arguments},
				})
			}
		}
	}
	return stats, sc.Err()
}

// PassthroughSSE 原样转发 SSE（协议一致时），同时解析 usage 用于计费
func PassthroughSSE(w http.ResponseWriter, src io.Reader, upstreamProto string) (*StreamStats, error) {
	stats := &StreamStats{}
	sc := newScanner(src)
	var buf bytes.Buffer
	for sc.Scan() {
		line := sc.Text()
		fmt.Fprintf(w, "%s\n", line)
		if line == "" {
			flush(w)
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		captureUsage(stats, data, upstreamProto)
	}
	buf.Reset()
	return stats, sc.Err()
}

func captureUsage(stats *StreamStats, data string, proto string) {
	if proto == "anthropic" {
		var p ssePayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			return
		}
		switch p.Type {
		case "message_start":
			if p.Message != nil && p.Message.Usage != nil {
				stats.Usage.PromptTokens = p.Message.Usage.InputTokens
			}
		case "message_delta":
			var d sseDelta
			if err := json.Unmarshal(p.Delta, &d); err == nil && d.Usage != nil {
				stats.Usage.CompletionTokens = d.Usage.OutputTokens
			}
		}
		return
	}
	var chunk ChatChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return
	}
	if chunk.Usage != nil && (chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0) {
		stats.Usage.PromptTokens = chunk.Usage.PromptTokens
		stats.Usage.CompletionTokens = chunk.Usage.CompletionTokens
	}
	if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
		if s, ok := chunk.Choices[0].Delta.Content.(string); ok {
			stats.TextBytes += len(s)
		}
	}
}
