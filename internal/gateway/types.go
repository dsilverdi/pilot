package gateway

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"` // Optional: auto-created if not provided
	Message   string `json:"message"`
	Stream    bool   `json:"stream,omitempty"` // Optional: streaming response
}

// ChatResponse represents the response from the agent
type ChatResponse struct {
	SessionID string     `json:"session_id"`
	Response  string     `json:"response"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation by the agent
type ToolCall struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Result string `json:"result"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// StreamEvent represents a Server-Sent Event for streaming responses
type StreamEvent struct {
	Event string `json:"event"` // text, tool_call, tool_result, done, error
	Data  any    `json:"data"`
}

// StreamTextData represents streamed text content
type StreamTextData struct {
	Content string `json:"content"`
}

// StreamToolCallData represents a tool call event
type StreamToolCallData struct {
	Name  string `json:"name"`
	Input string `json:"input"`
}

// StreamToolResultData represents a tool result event
type StreamToolResultData struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

// StreamDoneData represents the completion event
type StreamDoneData struct {
	SessionID string `json:"session_id"`
}
