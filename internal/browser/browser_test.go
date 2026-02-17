package browser

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Headless {
		t.Error("default config should be headless")
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultTimeout, cfg.Timeout)
	}
	if cfg.UserAgent == "" {
		t.Error("default config should have user agent")
	}
}

func TestNewManager(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.IsRunning() {
		t.Error("new manager should not have running browser")
	}
}

func TestNewManagerDefaultTimeout(t *testing.T) {
	cfg := Config{
		Headless: true,
		Timeout:  0, // Should be set to default
	}
	mgr := NewManager(cfg)

	if mgr.config.Timeout != DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultTimeout, mgr.config.Timeout)
	}
}

func TestNewManagerDefaultUserAgent(t *testing.T) {
	cfg := Config{
		Headless:  true,
		Timeout:   30 * time.Second,
		UserAgent: "", // Should be set to default
	}
	mgr := NewManager(cfg)

	if mgr.config.UserAgent != DefaultUserAgent {
		t.Errorf("expected user agent %q, got %q", DefaultUserAgent, mgr.config.UserAgent)
	}
}

func TestManagerCloseWithoutBrowser(t *testing.T) {
	cfg := DefaultConfig()
	mgr := NewManager(cfg)

	// Should not error when closing without browser
	if err := mgr.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
