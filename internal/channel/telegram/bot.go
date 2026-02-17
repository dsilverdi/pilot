package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/dsilverdi/pilot/internal/session"
)

// Bot represents a Telegram bot that interfaces with the agent
type Bot struct {
	api            *tgbotapi.BotAPI
	agent          *agent.Agent
	sessionManager *session.Manager
	config         *Config
}

// NewBot creates a new Telegram bot
func NewBot(cfg *Config, ag *agent.Agent, sm *session.Manager) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	return &Bot{
		api:            api,
		agent:          ag,
		sessionManager: sm,
		config:         cfg,
	}, nil
}

// Username returns the bot's username
func (b *Bot) Username() string {
	return b.api.Self.UserName
}

// Start starts the bot and listens for updates
func (b *Bot) Start(ctx context.Context) error {
	log.Printf("Telegram bot @%s started", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("Telegram bot shutting down...")
			b.api.StopReceivingUpdates()
			return nil
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

// handleUpdate processes an incoming update
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID

	// Check if user is allowed
	if !b.config.IsUserAllowed(userID) {
		b.sendPlainMessage(chatID, FormatUnauthorized())
		return
	}

	// Handle commands
	if msg.IsCommand() {
		b.handleCommand(ctx, msg)
		return
	}

	// Handle regular messages
	b.handleMessage(ctx, msg)
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	switch msg.Command() {
	case "start":
		b.sendPlainMessage(chatID, FormatWelcome(b.api.Self.UserName))

	case "help":
		b.sendPlainMessage(chatID, FormatHelp())

	case "clear":
		sessionID := b.sessionID(userID)
		if err := b.sessionManager.Delete(sessionID); err != nil {
			log.Printf("Failed to delete session %s: %v", sessionID, err)
		}
		b.sendPlainMessage(chatID, FormatSessionCleared())

	case "status":
		// Count active sessions (rough estimate)
		sessions, _ := b.sessionManager.List()
		tgSessions := 0
		for _, s := range sessions {
			if strings.HasPrefix(s.Name, "tg_") {
				tgSessions++
			}
		}
		b.sendPlainMessage(chatID, FormatStatus(b.api.Self.UserName, tgSessions))

	default:
		b.sendPlainMessage(chatID, "Hmm, I don't know that command. Try /help to see what I can do!")
	}
}

// handleMessage processes regular text messages
func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	text := msg.Text

	if text == "" {
		return
	}

	// Show typing indicator
	typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	b.api.Send(typing)

	// Get or create session
	sessionID := b.sessionID(userID)
	sess, err := b.sessionManager.GetOrCreate(sessionID)
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		b.sendPlainMessage(chatID, FormatError(fmt.Errorf("failed to get session")))
		return
	}

	// Build messages
	userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(text))
	messages := append(sess.GetMessagesWithContext(), userMsg)

	// Collect response
	var responseText strings.Builder
	var toolCalls []string
	toolNotified := false // Only notify for first tool call to avoid spam

	// Event handler
	eventHandler := func(e agent.Event) {
		switch e.Type {
		case agent.EventText:
			responseText.WriteString(e.Text)
		case agent.EventToolCall:
			toolCalls = append(toolCalls, e.ToolName)
			// Only send notification for the first tool call
			if !toolNotified {
				b.sendPlainMessage(chatID, FormatToolCall(e.ToolName))
				toolNotified = true
			}
			// Keep showing typing indicator
			typing := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
			b.api.Send(typing)
		}
	}

	// Call agent
	resultMessages, err := b.agent.Chat(ctx, messages, eventHandler)
	if err != nil {
		log.Printf("Agent error: %v", err)
		b.sendPlainMessage(chatID, FormatError(err))
		return
	}

	// Get response text
	response := responseText.String()
	if response == "" {
		response = extractTextFromMessages(resultMessages)
	}

	// Update session
	sess.AddMessage(userMsg)
	for _, m := range resultMessages {
		sess.AddMessage(m)
	}
	if err := b.sessionManager.Save(sess); err != nil {
		log.Printf("Failed to save session: %v", err)
	}

	// Format and send response with HTML (split if too long)
	formattedResponse := FormatResponse(response)
	parts := SplitLongMessageHTML(formattedResponse)
	for _, part := range parts {
		b.sendMessageHTML(chatID, part)
	}
}

// sessionID generates a session ID for a Telegram user
func (b *Bot) sessionID(userID int64) string {
	return fmt.Sprintf("tg_%d", userID)
}

// sendMessageHTML sends a text message with HTML formatting
func (b *Bot) sendMessageHTML(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send HTML message: %v", err)
		// Fallback to plain text if HTML fails
		b.sendPlainMessage(chatID, text)
	}
}

// sendPlainMessage sends a plain text message without formatting
func (b *Bot) sendPlainMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// extractTextFromMessages extracts text from agent response messages
func extractTextFromMessages(messages []anthropic.MessageParam) string {
	var texts []string
	for _, msg := range messages {
		text := extractTextContent(msg.Content)
		if text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "")
}

// extractTextContent extracts text from message content blocks
func extractTextContent(content []anthropic.ContentBlockParamUnion) string {
	// Marshal to JSON and extract text fields
	data, err := json.Marshal(content)
	if err != nil {
		return ""
	}

	var blocks []map[string]any
	if err := json.Unmarshal(data, &blocks); err != nil {
		return string(data)
	}

	var textParts []string
	for _, block := range blocks {
		if text, ok := block["text"].(string); ok {
			textParts = append(textParts, text)
		}
	}

	return strings.Join(textParts, " ")
}
