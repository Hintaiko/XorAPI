package relay

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"xorapi/internal/model"
	"xorapi/internal/service"
	"xorapi/internal/store"

	"github.com/gin-gonic/gin"
)

const maxBodySize = 20 << 20 // 20MB

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     90 * time.Second,
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 120 * time.Second,
	},
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func nowUnix() int64 { return time.Now().Unix() }

func buildUpstreamRequest(plan service.ChannelPlan, clientProto string, rawBody []byte, openaiReq *ChatRequest, anthropicReq *AnthropicRequest, stream bool) (*http.Request, error) {
	ch := plan.Channel
	var url, body []byte
	var err error

	switch {
	case clientProto == "openai" && ch.Protocol == "openai":
		url = []byte(strings.TrimRight(ch.BaseURL, "/") + "/chat/completions")
		if stream && (len(openaiReq.StreamOptions) == 0 || !strings.Contains(string(openaiReq.StreamOptions), "include_usage")) {
			var m map[string]any
			if err := json.Unmarshal(rawBody, &m); err == nil {
				m["stream_options"] = map[string]any{"include_usage": true}
				body, _ = json.Marshal(m)
			} else {
				body = rawBody
			}
		} else {
			body = rawBody
		}
	case clientProto == "anthropic" && ch.Protocol == "anthropic":
		url = []byte(strings.TrimRight(ch.BaseURL, "/") + "/v1/messages")
		body = rawBody
	case clientProto == "openai" && ch.Protocol == "anthropic":
		url = []byte(strings.TrimRight(ch.BaseURL, "/") + "/v1/messages")
		areq, err := OpenAIToAnthropic(openaiReq)
		if err != nil {
			return nil, err
		}
		areq.Stream = stream
		body, err = json.Marshal(areq)
		if err != nil {
			return nil, err
		}
	case clientProto == "anthropic" && ch.Protocol == "openai":
		url = []byte(strings.TrimRight(ch.BaseURL, "/") + "/chat/completions")
		oreq := AnthropicToOpenAI(anthropicReq)
		oreq.Stream = stream
		if stream {
			oreq.StreamOptions = json.RawMessage(`{"include_usage":true}`)
		}
		body, err = json.Marshal(oreq)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不支持的协议组合")
	}

	req, err := http.NewRequest(http.MethodPost, string(url), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	key, _ := service.Decrypt(store.Cfg.Server.EncryptKey, ch.APIKeyEnc)
	if ch.Protocol == "anthropic" {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return req, nil
}

func writeClientError(c *gin.Context, clientProto string, status int, msg string) {
	if clientProto == "anthropic" {
		var ae AnthropicError
		ae.Type = "error"
		ae.Error.Type = "api_error"
		ae.Error.Message = msg
		c.JSON(status, ae)
		return
	}
	c.JSON(status, gin.H{"error": gin.H{"message": msg, "type": "api_error"}})
}

// HandleRelay 统一中继入口
func HandleRelay(c *gin.Context, clientProto string) {
	ak := c.MustGet("apikey").(*model.APIKey)
	user := c.MustGet("user").(*model.User)

	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodySize))
	if err != nil || len(raw) == 0 {
		writeClientError(c, clientProto, http.StatusBadRequest, "请求体为空或读取失败")
		return
	}

	modelName, stream := "", false
	var openaiReq ChatRequest
	var anthropicReq AnthropicRequest
	if clientProto == "openai" {
		if err := json.Unmarshal(raw, &openaiReq); err != nil {
			writeClientError(c, clientProto, http.StatusBadRequest, "请求体不是合法的 OpenAI 格式: "+err.Error())
			return
		}
		modelName, stream = openaiReq.Model, openaiReq.Stream
	} else {
		if err := json.Unmarshal(raw, &anthropicReq); err != nil {
			writeClientError(c, clientProto, http.StatusBadRequest, "请求体不是合法的 Anthropic 格式: "+err.Error())
			return
		}
		modelName, stream = anthropicReq.Model, anthropicReq.Stream
	}
	if modelName == "" {
		writeClientError(c, clientProto, http.StatusBadRequest, "缺少 model 参数")
		return
	}

	plans, ok := service.ResolveChannels(modelName)
	if !ok {
		writeClientError(c, clientProto, http.StatusNotFound, "模型不存在或未开放调用: "+modelName)
		return
	}

	start := time.Now()
	ip, _ := c.Get("client_ip")
	ipStr, _ := ip.(string)
	if ipStr == "" {
		ipStr = c.ClientIP()
	}

	var lastErr string
	for _, plan := range plans {
		req, err := buildUpstreamRequest(plan, clientProto, raw, &openaiReq, &anthropicReq, stream)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = "上游连接失败: " + err.Error()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			lastErr = fmt.Sprintf("上游渠道 %s 返回 %d: %s", plan.Channel.Name, resp.StatusCode, truncate(string(errBody), 300))
			continue
		}
		// 成功：流式/非流式处理
		latency := time.Since(start).Milliseconds()
		if stream {
			handleStreamSuccess(c, resp, plan, clientProto, modelName, ak, user, latency, ipStr)
		} else {
			handleNonStreamSuccess(c, resp, plan, clientProto, modelName, ak, user, latency, ipStr)
		}
		return
	}

	service.LogCall(user.ID, ak.ID, 0, 0, modelName, clientProto, ipStr, 0, 0, 0,
		time.Since(start).Milliseconds(), "fail", truncate(lastErr, 250), &model.ModelInfo{})
	writeClientError(c, clientProto, http.StatusBadGateway, "所有渠道均调用失败："+truncate(lastErr, 500))
}

func handleNonStreamSuccess(c *gin.Context, resp *http.Response, plan service.ChannelPlan,
	clientProto, modelName string, ak *model.APIKey, user *model.User, latency int64, ip string) {

	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 50<<20))
	if err != nil {
		writeClientError(c, clientProto, http.StatusBadGateway, "读取上游响应失败: "+err.Error())
		return
	}
	stats := &StreamStats{}
	if plan.Channel.Protocol == "anthropic" {
		var ar AnthropicResponse
		if err := json.Unmarshal(body, &ar); err == nil {
			stats.Usage.PromptTokens = ar.Usage.InputTokens
			stats.Usage.CompletionTokens = ar.Usage.OutputTokens
			for _, b := range ar.Content {
				if b.Type == "text" {
					stats.TextBytes += len(b.Text)
				}
			}
			if clientProto == "openai" {
				out := AnthropicRespToOpenAI(&ar, modelName)
				c.Data(http.StatusOK, "application/json; charset=utf-8", mustJSON(out))
			} else {
				c.Data(http.StatusOK, "application/json; charset=utf-8", body)
			}
		} else {
			c.Data(http.StatusOK, "application/json; charset=utf-8", body)
		}
	} else {
		var cr ChatResponse
		if err := json.Unmarshal(body, &cr); err == nil {
			if cr.Usage != nil {
				stats.Usage.PromptTokens = cr.Usage.PromptTokens
				stats.Usage.CompletionTokens = cr.Usage.CompletionTokens
			}
			for _, ch := range cr.Choices {
				if ch.Message != nil {
					if s, ok := ch.Message.Content.(string); ok {
						stats.TextBytes += len(s)
					}
				}
			}
			if clientProto == "anthropic" {
				out := OpenAIRespToAnthropic(&cr, modelName)
				c.Data(http.StatusOK, "application/json; charset=utf-8", mustJSON(out))
			} else {
				c.Data(http.StatusOK, "application/json; charset=utf-8", body)
			}
		} else {
			c.Data(http.StatusOK, "application/json; charset=utf-8", body)
		}
	}
	billAndLog(user, ak, plan, clientProto, modelName, stats, latency, "success", "", ip)
}

func handleStreamSuccess(c *gin.Context, resp *http.Response, plan service.ChannelPlan,
	clientProto, modelName string, ak *model.APIKey, user *model.User, latency int64, ip string) {

	defer resp.Body.Close()
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	var stats *StreamStats
	var err error
	switch {
	case clientProto == "openai" && plan.Channel.Protocol == "anthropic":
		stats, err = RelayAnthropicSSEToOpenAI(c.Writer, resp.Body, modelName)
	case clientProto == "anthropic" && plan.Channel.Protocol == "openai":
		stats, err = RelayOpenAISSEToAnthropic(c.Writer, resp.Body, modelName)
	default:
		stats, err = PassthroughSSE(c.Writer, resp.Body, plan.Channel.Protocol)
	}
	if stats == nil {
		stats = &StreamStats{}
	}
	if err != nil {
		billAndLog(user, ak, plan, clientProto, modelName, stats, latency, "fail", truncate(err.Error(), 250), ip)
		return
	}
	billAndLog(user, ak, plan, clientProto, modelName, stats, latency, "success", "", ip)
}

func billAndLog(user *model.User, ak *model.APIKey, plan service.ChannelPlan,
	clientProto, modelName string, stats *StreamStats, latency int64, status, errMsg, ip string) {

	stats.EstimateIfNeeded()
	cost := service.ComputeCost(&plan.Model, stats.Usage.PromptTokens, stats.Usage.CompletionTokens)
	service.LogCall(user.ID, ak.ID, plan.GroupID, plan.Channel.ID, modelName, clientProto, ip,
		stats.Usage.PromptTokens, stats.Usage.CompletionTokens, cost, latency, status, errMsg, &plan.Model)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// HandleModels GET /v1/models（供客户端发现可用模型）
func HandleModels(c *gin.Context) {
	var models []model.ModelInfo
	if err := store.DB.Where("callable = ?", true).Find(&models).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "数据库错误"}})
		return
	}
	data := make([]gin.H, 0, len(models))
	seen := map[string]bool{}
	for _, m := range models {
		if seen[m.Name] {
			continue
		}
		seen[m.Name] = true
		data = append(data, gin.H{"id": m.Name, "object": "model", "owned_by": "xorapi"})
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}
