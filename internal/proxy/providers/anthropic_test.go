package providers

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yourorg/llmgw/internal/domain"
)

// newAnthropicForTest returns a provider using ANTHROPIC_API_KEY and HTTP_PROXY env vars.
// Tests are skipped if ANTHROPIC_API_KEY is not set.
func newAnthropicForTest(t *testing.T) *AnthropicProvider {
	t.Helper()
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}
	return NewAnthropicProvider(apiKey, os.Getenv("HTTP_PROXY"))
}

// TestAnthropicComplete tests the non-streaming path.
func TestAnthropicComplete(t *testing.T) {
	p := newAnthropicForTest(t)

	req := &domain.ChatRequest{
		Model: "claude-haiku-4-5",
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
	t.Logf("content:  %q", resp.Content)
	t.Logf("usage:    in=%d out=%d total=%d",
		resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
}

// TestAnthropicCompleteWithSystem verifies that a system message is correctly
// extracted and sent as the top-level "system" field.
func TestAnthropicCompleteWithSystem(t *testing.T) {
	p := newAnthropicForTest(t)

	req := &domain.ChatRequest{
		Model: "claude-haiku-4-5",
		Messages: []domain.Message{
			{Role: "system", Content: "Reply in Chinese."},
			{Role: "user", Content: "hi"},
		},
	}

	resp, err := p.Complete(context.Background(), "test-user", req)
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected non-empty content")
	}
	t.Logf("content: %q", resp.Content)
}

// TestAnthropicStream tests the streaming path via streamWithWriter.
func TestAnthropicStream(t *testing.T) {
	p := newAnthropicForTest(t)

	req := &domain.ChatRequest{
		Model: "claude-haiku-4-5",
		Messages: []domain.Message{
			{Role: "user", Content: "1+1=?"},
		},
		SessionID: "00000000-0000-0000-0000-000000000001",
	}

	var mu strings.Builder
	var doneSeen bool

	q := &mockQuota{}
	log := &mockLogger{}

	p.streamWithWriter(
		context.Background(),
		&bytes.Buffer{},
		"test-user",
		req,
		q,
		log,
		func(chunk string) {
			if chunk == "[DONE]" {
				doneSeen = true
				return
			}
			mu.WriteString(chunk)
		},
	)

	// Give the async goroutine time to finish.
	time.Sleep(300 * time.Millisecond)

	if mu.Len() == 0 {
		t.Fatal("expected streamed content, got none")
	}
	if !doneSeen {
		t.Error("expected [DONE] sentinel")
	}
	if q.deducted == 0 {
		t.Error("expected quota to be deducted")
	}
	if log.saved == nil {
		t.Fatal("expected chat log to be saved")
	}
	if log.saved.ResponseContent == "" {
		t.Error("expected non-empty ResponseContent in log")
	}

	t.Logf("streamed content: %q", mu.String())
	t.Logf("usage: in=%d out=%d", log.saved.InputTokens, log.saved.OutputTokens)
}

// ---- test doubles ----

type mockQuota struct{ deducted int }

func (m *mockQuota) Deduct(_ context.Context, _, _ string, tokens int) error {
	m.deducted += tokens
	return nil
}

type mockLogger struct{ saved *domain.ChatLog }

func (m *mockLogger) Save(_ context.Context, l *domain.ChatLog) error {
	m.saved = l
	return nil
}
