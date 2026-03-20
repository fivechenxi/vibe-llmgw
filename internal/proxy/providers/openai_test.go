package providers

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yourorg/llmgw/internal/domain"
)

// newOpenAIForTest reads OPENAI_API_KEY and HTTP_PROXY from env.
// Tests are skipped if OPENAI_API_KEY is not set.
func newOpenAIForTest(t *testing.T) *OpenAIProvider {
	t.Helper()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}
	return NewOpenAIProvider(apiKey, "https://api.openai.com/v1", os.Getenv("HTTP_PROXY"))
}

func TestOpenAIComplete(t *testing.T) {
	p := newOpenAIForTest(t)

	req := &domain.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []domain.Message{
			{Role: "user", Content: "reply: OK"},
		},
	}

	resp, err := p.Complete(context.Background(), "test-user", req)
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected non-empty content")
	}
	if resp.Usage.InputTokens == 0 {
		t.Error("expected non-zero input_tokens")
	}
	if resp.Usage.OutputTokens == 0 {
		t.Error("expected non-zero output_tokens")
	}
	if resp.Usage.TotalTokens != resp.Usage.InputTokens+resp.Usage.OutputTokens {
		t.Errorf("total_tokens mismatch: %d != %d+%d",
			resp.Usage.TotalTokens, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	}
	t.Logf("content: %q", resp.Content)
	t.Logf("usage:   in=%d out=%d total=%d",
		resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
}

func TestOpenAIStream(t *testing.T) {
	p := newOpenAIForTest(t)

	req := &domain.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []domain.Message{
			{Role: "user", Content: "1+1=?"},
		},
		SessionID: "00000000-0000-0000-0000-000000000002",
	}

	var buf strings.Builder
	var doneSeen bool

	q := &mockQuota{}
	log := &mockLogger{}

	p.streamWithWriter(
		context.Background(),
		"test-user",
		req,
		q,
		log,
		func(chunk string) {
			if chunk == "[DONE]" {
				doneSeen = true
				return
			}
			buf.WriteString(chunk)
		},
	)

	time.Sleep(300 * time.Millisecond)

	if buf.Len() == 0 {
		t.Fatal("expected streamed content, got none")
	}
	if !doneSeen {
		t.Error("expected [DONE] sentinel")
	}
	if log.saved == nil {
		t.Fatal("expected chat log to be saved")
	}
	if log.saved.ResponseContent == "" {
		t.Error("expected non-empty ResponseContent in log")
	}

	t.Logf("streamed content: %q", buf.String())
	t.Logf("usage: in=%d out=%d", log.saved.InputTokens, log.saved.OutputTokens)
}
