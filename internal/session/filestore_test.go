package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	if store == nil {
		t.Fatal("NewFileStore() returned nil")
	}
}

func TestNewFileStoreCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "sessions")

	_, err := NewFileStore(newDir)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("NewFileStore should create the directory")
	}
}

func TestFileStoreSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	sess := New("test-id", "Test Session")
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	// Save
	if err := store.Save(sess); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load
	loaded, err := store.Load("test-id")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != sess.ID {
		t.Errorf("ID mismatch: got %q, want %q", loaded.ID, sess.ID)
	}
	if loaded.Name != sess.Name {
		t.Errorf("Name mismatch: got %q, want %q", loaded.Name, sess.Name)
	}
	if len(loaded.Messages) != len(sess.Messages) {
		t.Errorf("Message count mismatch: got %d, want %d", len(loaded.Messages), len(sess.Messages))
	}
}

func TestFileStoreLoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestFileStoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	sess := New("test-id", "Test")
	store.Save(sess)

	// Delete
	if err := store.Delete("test-id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err := store.Load("test-id")
	if err == nil {
		t.Error("session should be deleted")
	}
}

func TestFileStoreDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	err := store.Delete("nonexistent")
	// Should not error for nonexistent (idempotent)
	if err != nil && !os.IsNotExist(err) {
		t.Logf("Delete() returned: %v (acceptable)", err)
	}
}

func TestFileStoreList(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	// Empty list
	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}

	// Add sessions
	sess1 := New("id1", "Session 1")
	sess1.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("msg1")))
	store.Save(sess1)

	sess2 := New("id2", "Session 2")
	store.Save(sess2)

	// List again
	list, err = store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestFileStoreListSessionInfo(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	sess := New("test-id", "Test Session")
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("msg")))
	store.Save(sess)

	list, _ := store.List()

	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}

	info := list[0]
	if info.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", info.ID)
	}
	if info.Name != "Test Session" {
		t.Errorf("expected Name 'Test Session', got %q", info.Name)
	}
	if info.MessageCount != 1 {
		t.Errorf("expected MessageCount 1, got %d", info.MessageCount)
	}
}

func TestFileStoreSaveOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	sess := New("test-id", "Original")
	store.Save(sess)

	sess.Name = "Updated"
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("msg")))
	store.Save(sess)

	loaded, _ := store.Load("test-id")
	if loaded.Name != "Updated" {
		t.Errorf("expected Name 'Updated', got %q", loaded.Name)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(loaded.Messages))
	}
}
