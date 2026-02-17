package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/google/uuid"
)

// handleHealth handles GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Version: s.config.Version,
	})
}

// handleChat handles POST /chat
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", err.Error())
		return
	}

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required", "")
		return
	}

	// Generate session ID if not provided
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "gw_" + uuid.New().String()[:8]
	}

	// Get or create session
	sess, err := s.sessionManager.GetOrCreate(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get session", err.Error())
		return
	}

	// Build messages: existing session messages + new user message
	userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(req.Message))
	messages := append(sess.GetMessagesWithContext(), userMsg)

	// Collect tool calls during execution
	var toolCalls []ToolCall
	var responseText strings.Builder

	// Create event handler to capture tool calls and text
	eventHandler := func(e agent.Event) {
		switch e.Type {
		case agent.EventText:
			responseText.WriteString(e.Text)
		case agent.EventToolCall:
			toolCalls = append(toolCalls, ToolCall{
				Name:  e.ToolName,
				Input: e.ToolInput,
			})
		case agent.EventToolResult:
			// Update the last tool call with its result
			for i := len(toolCalls) - 1; i >= 0; i-- {
				if toolCalls[i].Name == e.ToolName && toolCalls[i].Result == "" {
					toolCalls[i].Result = e.ToolResult
					break
				}
			}
		}
	}

	// Execute agent
	resultMessages, err := s.agent.Chat(r.Context(), messages, eventHandler)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "agent error", err.Error())
		return
	}

	// Extract the final assistant response
	response := responseText.String()
	if response == "" && len(resultMessages) > 0 {
		// Extract text from the last assistant message
		response = extractTextFromMessages(resultMessages)
	}

	// Update session with new messages
	sess.AddMessage(userMsg)
	for _, msg := range resultMessages {
		sess.AddMessage(msg)
	}

	if err := s.sessionManager.Save(sess); err != nil {
		log.Printf("Warning: failed to save session %s: %v", sessionID, err)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		SessionID: sessionID,
		Response:  response,
		ToolCalls: toolCalls,
	})
}

// handleChatStream handles POST /chat/stream with Server-Sent Events
func (s *Server) handleChatStream(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON", err.Error())
		return
	}

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required", "")
		return
	}

	// Generate session ID if not provided
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "gw_" + uuid.New().String()[:8]
	}

	// Get or create session
	sess, err := s.sessionManager.GetOrCreate(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get session", err.Error())
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	// Build messages
	userMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(req.Message))
	messages := append(sess.GetMessagesWithContext(), userMsg)

	// Create event handler for streaming
	var fullResponse strings.Builder
	eventHandler := func(e agent.Event) {
		switch e.Type {
		case agent.EventText:
			fullResponse.WriteString(e.Text)
			sendSSE(w, flusher, "text", StreamTextData{Content: e.Text})
		case agent.EventToolCall:
			sendSSE(w, flusher, "tool_call", StreamToolCallData{Name: e.ToolName, Input: e.ToolInput})
		case agent.EventToolResult:
			sendSSE(w, flusher, "tool_result", StreamToolResultData{Name: e.ToolName, Result: e.ToolResult})
		case agent.EventError:
			sendSSE(w, flusher, "error", map[string]string{"error": e.Error.Error()})
		}
	}

	// Execute agent
	resultMessages, err := s.agent.Chat(r.Context(), messages, eventHandler)
	if err != nil {
		sendSSE(w, flusher, "error", map[string]string{"error": err.Error()})
		return
	}

	// Update session
	sess.AddMessage(userMsg)
	for _, msg := range resultMessages {
		sess.AddMessage(msg)
	}

	if err := s.sessionManager.Save(sess); err != nil {
		log.Printf("Warning: failed to save session %s: %v", sessionID, err)
	}

	// Send done event
	sendSSE(w, flusher, "done", StreamDoneData{SessionID: sessionID})
}

// handleDeleteSession handles DELETE /session/{id}
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session ID is required", "")
		return
	}

	if err := s.sessionManager.Delete(sessionID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete session", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "deleted",
		"session_id": sessionID,
	})
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, code int, message, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}

// sendSSE sends a Server-Sent Event
func sendSSE(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// extractTextFromMessages extracts text content from the last message(s)
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
