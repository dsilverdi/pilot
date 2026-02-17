package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/dsilverdi/pilot/internal/browser"
	"github.com/dsilverdi/pilot/internal/cli"
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

// parseArgs parses command line arguments and returns the prompt if -p flag is used
func parseArgs() (prompt string, showVersion bool, showHelp bool) {
	flag.StringVar(&prompt, "p", "", "Execute a prompt directly without entering interactive mode")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.Parse()

	// If -p flag is used but empty, collect remaining args as the prompt
	if prompt == "" && flag.NArg() > 0 {
		// Check if first non-flag arg follows -p pattern
		args := flag.Args()
		prompt = strings.Join(args, " ")
	}

	return prompt, showVersion, showHelp
}

func printUsage() {
	fmt.Println("Pilot - A lightweight agentic CLI system")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pilot                    Start interactive mode")
	fmt.Println("  pilot -p <prompt>        Execute a prompt directly")
	fmt.Println("  pilot -v, --version      Show version")
	fmt.Println("  pilot -h, --help         Show this help")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  pilot api-key generate --name <name>    Generate a new API key")
	fmt.Println("  pilot api-key list                      List all API keys")
	fmt.Println("  pilot api-key revoke --name <name>      Revoke an API key")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pilot -p \"What is Go?\"")
	fmt.Println("  pilot -p \"Read the file main.go and explain it\"")
	fmt.Println("  pilot api-key generate --name telegram-bot")
	fmt.Println()
}

// handleAPIKeyCommand handles the api-key subcommand
func handleAPIKeyCommand(args []string) error {
	if len(args) == 0 {
		printAPIKeyUsage()
		return nil
	}

	pilotDir, err := getPilotDir()
	if err != nil {
		return fmt.Errorf("failed to get pilot directory: %w", err)
	}

	keyManager := apikey.NewManager(pilotDir)

	switch args[0] {
	case "generate":
		return handleAPIKeyGenerate(keyManager, args[1:])
	case "list":
		return handleAPIKeyList(keyManager)
	case "revoke":
		return handleAPIKeyRevoke(keyManager, args[1:])
	case "help", "--help", "-h":
		printAPIKeyUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "Unknown api-key command: %s\n", args[0])
		printAPIKeyUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printAPIKeyUsage() {
	fmt.Println("API Key Management")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pilot api-key generate --name <name>    Generate a new API key")
	fmt.Println("  pilot api-key list                      List all API keys")
	fmt.Println("  pilot api-key revoke --name <name>      Revoke an API key")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pilot api-key generate --name telegram-bot")
	fmt.Println("  pilot api-key list")
	fmt.Println("  pilot api-key revoke --name telegram-bot")
	fmt.Println()
}

func handleAPIKeyGenerate(keyManager *apikey.Manager, args []string) error {
	fs := flag.NewFlagSet("api-key generate", flag.ExitOnError)
	name := fs.String("name", "", "Name for the API key")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	key, err := keyManager.Generate(*name)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	fmt.Println()
	fmt.Println("Generated API key:")
	fmt.Printf("  %s\n", key)
	fmt.Println()
	fmt.Printf("Name: %s\n", *name)
	fmt.Println()
	fmt.Println("Store this key securely - it won't be shown again.")
	fmt.Println("Use it with pilot-gateway via the X-API-Key header.")
	fmt.Println()

	return nil
}

func handleAPIKeyList(keyManager *apikey.Manager) error {
	keys, err := keyManager.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		fmt.Println("No API keys configured.")
		fmt.Println()
		fmt.Println("Generate one with: pilot api-key generate --name <name>")
		return nil
	}

	fmt.Println()
	fmt.Printf("%-20s %-25s %s\n", "NAME", "CREATED", "KEY HASH")
	fmt.Printf("%-20s %-25s %s\n", "----", "-------", "--------")
	for _, k := range keys {
		// Show truncated hash
		hash := k.KeyHash
		if len(hash) > 20 {
			hash = hash[:20] + "..."
		}
		fmt.Printf("%-20s %-25s %s\n", k.Name, k.CreatedAt.Format("2006-01-02 15:04:05"), hash)
	}
	fmt.Println()

	return nil
}

func handleAPIKeyRevoke(keyManager *apikey.Manager, args []string) error {
	fs := flag.NewFlagSet("api-key revoke", flag.ExitOnError)
	name := fs.String("name", "", "Name of the API key to revoke")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	if err := keyManager.Revoke(*name); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	fmt.Printf("API key '%s' has been revoked.\n", *name)
	return nil
}

// getPilotDir returns the path to ~/.pilot directory
func getPilotDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pilot"), nil
}

// ensurePilotDirs creates the ~/.pilot directory structure
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

func run() error {
	// Handle subcommands before parsing flags
	if len(os.Args) > 1 && os.Args[1] == "api-key" {
		return handleAPIKeyCommand(os.Args[2:])
	}

	// Parse command line arguments
	prompt, showVersion, showHelp := parseArgs()

	if showHelp {
		printUsage()
		return nil
	}

	if showVersion {
		fmt.Printf("pilot version %s\n", version)
		return nil
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get and ensure ~/.pilot directory
	pilotDir, err := getPilotDir()
	if err != nil {
		return fmt.Errorf("failed to get pilot directory: %w", err)
	}
	if err := ensurePilotDirs(pilotDir); err != nil {
		return fmt.Errorf("failed to create pilot directories: %w", err)
	}

	// Initialize session manager
	sessionsDir := filepath.Join(pilotDir, "sessions")
	sessionStore, err := session.NewFileStore(sessionsDir)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}
	sessionManager := session.NewManager(sessionStore, sessionsDir)

	// Initialize skill loaders (both ~/.pilot/skills and local ./skills)
	var skillLoaders []*skills.Loader

	// Global skills from ~/.pilot/skills
	globalSkillLoader := skills.NewLoader(filepath.Join(pilotDir, "skills"))
	if err := globalSkillLoader.LoadAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load global skills: %v\n", err)
	}
	skillLoaders = append(skillLoaders, globalSkillLoader)

	// Local skills from ./skills in working directory
	localSkillDir := filepath.Join(workDir, "skills")
	if _, err := os.Stat(localSkillDir); err == nil {
		localSkillLoader := skills.NewLoader(localSkillDir)
		if err := localSkillLoader.LoadAll(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load local skills: %v\n", err)
		}
		skillLoaders = append(skillLoaders, localSkillLoader)
	}

	// Initialize browser manager (lazy initialization - browser starts on first use)
	browserCfg := browser.DefaultConfig()
	browserMgr := browser.NewManager(browserCfg)

	// Initialize tool registry with file tools
	registry := tools.NewRegistry()
	registry.Register(tools.NewFileReadTool(workDir))
	registry.Register(tools.NewFileWriteTool(workDir))

	// Add web tools
	registry.Register(tools.NewWebSearchTool()) // Uses SearXNG
	registry.Register(tools.NewWebFetchTool(browserMgr))

	// Add invoke_skill tool if skills are loaded
	if len(skillLoaders) > 0 {
		registry.Register(tools.NewInvokeSkillTool(skillLoaders))
	}

	// Add bash_exec tool with session manager for dynamic session directory
	fallbackDir := filepath.Join(pilotDir, "tmp")
	registry.Register(tools.NewBashExecTool(workDir, sessionManager, fallbackDir))

	// Build system prompt based on available tools
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

	// Create CLI with all options
	cliOpts := &cli.Options{
		SessionManager: sessionManager,
		SkillLoaders:   skillLoaders,
		ToolRegistry:   registry,
		GlobalSkillDir: filepath.Join(pilotDir, "skills"),
	}
	c := cli.New(ag, cliOpts)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer browserMgr.Close() // Ensure browser is closed on exit

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		browserMgr.Close()
		cancel()
	}()

	// If prompt is provided via -p flag, execute it directly
	if prompt != "" {
		return c.ExecutePrompt(ctx, prompt)
	}

	// Run interactive CLI
	return c.Run(ctx)
}
