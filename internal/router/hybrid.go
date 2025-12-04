package router

import (
	"context"
	"fmt"

	"github.com/aescanero/dago-libs/pkg/domain"
	"go.uber.org/zap"
)

// routeHybrid performs hybrid routing: fast CEL rules with LLM fallback
func (r *Router) routeHybrid(ctx context.Context, state *domain.GraphState, config *NodeConfig) (*RoutingResult, error) {
	// Validate configuration
	if err := r.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Phase 1: Try fast rules (CEL)
	r.logger.Debug("trying fast rules",
		zap.Int("num_rules", len(config.FastRules)),
	)

	celState := r.prepareStateForCEL(state)

	for i, rule := range config.FastRules {
		r.logger.Debug("evaluating fast rule",
			zap.Int("rule_index", i),
			zap.String("condition", rule.Condition),
		)

		// Evaluate the condition
		result, err := r.celEvaluator.Evaluate(ctx, rule.Condition, celState)
		if err != nil {
			r.logger.Warn("fast rule evaluation error",
				zap.Int("rule_index", i),
				zap.String("condition", rule.Condition),
				zap.Error(err),
			)
			// Continue to next rule on error
			continue
		}

		// Check if condition is true
		matched, ok := result.(bool)
		if !ok {
			r.logger.Warn("fast rule condition did not return boolean",
				zap.Int("rule_index", i),
				zap.String("condition", rule.Condition),
				zap.Any("result", result),
			)
			continue
		}

		if matched {
			r.logger.Info("fast rule matched",
				zap.Int("rule_index", i),
				zap.String("condition", rule.Condition),
				zap.String("target", rule.Target),
			)

			return &RoutingResult{
				TargetNode: rule.Target,
				Reasoning:  fmt.Sprintf("matched fast rule %d: %s", i, rule.Condition),
				Mode:       string(ModeHybrid),
				PathTaken:  "fast",
			}, nil
		}
	}

	// Phase 2: Fast rules didn't match, try LLM fallback
	r.logger.Debug("fast rules did not match, trying llm fallback")

	if r.llmClient == nil {
		r.logger.Warn("llm client not configured, using fallback route")
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  "fast rules did not match and llm client not configured",
			Mode:       string(ModeHybrid),
			PathTaken:  "fallback",
		}, nil
	}

	// Render prompt template
	prompt, err := r.renderPrompt(state, config.LLMFallback.PromptTemplate)
	if err != nil {
		r.logger.Error("failed to render llm prompt",
			zap.Error(err),
		)
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  fmt.Sprintf("failed to render prompt: %v", err),
			Mode:       string(ModeHybrid),
			PathTaken:  "fallback",
		}, nil
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
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  fmt.Sprintf("llm call failed: %v", err),
			Mode:       string(ModeHybrid),
			PathTaken:  "fallback",
		}, nil
	}

	r.logger.Debug("llm response received",
		zap.String("response", response),
	)

	// Parse LLM response and match to routes
	target, matched := r.matchLLMResponse(response, config.LLMFallback.Routes)
	if !matched {
		r.logger.Warn("llm response did not match any route",
			zap.String("response", response),
		)
		return &RoutingResult{
			TargetNode: config.Fallback,
			Reasoning:  fmt.Sprintf("llm response '%s' did not match any route", response),
			Mode:       string(ModeHybrid),
			PathTaken:  "fallback",
		}, nil
	}

	return &RoutingResult{
		TargetNode: target,
		Reasoning:  fmt.Sprintf("llm classified as: %s (after fast rules failed)", response),
		Mode:       string(ModeHybrid),
		PathTaken:  "slow",
	}, nil
}
