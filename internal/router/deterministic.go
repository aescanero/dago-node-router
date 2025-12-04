package router

import (
	"context"
	"fmt"

	"github.com/aescanero/dago-libs/pkg/domain"
	"go.uber.org/zap"
)

// routeDeterministic performs deterministic routing using CEL rules
func (r *Router) routeDeterministic(ctx context.Context, state *domain.GraphState, config *NodeConfig) (*RoutingResult, error) {
	// Validate configuration
	if err := r.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Prepare state for CEL evaluation
	celState := r.prepareStateForCEL(state)

	// Evaluate rules in order
	for i, rule := range config.Rules {
		r.logger.Debug("evaluating rule",
			zap.Int("rule_index", i),
			zap.String("condition", rule.Condition),
		)

		// Evaluate the condition
		result, err := r.celEvaluator.Evaluate(ctx, rule.Condition, celState)
		if err != nil {
			r.logger.Warn("rule evaluation error",
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
			r.logger.Warn("rule condition did not return boolean",
				zap.Int("rule_index", i),
				zap.String("condition", rule.Condition),
				zap.Any("result", result),
			)
			continue
		}

		if matched {
			r.logger.Info("rule matched",
				zap.Int("rule_index", i),
				zap.String("condition", rule.Condition),
				zap.String("target", rule.Target),
			)

			return &RoutingResult{
				TargetNode: rule.Target,
				Reasoning:  fmt.Sprintf("matched rule %d: %s", i, rule.Condition),
				Mode:       string(ModeDeterministic),
				PathTaken:  "fast",
			}, nil
		}
	}

	// No rules matched, use fallback
	r.logger.Info("no rules matched, using fallback",
		zap.String("fallback", config.Fallback),
	)

	return &RoutingResult{
		TargetNode: config.Fallback,
		Reasoning:  "no rules matched",
		Mode:       string(ModeDeterministic),
		PathTaken:  "fallback",
	}, nil
}

// prepareStateForCEL converts GraphState to a map for CEL evaluation
func (r *Router) prepareStateForCEL(state *domain.GraphState) map[string]interface{} {
	return map[string]interface{}{
		"state": map[string]interface{}{
			"graph_id":    state.GraphID,
			"status":      string(state.Status),
			"inputs":      state.Inputs,
			"node_states": r.convertNodeStates(state.NodeStates),
		},
	}
}

// convertNodeStates converts node states to a CEL-friendly format
func (r *Router) convertNodeStates(nodeStates map[string]*domain.NodeState) map[string]interface{} {
	result := make(map[string]interface{})
	for nodeID, nodeState := range nodeStates {
		result[nodeID] = map[string]interface{}{
			"status":       string(nodeState.Status),
			"output":       nodeState.Output,
			"error":        nodeState.Error,
			"started_at":   nodeState.StartedAt,
			"completed_at": nodeState.CompletedAt,
		}
	}
	return result
}
