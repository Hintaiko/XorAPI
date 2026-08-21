package relay

import "encoding/json"

// ---------- OpenAI 协议类型 ----------

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Index    *int     `json:"index,omitempty"`
	Function FuncCall `json:"function"`
}

type FuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Message struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	Name       string          `json:"name,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type OpenAITool struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"` // {name, description, parameters}
}

type OpenAIToolFn struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ChatRequest struct {
	Model            string          `json:"model"`
	Messages         []Message       `json:"messages"`
	MaxTokens        int             `json:"max_tokens,omitempty"`
	MaxCompletionTok int             `json:"max_completion_tokens,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"`
	TopP             *float64        `json:"top_p,omitempty"`
	Stream           bool            `json:"stream,omitempty"`
	StreamOptions    json.RawMessage `json:"stream_options,omitempty"`
	Stop             json.RawMessage `json:"stop,omitempty"`
	Tools            []OpenAITool    `json:"tools,omitempty"`
	ToolChoice       json.RawMessage `json:"tool_choice,omitempty"`
	User             string          `json:"user,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type RespMessage struct {
	Role      string     `json:"role,omitempty"`
	Content   any        `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Choice struct {
	Index        int          `json:"index"`
	Message      *RespMessage `json:"message,omitempty"`
	Delta        *RespMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type ChatChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// ---------- Anthropic 协议类型 ----------

type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type Block struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// tool_result
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	// image
	Source *ImageSource `json:"source,omitempty"`
	// thinking
	Thinking string `json:"thinking,omitempty"`
}

type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type AnthropicRequest struct {
	Model         string             `json:"model"`
	System        json.RawMessage    `json:"system,omitempty"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Tools         []AnthropicTool    `json:"tools,omitempty"`
	ToolChoice    json.RawMessage    `json:"tool_choice,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"message"`
	Role       string          `json:"role"`
	Model      string          `json:"model"`
	Content    []Block         `json:"content"`
	StopReason string          `json:"stop_reason,omitempty"`
	Usage      AnthropicUsage  `json:"usage"`
}

// AnthropicError 客户端为 Anthropic 风格时的错误格式
type AnthropicError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ---------- OpenAI content 解析辅助 ----------

type openaiContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

// extractText 从 OpenAI content（string 或 parts 数组）提取文本
func extractText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []openaiContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		out := ""
		for _, p := range parts {
			if p.Type == "text" {
				out += p.Text
			}
		}
		return out
	}
	return ""
}

// extractImages 提取图片（返回 base64 数据与媒体类型）
func extractImages(raw json.RawMessage) []ImageSource {
	if len(raw) == 0 {
		return nil
	}
	var parts []openaiContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil
	}
	var out []ImageSource
	for _, p := range parts {
		if p.Type == "image_url" && p.ImageURL != nil {
			url := p.ImageURL.URL
			if mediaType, data, ok := parseDataURL(url); ok {
				out = append(out, ImageSource{Type: "base64", MediaType: mediaType, Data: data})
			}
		}
	}
	return out
}

func parseDataURL(url string) (mediaType, data string, ok bool) {
	const prefix = "data:"
	if len(url) > 5 && url[:5] == prefix {
		if idx := indexByteStr(url, ';'); idx > 0 {
			if idx2 := indexByteStr(url, ','); idx2 > idx {
				return url[5:idx], url[idx2+1:], true
			}
		}
	}
	return "", "", false
}

func indexByteStr(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// blockText 从 Anthropic content（string 或 []Block）提取文本
func blockText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []Block
	if err := json.Unmarshal(raw, &blocks); err == nil {
		out := ""
		for _, b := range blocks {
			if b.Type == "text" {
				out += b.Text
			}
		}
		return out
	}
	return ""
}
