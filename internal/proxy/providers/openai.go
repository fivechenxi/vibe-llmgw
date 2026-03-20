package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yourorg/llmgw/internal/domain"
)

// OpenAIProvider handles OpenAI-compatible APIs (OpenAI, DeepSeek, Alibaba Qwen, etc.)
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider creates a provider; proxyURL may be empty for direct connection.
func NewOpenAIProvider(apiKey, baseURL, proxyURL string) *OpenAIProvider {
	transport := &http.Transport{}
	if proxyURL != "" {
		if parsed, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}
	return &OpenAIProvider{
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Transport: transport, Timeout: 120 * time.Second},
	}
}

// ---- OpenAI wire types ----

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []domain.Message `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// ---- helpers ----

func (p *OpenAIProvider) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	return p.httpClient.Do(req)
}

// ---- Complete (non-streaming) ----

func (p *OpenAIProvider) Complete(ctx context.Context, userID string, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	payload := openAIRequest{
		Model:    req.Model,
		Messages: req.Messages,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var or openAIResponse
	if err := json.Unmarshal(raw, &or); err != nil {
		return nil, err
	}
	if or.Error != nil {
		return nil, fmt.Errorf("openai error %s: %s", or.Error.Type, or.Error.Message)
	}
	if len(or.Choices) == 0 {
		return nil, fmt.Errorf("openai returned empty choices")
	}

	usage := domain.TokenUsage{
		InputTokens:  or.Usage.PromptTokens,
		OutputTokens: or.Usage.CompletionTokens,
		TotalTokens:  or.Usage.TotalTokens,
	}
	return &domain.ChatResponse{Content: or.Choices[0].Message.Content, Usage: usage}, nil
}

// ---- Stream (SSE) ----

func (p *OpenAIProvider) Stream(c *gin.Context, userID string, req *domain.ChatRequest, q QuotaDeductor, logger ChatLogger) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	p.streamWithWriter(c.Request.Context(), userID, req, q, logger, func(chunk string) {
		c.SSEvent("", chunk)
		c.Writer.Flush()
	})
}

// streamWithWriter is the testable core of Stream.
func (p *OpenAIProvider) streamWithWriter(
	ctx context.Context,
	userID string,
	req *domain.ChatRequest,
	q QuotaDeductor,
	logger ChatLogger,
	onChunk func(string),
) {
	payload := openAIRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		onChunk("[ERROR] " + err.Error())
		return
	}

	resp, err := p.doRequest(ctx, body)
	if err != nil {
		onChunk("[ERROR] " + err.Error())
		return
	}
	defer resp.Body.Close()

	requestAt := time.Now()
	var fullContent strings.Builder
	var inputTokens, outputTokens int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			if data == "[DONE]" {
				onChunk("[DONE]")
			}
			continue
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Some providers (e.g. DeepSeek) send usage in the last chunk
		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) > 0 {
			text := chunk.Choices[0].Delta.Content
			if text != "" {
				fullContent.WriteString(text)
				onChunk(text)
			}
		}
	}

	// Fallback: if provider didn't return usage in stream, estimate from content length
	if inputTokens == 0 && outputTokens == 0 {
		outputTokens = len(strings.Fields(fullContent.String()))
	}
	total := inputTokens + outputTokens

	sessionID, _ := uuid.Parse(req.SessionID)
	reqMsgJSON, _ := json.Marshal(req.Messages)

	go func() {
		bgCtx := context.Background()
		_ = q.Deduct(bgCtx, userID, req.Model, total)
		_ = logger.Save(bgCtx, &domain.ChatLog{
			ID:              uuid.New(),
			UserID:          userID,
			SessionID:       sessionID,
			ModelID:         req.Model,
			RequestAt:       requestAt,
			ResponseAt:      time.Now(),
			RequestMessages: reqMsgJSON,
			ResponseContent: fullContent.String(),
			InputTokens:     inputTokens,
			OutputTokens:    outputTokens,
			Status:          "success",
		})
	}()
}
