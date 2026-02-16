package session

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)

	mgr := NewManager(store)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManagerCreate(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess, err := mgr.Create("Test Session")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sess == nil {
		t.Fatal("Create() returned nil session")
	}
	if sess.Name != "Test Session" {
		t.Errorf("expected Name 'Test Session', got %q", sess.Name)
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
}

func TestManagerCreateSetsCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess, _ := mgr.Create("Test")

	current := mgr.Current()
	if current == nil {
		t.Fatal("Current() returned nil after Create()")
	}
	if current.ID != sess.ID {
		t.Errorf("Current session ID mismatch")
	}
}

func TestManagerSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess1, _ := mgr.Create("Session 1")
	sess2, _ := mgr.Create("Session 2")

	// Currently on sess2
	if mgr.Current().ID != sess2.ID {
		t.Error("expected current to be sess2")
	}

	// Switch to sess1
	if err := mgr.Switch(sess1.ID); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	if mgr.Current().ID != sess1.ID {
		t.Error("expected current to be sess1 after switch")
	}
}

func TestManagerSwitchNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	err := mgr.Switch("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestManagerList(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	// Empty list
	list, err := mgr.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}

	// Create sessions
	mgr.Create("Session 1")
	mgr.Create("Session 2")

	list, _ = mgr.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(list))
	}
}

func TestManagerDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess, _ := mgr.Create("Test")
	id := sess.ID

	if err := mgr.Delete(id); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	list, _ := mgr.List()
	if len(list) != 0 {
		t.Error("session should be deleted")
	}
}

func TestManagerDeleteCurrentSession(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess, _ := mgr.Create("Test")

	if err := mgr.Delete(sess.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Current should be nil after deleting current session
	if mgr.Current() != nil {
		t.Error("current session should be nil after deletion")
	}
}

func TestManagerSaveCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	sess, _ := mgr.Create("Test")

	// Add message to current
	sess.AddMessage(anthropic.NewUserMessage(anthropic.NewTextBlock("Hello")))

	// Save
	if err := mgr.SaveCurrent(); err != nil {
		t.Fatalf("SaveCurrent() error = %v", err)
	}

	// Reload and verify
	loaded, _ := store.Load(sess.ID)
	if len(loaded.Messages) != 1 {
		t.Error("message was not saved")
	}
}

func TestManagerSaveCurrentNil(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	// Should not error when no current session
	if err := mgr.SaveCurrent(); err != nil {
		t.Errorf("SaveCurrent() should not error with nil current: %v", err)
	}
}

func TestManagerGetOrCreate(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := NewFileStore(tmpDir)
	mgr := NewManager(store)

	// First call should create
	sess1, err := mgr.GetOrCreate("default")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if sess1 == nil {
		t.Fatal("GetOrCreate() returned nil")
	}

	// Second call with same name should get existing
	sess2, err := mgr.GetOrCreate("default")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if sess2.ID != sess1.ID {
		t.Error("GetOrCreate should return existing session")
	}
}
