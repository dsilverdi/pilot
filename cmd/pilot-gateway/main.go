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
	"github.com/dsilverdi/pilot/internal/channel/telegram"
	"github.com/dsilverdi/pilot/internal/gateway"
	"github.com/dsilverdi/pilot/internal/gateway/apikey"
	"github.com/dsilverdi/pilot/internal/session"
	"github.com/dsilverdi/pilot/internal/skills"
	"github.com/dsilverdi/pilot/internal/tools"
)

var (
	version = "dev"
)

// Default personality if PILOT.md is not found
const defaultPersonality = `# Bread

You are Bread, a helpful and joyful AI assistant!

## Personality
- Warm, friendly, and a bit playful
- Keep things light and fun
- Use casual language, like chatting with a friend
- Sprinkle in emojis naturally (but don't overdo it)
- Be honest when you don't know something

## Communication Style
- Keep responses concise - this is chat, not an essay
- Avoid technical jargon and tool names
- Focus on results, not process
- Break up long text into digestible chunks

## When Using Tools
- Don't mention tool names or technical processes
- Describe what you're doing naturally: "Let me search for that..." or "Checking the web..."
- Focus on delivering results, not explaining the machinery

## Formatting
- Use **bold** for emphasis
- Use bullet points for lists
- No markdown headers (## or ###) in responses
- Keep paragraphs short and scannable
`

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

	// Build system prompt for HTTP API (technical/formal)
	systemPrompt := buildSystemPrompt(skillLoaders)

	// Initialize agent configuration for HTTP API
	config := &agent.Config{
		Model:        anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens:    4096,
		Temperature:  0.7,
		SystemPrompt: systemPrompt,
	}

	// Create agent for HTTP API
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

	// Initialize Telegram bot if configured
	tgConfig := telegram.LoadConfig()
	var tgBot *telegram.Bot
	if tgConfig.Enabled() {
		// Load personality from PILOT.md for Telegram
		personality := loadPilotPersonality(pilotDir, *workDir)

		// Build Telegram-specific system prompt with personality
		tgSystemPrompt := buildTelegramSystemPrompt(personality, skillLoaders)

		// Create separate agent for Telegram with friendly personality
		tgAgentConfig := &agent.Config{
			Model:        anthropic.ModelClaudeSonnet4_5_20250929,
			MaxTokens:    4096,
			Temperature:  0.8, // Slightly higher for more creative/friendly responses
			SystemPrompt: tgSystemPrompt,
		}

		tgAgent, err := agent.New(tgAgentConfig, registry)
		if err != nil {
			return fmt.Errorf("failed to create telegram agent: %w", err)
		}

		tgBot, err = telegram.NewBot(tgConfig, tgAgent, sessionManager)
		if err != nil {
			return fmt.Errorf("failed to create telegram bot: %w", err)
		}
		log.Printf("Telegram bot enabled: @%s", tgBot.Username())
	}

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

	// Start Telegram bot in background (if enabled)
	if tgBot != nil {
		go func() {
			if err := tgBot.Start(ctx); err != nil {
				log.Printf("Telegram bot error: %v", err)
			}
		}()
	}

	// Start HTTP server (blocking)
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
	fmt.Println("Telegram Bot:")
	fmt.Println("  Set TELEGRAM_BOT_TOKEN to enable Telegram bot")
	fmt.Println("  Optionally set TELEGRAM_ALLOWED_USERS to restrict access")
	fmt.Println("  Customize personality via PILOT.md in PILOT_HOME or working directory")
	fmt.Println()
}

func getPilotDir() (string, error) {
	// Check PILOT_HOME first (for systemd/production deployments)
	if pilotHome := os.Getenv("PILOT_HOME"); pilotHome != "" {
		return pilotHome, nil
	}
	// Fall back to ~/.pilot
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

// loadPilotPersonality loads the personality from PILOT.md
// Priority: $PILOT_HOME/PILOT.md > ./PILOT.md > default
func loadPilotPersonality(pilotDir, workDir string) string {
	// Try PILOT_HOME first
	pilotMdPath := filepath.Join(pilotDir, "PILOT.md")
	if content, err := os.ReadFile(pilotMdPath); err == nil {
		log.Printf("Loaded personality from %s", pilotMdPath)
		return string(content)
	}

	// Try working directory
	pilotMdPath = filepath.Join(workDir, "PILOT.md")
	if content, err := os.ReadFile(pilotMdPath); err == nil {
		log.Printf("Loaded personality from %s", pilotMdPath)
		return string(content)
	}

	// Fall back to default
	log.Println("Using default personality (PILOT.md not found)")
	return defaultPersonality
}

// buildTelegramSystemPrompt creates a system prompt for Telegram with personality
func buildTelegramSystemPrompt(personality string, skillLoaders []*skills.Loader) string {
	// Start with personality
	prompt := personality

	// Add available capabilities (without technical details)
	prompt += `

## What You Can Do
You have access to helpful tools that let you:
- Search the web for current information
- Read and fetch content from websites
- Read and write files
- Run commands and scripts`

	// Add skills if available
	totalSkills := 0
	for _, loader := range skillLoaders {
		totalSkills += loader.Count()
	}
	if totalSkills > 0 {
		prompt += `
- Use specialized skills for documents, research, and more`
	}

	prompt += `

Remember: When using these capabilities, don't mention technical details.
Just naturally describe what you're doing and focus on helping the user!`

	return prompt
}

// buildSystemPrompt creates the technical system prompt for HTTP API
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
