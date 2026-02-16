package agent

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
)

// runAgenticLoop implements the streaming agentic loop with tool execution
func (a *Agent) runAgenticLoop(ctx context.Context, messages []anthropic.MessageParam, onEvent EventHandler) ([]anthropic.MessageParam, error) {
	for {
		select {
		case <-ctx.Done():
			return messages, ErrContextCanceled
		default:
		}

		// Build request parameters
		params := anthropic.MessageNewParams{
			Model:     a.config.Model,
			MaxTokens: a.config.MaxTokens,
			Messages:  messages,
			Tools:     a.registry.GetToolParams(),
		}

		// Add system prompt if configured
		if a.config.SystemPrompt != "" {
			params.System = []anthropic.TextBlockParam{
				{Text: a.config.SystemPrompt},
			}
		}

		// Add temperature if non-zero
		if a.config.Temperature > 0 {
			params.Temperature = anthropic.Float(a.config.Temperature)
		}

		// Stream the response
		stream := a.client.Messages.NewStreaming(ctx, params)
		message := anthropic.Message{}

		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				if onEvent != nil {
					onEvent(Event{Type: EventError, Error: err})
				}
				return messages, err
			}

			// Handle streaming events
			a.handleStreamEvent(event, onEvent)
		}

		if err := stream.Err(); err != nil {
			if onEvent != nil {
				onEvent(Event{Type: EventError, Error: err})
			}
			return messages, err
		}
		stream.Close()

		// Add assistant response to messages
		messages = append(messages, message.ToParam())

		// Check if we need to execute tools
		if message.StopReason != anthropic.StopReasonToolUse {
			if onEvent != nil {
				onEvent(Event{Type: EventDone})
			}
			break
		}

		// Execute tools and collect results
		toolResults := a.executeTools(ctx, message.Content, onEvent)

		if len(toolResults) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResults...))
		}
	}

	return messages, nil
}

// handleStreamEvent processes streaming events and calls the event handler
func (a *Agent) handleStreamEvent(event anthropic.MessageStreamEventUnion, onEvent EventHandler) {
	if onEvent == nil {
		return
	}

	switch e := event.AsAny().(type) {
	case anthropic.ContentBlockDeltaEvent:
		switch delta := e.Delta.AsAny().(type) {
		case anthropic.TextDelta:
			onEvent(Event{Type: EventText, Text: delta.Text})
		}
	case anthropic.ContentBlockStartEvent:
		if e.ContentBlock.Name != "" {
			onEvent(Event{Type: EventToolCall, ToolName: e.ContentBlock.Name})
		}
	}
}

// executeTools processes tool use blocks and returns tool results
func (a *Agent) executeTools(ctx context.Context, content []anthropic.ContentBlockUnion, onEvent EventHandler) []anthropic.ContentBlockParamUnion {
	var results []anthropic.ContentBlockParamUnion

	for _, block := range content {
		toolUse, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}

		if onEvent != nil {
			inputJSON, _ := json.Marshal(toolUse.Input)
			onEvent(Event{
				Type:      EventToolCall,
				ToolName:  toolUse.Name,
				ToolInput: string(inputJSON),
			})
		}

		// Execute the tool via registry
		result, isError := a.executeTool(ctx, toolUse.Name, toolUse.Input)

		if onEvent != nil {
			onEvent(Event{
				Type:       EventToolResult,
				ToolName:   toolUse.Name,
				ToolResult: result,
			})
		}

		results = append(results, anthropic.NewToolResultBlock(toolUse.ID, result, isError))
	}

	return results
}

// executeTool executes a single tool by name
func (a *Agent) executeTool(ctx context.Context, name string, input any) (string, bool) {
	// Convert input to JSON for the registry
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "failed to marshal tool input: " + err.Error(), true
	}

	return a.registry.Execute(ctx, name, inputJSON)
}
