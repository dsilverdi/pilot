package session

import (
	"context"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// mockCompactionClient implements CompactionClient for testing
type mockCompactionClient struct {
	response *anthropic.Message
	err      error
}

func (m *mockCompactionClient) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestNewCompactor(t *testing.T) {
	client := &mockCompactionClient{}
	compactor := NewCompactor(client, anthropic.ModelClaudeSonnet4_5_20250929, 0)

	if compactor == nil {
		t.Fatal("NewCompactor returned nil")
	}

	// Default threshold should be used when 0 is passed
	if compactor.threshold != DefaultTokenThreshold {
		t.Errorf("expected default threshold %d, got %d", DefaultTokenThreshold, compactor.threshold)
	}
}

func TestNewCompactorCustomThreshold(t *testing.T) {
	client := &mockCompactionClient{}
	customThreshold := 50000
	compactor := NewCompactor(client, anthropic.ModelClaudeSonnet4_5_20250929, customThreshold)

	if compactor.threshold != customThreshold {
		t.Errorf("expected threshold %d, got %d", customThreshold, compactor.threshold)
	}
}

func TestShouldCompact(t *testing.T) {
	client := &mockCompactionClient{}
	compactor := NewCompactor(client, anthropic.ModelClaudeSonnet4_5_20250929, 100) // Low threshold for testing

	sess := New("test-id", "test-session")

	// Empty session should not need compaction
	if compactor.ShouldCompact(sess) {
		t.Error("empty session should not need compaction")
	}

	// Add some messages to exceed threshold
	for i := 0; i < 20; i++ {
		sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("This is a test message that should contribute to the token count")))
	}

	if !compactor.ShouldCompact(sess) {
		t.Error("session with many messages should need compaction")
	}
}

func TestCompactNotEnoughMessages(t *testing.T) {
	client := &mockCompactionClient{}
	compactor := NewCompactor(client, anthropic.ModelClaudeSonnet4_5_20250929, 100)

	sess := New("test-id", "test-session")

	// Add fewer messages than RecentMessagesToKeep
	for i := 0; i < RecentMessagesToKeep-1; i++ {
		sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Test message")))
	}

	err := compactor.Compact(context.Background(), sess)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should not compact since not enough messages
	if sess.Compaction != nil {
		t.Error("should not have compacted with few messages")
	}
}

func TestCompact(t *testing.T) {
	// Mock response with summary
	mockResponse := &anthropic.Message{
		Content: []anthropic.ContentBlockUnion{
			anthropic.ContentBlockUnion{
				Type: "text",
			},
		},
	}

	client := &mockCompactionClient{response: mockResponse}
	compactor := NewCompactor(client, anthropic.ModelClaudeSonnet4_5_20250929, 100)

	sess := New("test-id", "test-session")

	// Add more messages than RecentMessagesToKeep
	totalMessages := RecentMessagesToKeep + 5
	for i := 0; i < totalMessages; i++ {
		sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Test message")))
	}

	err := compactor.Compact(context.Background(), sess)
	if err != nil {
		t.Fatalf("Compact() error = %v", err)
	}

	// Should have kept only recent messages
	if len(sess.Messages) != RecentMessagesToKeep {
		t.Errorf("expected %d messages after compaction, got %d", RecentMessagesToKeep, len(sess.Messages))
	}

	// Should have compaction info
	if sess.Compaction == nil {
		t.Fatal("compaction should be set")
	}

	if sess.Compaction.CompactedCount != 5 {
		t.Errorf("expected 5 compacted messages, got %d", sess.Compaction.CompactedCount)
	}
}

func TestSessionEstimateTokens(t *testing.T) {
	sess := New("test-id", "test-session")

	// Empty session should have 0 tokens
	if tokens := sess.EstimateTokens(); tokens != 0 {
		t.Errorf("expected 0 tokens for empty session, got %d", tokens)
	}

	// Add a message
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello world")))

	tokens := sess.EstimateTokens()
	if tokens == 0 {
		t.Error("session with message should have non-zero token count")
	}
}

func TestSessionNeedsCompaction(t *testing.T) {
	sess := New("test-id", "test-session")

	// Empty session should not need compaction
	if sess.NeedsCompaction(100) {
		t.Error("empty session should not need compaction")
	}

	// Add messages to exceed threshold
	for i := 0; i < 50; i++ {
		sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("This is a longer test message")))
	}

	// With low threshold, should need compaction
	if !sess.NeedsCompaction(100) {
		t.Error("session should need compaction with low threshold")
	}

	// With high threshold, should not need compaction
	if sess.NeedsCompaction(1000000) {
		t.Error("session should not need compaction with high threshold")
	}
}

func TestSessionHasCompaction(t *testing.T) {
	sess := New("test-id", "test-session")

	if sess.HasCompaction() {
		t.Error("new session should not have compaction")
	}

	sess.SetCompaction("Test summary", 5)

	if !sess.HasCompaction() {
		t.Error("session should have compaction after SetCompaction")
	}
}

func TestSessionGetMessagesWithContext(t *testing.T) {
	sess := New("test-id", "test-session")
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	// Without compaction, should return messages as-is
	msgs := sess.GetMessagesWithContext()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}

	// With compaction, should prepend summary
	sess.SetCompaction("Previous conversation summary", 5)

	msgs = sess.GetMessagesWithContext()
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages (summary + original), got %d", len(msgs))
	}
}
