package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/dsilverdi/pilot/internal/browser"
	"github.com/dsilverdi/pilot/internal/gateway"
	"github.com/dsilverdi/pilot/internal/gateway/apikey"
	"github.com/dsilverdi/pilot/internal/session"
	"github.com/dsilverdi/pilot/internal/skills"
	"github.com/dsilverdi/pilot/internal/tools"
)

var (
	version = "dev"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse command line arguments
	addr := flag.String("addr", ":8080", "Listen address")
	workDir := flag.String("workdir", "", "Working directory for tools (default: current directory)")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *showHelp {
		printUsage()
		return nil
	}

	if *showVersion {
		fmt.Printf("pilot-gateway version %s\n", version)
		return nil
	}

	// Get working directory
	if *workDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		*workDir = cwd
	}

	// Get and ensure ~/.pilot directory
	pilotDir, err := getPilotDir()
	if err != nil {
		return fmt.Errorf("failed to get pilot directory: %w", err)
	}
	if err := ensurePilotDirs(pilotDir); err != nil {
		return fmt.Errorf("failed to create pilot directories: %w", err)
	}

	// Initialize API key manager
	keyManager := apikey.NewManager(pilotDir)

	// Initialize session manager
	sessionsDir := filepath.Join(pilotDir, "sessions")
	sessionStore, err := session.NewFileStore(sessionsDir)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}
	sessionManager := session.NewManager(sessionStore, sessionsDir)

	// Initialize skill loaders
	var skillLoaders []*skills.Loader

	// Global skills from ~/.pilot/skills
	globalSkillLoader := skills.NewLoader(filepath.Join(pilotDir, "skills"))
	if err := globalSkillLoader.LoadAll(); err != nil {
		log.Printf("Warning: failed to load global skills: %v", err)
	}
	skillLoaders = append(skillLoaders, globalSkillLoader)

	// Local skills from ./skills in working directory
	localSkillDir := filepath.Join(*workDir, "skills")
	if _, err := os.Stat(localSkillDir); err == nil {
		localSkillLoader := skills.NewLoader(localSkillDir)
		if err := localSkillLoader.LoadAll(); err != nil {
			log.Printf("Warning: failed to load local skills: %v", err)
		}
		skillLoaders = append(skillLoaders, localSkillLoader)
	}

	// Initialize browser manager
	browserCfg := browser.DefaultConfig()
	browserMgr := browser.NewManager(browserCfg)

	// Initialize tool registry
	registry := tools.NewRegistry()
	registry.Register(tools.NewFileReadTool(*workDir))
	registry.Register(tools.NewFileWriteTool(*workDir))
	registry.Register(tools.NewWebSearchTool())
	registry.Register(tools.NewWebFetchTool(browserMgr))

	// Add invoke_skill tool if skills are loaded
	if len(skillLoaders) > 0 {
		registry.Register(tools.NewInvokeSkillTool(skillLoaders))
	}

	// Add bash_exec tool
	fallbackDir := filepath.Join(pilotDir, "tmp")
	registry.Register(tools.NewBashExecTool(*workDir, sessionManager, fallbackDir))

	// Build system prompt
	systemPrompt := buildSystemPrompt(skillLoaders)

	// Initialize agent configuration
	config := &agent.Config{
		Model:        anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens:    4096,
		Temperature:  0.7,
		SystemPrompt: systemPrompt,
	}

	// Create agent
	ag, err := agent.New(config, registry)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Create gateway config
	gwConfig := gateway.DefaultConfig()
	gwConfig.Addr = *addr
	gwConfig.Version = version

	// Create gateway server
	srv := gateway.NewServer(ag, sessionManager, registry, keyManager, gwConfig)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer browserMgr.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Received shutdown signal...")
		srv.Shutdown(ctx)
		browserMgr.Close()
		cancel()
	}()

	// Start server
	return srv.Start()
}

func printUsage() {
	fmt.Println("pilot-gateway - HTTP service for pilot agent")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pilot-gateway [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --addr <address>    Listen address (default: :8080)")
	fmt.Println("  --workdir <path>    Working directory for tools")
	fmt.Println("  --version           Show version")
	fmt.Println("  --help              Show this help")
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("  GET  /health           Health check")
	fmt.Println("  POST /chat             Send a message")
	fmt.Println("  POST /chat/stream      Send a message (streaming)")
	fmt.Println("  DELETE /session/{id}   Delete a session")
	fmt.Println()
	fmt.Println("Authentication:")
	fmt.Println("  Generate API keys using: pilot api-key generate --name <name>")
	fmt.Println("  Include key in requests: X-API-Key: psk_...")
	fmt.Println()
}

func getPilotDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pilot"), nil
}

func ensurePilotDirs(pilotDir string) error {
	dirs := []string{
		filepath.Join(pilotDir, "sessions"),
		filepath.Join(pilotDir, "skills"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func buildSystemPrompt(skillLoaders []*skills.Loader) string {
	systemPrompt := `You are a helpful AI assistant with access to tools.

Available tools:
- file_read: Read the contents of files
- file_write: Write content to files
- bash_exec: Execute bash commands (run scripts, install dependencies, etc.)
- web_search: Search the web using SearXNG (aggregates Google, Brave, Startpage, and more)
- web_fetch: Fetch and extract content from a specific web page URL`

	// Add skills info if available
	totalSkills := 0
	for _, loader := range skillLoaders {
		totalSkills += loader.Count()
	}
	if totalSkills > 0 {
		systemPrompt += fmt.Sprintf(`
- invoke_skill: Invoke specialized skills for complex tasks (%d skills available)

IMPORTANT: When the user's request matches a skill (like creating documents, presentations, spreadsheets, or research tasks):
1. ALWAYS use invoke_skill FIRST to get detailed instructions
2. Follow the skill instructions to create necessary scripts
3. Use bash_exec to install dependencies if needed and run the scripts`, totalSkills)
	}

	systemPrompt += `

Be concise but thorough. When using tools, explain what you're doing.`

	return systemPrompt
}
