package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/aescanero/dago-libs/pkg/domain"
	"go.uber.org/zap"
)

// routeLLM performs LLM-based routing
func (r *Router) routeLLM(ctx context.Context, state *domain.GraphState, config *NodeConfig) (*RoutingResult, error) {
	// Validate configuration
	if err := r.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if r.llmClient == nil {
		return nil, fmt.Errorf("llm client not configured")
	}

	// Render prompt template
	prompt, err := r.renderPrompt(state, config.LLMConfig.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	r.logger.Debug("calling llm for routing",
		zap.String("prompt", prompt),
	)

	// Call LLM
	response, err := r.callLLM(ctx, prompt)
	if err != nil {
		r.logger.Error("llm call failed",
			zap.Error(err),
		)
		// Fall back to default route on LLM error
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  fmt.Sprintf("llm call failed: %v", err),
			Mode:       string(ModeLLM),
			PathTaken:  "fallback",
		}, nil
	}

	r.logger.Debug("llm response received",
		zap.String("response", response),
	)

	// Parse LLM response and match to routes
	target, matched := r.matchLLMResponse(response, config.LLMConfig.Routes)
	if !matched {
		r.logger.Warn("llm response did not match any route",
			zap.String("response", response),
		)
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  fmt.Sprintf("llm response '%s' did not match any route", response),
			Mode:       string(ModeLLM),
			PathTaken:  "fallback",
		}, nil
	}

	return &RoutingResult{
		TargetNode: target,
		Reasoning:  fmt.Sprintf("llm classified as: %s", response),
		Mode:       string(ModeLLM),
		PathTaken:  "slow",
	}, nil
}

// renderPrompt renders a Handlebars template with state data
func (r *Router) renderPrompt(state *domain.GraphState, template string) (string, error) {
	data := map[string]interface{}{
		"state": map[string]interface{}{
			"graph_id": state.GraphID,
			"status":   string(state.Status),
			"inputs":   state.Inputs,
		},
	}

	// Flatten inputs for easier access
	for key, value := range state.Inputs {
		data[key] = value
	}

	return r.templateEngine.Render(template, data)
}

// callLLM calls the LLM with the given prompt
func (r *Router) callLLM(ctx context.Context, prompt string) (string, error) {
	// Use GenerateCompletion for compatibility with domain types
	req := &domain.LLMRequest{
		Model: "claude-sonnet-4-20250514", // Default model
		Messages: []domain.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 1024,
	}

	respInterface, err := r.llmClient.GenerateCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm completion failed: %w", err)
	}

	// Type assert response
	resp, ok := respInterface.(*domain.LLMResponse)
	if !ok {
		return "", fmt.Errorf("unexpected response type from LLM")
	}

	return resp.Content, nil
}

// matchLLMResponse matches the LLM response to a route
func (r *Router) matchLLMResponse(response string, routes map[string]string) (string, bool) {
	// Normalize response: trim whitespace and convert to lowercase
	normalized := strings.TrimSpace(strings.ToLower(response))

	// Try exact match first
	if target, ok := routes[normalized]; ok {
		return target, true
	}

	// Try case-insensitive match with original routes
	for key, target := range routes {
		if strings.EqualFold(key, normalized) {
			return target, true
		}
	}

	// Try partial match - check if response contains any route key
	for key, target := range routes {
		if strings.Contains(normalized, strings.ToLower(key)) {
			r.logger.Debug("matched route by partial match",
				zap.String("response", response),
				zap.String("matched_key", key),
			)
			return target, true
		}
	}

	return "", false
}
