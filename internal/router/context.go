package router

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// TokenCounter provides token counting utilities
type TokenCounter struct {
	// CharsPerToken is the average characters per token
	// Default is 4 for English text, can be adjusted
	CharsPerToken float64
}

// NewTokenCounter creates a new token counter with default settings
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		CharsPerToken: 4.0, // Conservative estimate for English
	}
}

// EstimateTokens estimates the number of tokens in a text string
// This is an approximation - actual tokenization varies by model
func (tc *TokenCounter) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count characters (excluding whitespace for better accuracy)
	chars := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			chars++
		}
	}

	// Estimate tokens
	tokens := float64(chars) / tc.CharsPerToken

	// Round up to be conservative
	return int(tokens) + 1
}

// EstimateRequestTokens estimates total tokens for a generation request
func (tc *TokenCounter) EstimateRequestTokens(req *GenerateRequest) int {
	total := 0

	// Count prompt tokens
	total += tc.EstimateTokens(req.Prompt)

	// Count system prompt tokens
	total += tc.EstimateTokens(req.SystemPrompt)

	// Count context message tokens
	for _, msg := range req.Context {
		total += tc.EstimateTokens(msg.Content)
		// Add overhead for role and formatting (~5 tokens per message)
		total += 5
	}

	// Add overhead for request structure (~20 tokens)
	total += 20

	return total
}

// ContextValidator validates that a request fits within model context windows
type ContextValidator struct {
	counter *TokenCounter
}

// NewContextValidator creates a new context validator
func NewContextValidator() *ContextValidator {
	return &ContextValidator{
		counter: NewTokenCounter(),
	}
}

// ValidateRequest checks if a request fits within the model's context window
func (cv *ContextValidator) ValidateRequest(req *GenerateRequest, model *Model) error {
	inputTokens := cv.counter.EstimateRequestTokens(req)
	outputTokens := req.MaxTokens
	if outputTokens == 0 {
		outputTokens = 2048 // Default max output
	}

	totalTokens := inputTokens + outputTokens

	if totalTokens > model.ContextWindow {
		return fmt.Errorf(
			"request exceeds model context window: need %d tokens (input: %d + output: %d), model supports %d tokens",
			totalTokens, inputTokens, outputTokens, model.ContextWindow,
		)
	}

	return nil
}

// GetInputTokenCount returns the estimated input token count for a request
func (cv *ContextValidator) GetInputTokenCount(req *GenerateRequest) int {
	return cv.counter.EstimateRequestTokens(req)
}

// TruncationStrategy defines how to truncate oversized contexts
type TruncationStrategy string

const (
	// TruncateOldest removes oldest context messages first
	TruncateOldest TruncationStrategy = "oldest"

	// TruncatePrompt truncates the main prompt (preserves context)
	TruncatePrompt TruncationStrategy = "prompt"

	// TruncateContext removes context messages (preserves prompt)
	TruncateContext TruncationStrategy = "context"

	// TruncateProportional reduces both prompt and context proportionally
	TruncateProportional TruncationStrategy = "proportional"
)

// ContextTruncator handles truncating requests to fit context windows
type ContextTruncator struct {
	counter  *TokenCounter
	strategy TruncationStrategy
}

// NewContextTruncator creates a new context truncator with the given strategy
func NewContextTruncator(strategy TruncationStrategy) *ContextTruncator {
	return &ContextTruncator{
		counter:  NewTokenCounter(),
		strategy: strategy,
	}
}

// TruncateRequest truncates a request to fit within the model's context window
// Returns a new request (does not modify original) and whether truncation occurred
func (ct *ContextTruncator) TruncateRequest(req *GenerateRequest, model *Model) (*GenerateRequest, bool, error) {
	inputTokens := ct.counter.EstimateRequestTokens(req)
	outputTokens := req.MaxTokens
	if outputTokens == 0 {
		outputTokens = 2048
	}

	totalNeeded := inputTokens + outputTokens

	// No truncation needed
	if totalNeeded <= model.ContextWindow {
		return req, false, nil
	}

	// Calculate how many tokens we need to remove
	maxInput := model.ContextWindow - outputTokens
	if maxInput < 100 {
		return nil, false, fmt.Errorf(
			"insufficient context window: model supports %d tokens, but output needs %d tokens",
			model.ContextWindow, outputTokens,
		)
	}

	tokensToRemove := inputTokens - maxInput

	// Create a copy of the request
	truncated := &GenerateRequest{
		Prompt:       req.Prompt,
		SystemPrompt: req.SystemPrompt,
		ModelHint:    req.ModelHint,
		Complexity:   req.Complexity,
		Priority:     req.Priority,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
		Tools:        req.Tools,
		Context:      make([]provider.Message, len(req.Context)),
		ContextSize:  req.ContextSize,
		TaskID:       req.TaskID,
	}
	copy(truncated.Context, req.Context)

	// Apply truncation strategy
	switch ct.strategy {
	case TruncateOldest:
		ct.truncateOldestMessages(truncated, tokensToRemove)
	case TruncatePrompt:
		ct.truncatePrompt(truncated, tokensToRemove)
	case TruncateContext:
		ct.truncateAllContext(truncated, tokensToRemove)
	case TruncateProportional:
		ct.truncateProportional(truncated, tokensToRemove)
	default:
		return nil, false, fmt.Errorf("unknown truncation strategy: %s", ct.strategy)
	}

	return truncated, true, nil
}

// truncateOldestMessages removes oldest context messages until under limit
func (ct *ContextTruncator) truncateOldestMessages(req *GenerateRequest, tokensToRemove int) {
	removed := 0
	newContext := make([]provider.Message, 0, len(req.Context))

	// Remove from start (oldest first)
	for i, msg := range req.Context {
		msgTokens := ct.counter.EstimateTokens(msg.Content) + 5 // +5 for overhead

		if removed < tokensToRemove {
			removed += msgTokens
			continue // Skip this message
		}

		// Keep this message and all following
		newContext = append(newContext, req.Context[i:]...)
		break
	}

	req.Context = newContext
}

// truncatePrompt truncates the main prompt to fit
func (ct *ContextTruncator) truncatePrompt(req *GenerateRequest, tokensToRemove int) {
	promptTokens := ct.counter.EstimateTokens(req.Prompt)
	targetTokens := promptTokens - tokensToRemove

	if targetTokens < 50 {
		targetTokens = 50 // Keep at least 50 tokens
	}

	// Truncate by character count (approximate)
	targetChars := int(float64(targetTokens) * ct.counter.CharsPerToken)
	if targetChars < len(req.Prompt) {
		req.Prompt = req.Prompt[:targetChars] + "...[truncated]"
	}
}

// truncateAllContext removes all context messages
func (ct *ContextTruncator) truncateAllContext(req *GenerateRequest, tokensToRemove int) {
	// Calculate tokens in context
	contextTokens := 0
	for _, msg := range req.Context {
		contextTokens += ct.counter.EstimateTokens(msg.Content) + 5
	}

	if contextTokens >= tokensToRemove {
		// Removing all context is enough
		req.Context = nil
		return
	}

	// Need to truncate prompt too
	req.Context = nil
	remaining := tokensToRemove - contextTokens
	ct.truncatePrompt(req, remaining)
}

// truncateProportional reduces prompt and context proportionally
func (ct *ContextTruncator) truncateProportional(req *GenerateRequest, tokensToRemove int) {
	promptTokens := ct.counter.EstimateTokens(req.Prompt)
	contextTokens := 0
	for _, msg := range req.Context {
		contextTokens += ct.counter.EstimateTokens(msg.Content) + 5
	}

	totalTokens := promptTokens + contextTokens
	if totalTokens == 0 {
		return
	}

	// Calculate proportional reduction
	promptRatio := float64(promptTokens) / float64(totalTokens)
	contextRatio := float64(contextTokens) / float64(totalTokens)

	promptToRemove := int(float64(tokensToRemove) * promptRatio)
	contextToRemove := int(float64(tokensToRemove) * contextRatio)

	// Truncate prompt if needed
	if promptToRemove > 0 {
		ct.truncatePrompt(req, promptToRemove)
	}

	// Truncate context if needed
	if contextToRemove > 0 {
		ct.truncateOldestMessages(req, contextToRemove)
	}
}

// SummarizeContext creates a condensed version of context messages
// This is more sophisticated than simple truncation
func SummarizeContext(messages []provider.Message, targetTokens int) string {
	if len(messages) == 0 {
		return ""
	}

	counter := NewTokenCounter()
	var builder strings.Builder
	currentTokens := 0

	// Build summary by taking key parts of each message
	for i, msg := range messages {
		msgTokens := counter.EstimateTokens(msg.Content)

		if currentTokens+msgTokens <= targetTokens {
			// Include full message
			builder.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
			currentTokens += msgTokens
		} else {
			// Take first part of message
			remaining := targetTokens - currentTokens
			if remaining > 20 {
				chars := int(float64(remaining) * counter.CharsPerToken)
				if chars > 0 && chars < len(msg.Content) {
					builder.WriteString(fmt.Sprintf("[%s]: %s...\n", msg.Role, msg.Content[:chars]))
				}
			}

			// Indicate more messages follow
			if i < len(messages)-1 {
				builder.WriteString(fmt.Sprintf("...[%d more messages omitted]\n", len(messages)-i-1))
			}
			break
		}
	}

	return builder.String()
}
