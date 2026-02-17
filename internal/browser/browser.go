package browser

import (
	"context"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	// DefaultTimeout for page operations
	DefaultTimeout = 30 * time.Second

	// DefaultUserAgent for browser requests
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

// Config holds browser configuration
type Config struct {
	Headless  bool          // Run browser in headless mode
	Timeout   time.Duration // Default timeout for operations
	UserAgent string        // User agent string
}

// DefaultConfig returns default browser configuration
func DefaultConfig() Config {
	return Config{
		Headless:  true,
		Timeout:   DefaultTimeout,
		UserAgent: DefaultUserAgent,
	}
}

// Manager manages browser lifecycle with lazy initialization
type Manager struct {
	browser *rod.Browser
	mu      sync.RWMutex
	config  Config
}

// NewManager creates a new browser manager
func NewManager(cfg Config) *Manager {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	return &Manager{
		config: cfg,
	}
}

// getBrowser returns the browser instance, starting it if needed
func (m *Manager) getBrowser() (*rod.Browser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser != nil {
		return m.browser, nil
	}

	// Use launcher for managed browser with auto-download
	l := launcher.New().
		Headless(m.config.Headless).
		Set("disable-gpu", "true").
		Set("no-sandbox", "true").
		Set("disable-dev-shm-usage", "true").
		Set("ignore-certificate-errors", "true").
		Set("allow-insecure-localhost", "true")

	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return nil, err
	}

	m.browser = browser
	return m.browser, nil
}

// NewPage creates a new browser page with configured settings
func (m *Manager) NewPage(ctx context.Context) (*rod.Page, error) {
	browser, err := m.getBrowser()
	if err != nil {
		return nil, err
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}

	// Set user agent
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: m.config.UserAgent,
	}); err != nil {
		page.Close()
		return nil, err
	}

	// Set timeout
	page = page.Timeout(m.config.Timeout)

	// Apply context for cancellation
	page = page.Context(ctx)

	return page, nil
}

// Close shuts down the browser
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser != nil {
		err := m.browser.Close()
		m.browser = nil
		return err
	}
	return nil
}

// IsRunning returns true if the browser is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.browser != nil
}
