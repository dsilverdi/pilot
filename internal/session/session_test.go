package session

import (
	"sync"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewSession(t *testing.T) {
	sess := New("test-id", "Test Session")

	if sess.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", sess.ID)
	}
	if sess.Name != "Test Session" {
		t.Errorf("expected Name 'Test Session', got %q", sess.Name)
	}
	if len(sess.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(sess.Messages))
	}
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestSessionAddMessage(t *testing.T) {
	sess := New("test-id", "Test")

	msg := anthropic.NewUserMessage(anthropic.NewTextBlock("Hello"))
	sess.AddMessage(msg)

	if len(sess.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(sess.Messages))
	}
}

func TestSessionAddMultipleMessages(t *testing.T) {
	sess := New("test-id", "Test")

	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("World")))

	if len(sess.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(sess.Messages))
	}
}

func TestSessionGetMessages(t *testing.T) {
	sess := New("test-id", "Test")

	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	msgs := sess.GetMessages()

	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}

	// Verify it returns a copy (modifying returned slice shouldn't affect original)
	msgs = append(msgs, anthropic.NewUserMessage(anthropic.NewTextBlock("Extra")))
	if len(sess.Messages) != 1 {
		t.Error("GetMessages should return a copy, not the original slice")
	}
}

func TestSessionClear(t *testing.T) {
	sess := New("test-id", "Test")

	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))
	sess.Clear()

	if len(sess.Messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(sess.Messages))
	}
}

func TestSessionUpdatedAtChanges(t *testing.T) {
	sess := New("test-id", "Test")
	originalUpdatedAt := sess.UpdatedAt

	time.Sleep(10 * time.Millisecond) // Ensure time passes

	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	if !sess.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated after adding message")
	}
}

func TestSessionThreadSafety(t *testing.T) {
	sess := New("test-id", "Test")

	var wg sync.WaitGroup
	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("msg")))
		}()
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sess.GetMessages()
		}()
	}

	wg.Wait()

	// Should have exactly 100 messages
	if len(sess.Messages) != 100 {
		t.Errorf("expected 100 messages, got %d", len(sess.Messages))
	}
}

func TestSessionMessageCount(t *testing.T) {
	sess := New("test-id", "Test")

	if sess.MessageCount() != 0 {
		t.Errorf("expected 0, got %d", sess.MessageCount())
	}

	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	if sess.MessageCount() != 1 {
		t.Errorf("expected 1, got %d", sess.MessageCount())
	}
}
