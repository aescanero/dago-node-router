package router

import (
	"context"
	"fmt"

	"github.com/aescanero/dago-libs/pkg/domain"
	"github.com/aescanero/dago-libs/pkg/ports"
	"github.com/aescanero/dago-node-router/internal/eval/cel"
	"github.com/aescanero/dago-node-router/internal/eval/template"
	"go.uber.org/zap"
)

// RoutingMode represents the routing strategy
type RoutingMode string

const (
	// ModeDeterministic uses CEL expressions for routing
	ModeDeterministic RoutingMode = "deterministic"

	// ModeLLM uses LLM for semantic routing
	ModeLLM RoutingMode = "llm"

	// ModeHybrid uses CEL rules with LLM fallback
	ModeHybrid RoutingMode = "hybrid"
)

// NodeConfig represents the routing configuration for a node
type NodeConfig struct {
	Mode       RoutingMode            `json:"mode"`
	Rules      []Rule                 `json:"rules,omitempty"`
	FastRules  []Rule                 `json:"fast_rules,omitempty"`
	LLMConfig  *LLMConfig             `json:"llm_config,omitempty"`
	LLMFallback *LLMConfig            `json:"llm_fallback,omitempty"`
	Fallback   string                 `json:"fallback"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// Rule represents a CEL-based routing rule
type Rule struct {
	Condition string `json:"condition"`
	Target    string `json:"target"`
}

// LLMConfig represents LLM routing configuration
type LLMConfig struct {
	PromptTemplate string            `json:"prompt_template"`
	Routes         map[string]string `json:"routes"`
}

// RoutingResult represents the result of a routing decision
type RoutingResult struct {
	TargetNode string `json:"target_node"`
	Reasoning  string `json:"reasoning"`
	Mode       string `json:"mode"`
	PathTaken  string `json:"path_taken"` // "fast", "slow", "fallback"
}

// Router handles routing decisions
type Router struct {
	celEvaluator     *cel.Evaluator
	templateEngine   *template.Engine
	llmClient        ports.LLMClient
	logger           *zap.Logger
}

// NewRouter creates a new router
func NewRouter(llmClient ports.LLMClient, logger *zap.Logger) *Router {
	return &Router{
		celEvaluator:   cel.NewEvaluator(),
		templateEngine: template.NewEngine(),
		llmClient:      llmClient,
		logger:         logger,
	}
}

// Route performs routing based on state and configuration
func (r *Router) Route(ctx context.Context, state *domain.GraphState, config *NodeConfig) (*RoutingResult, error) {
	r.logger.Info("routing request",
		zap.String("graph_id", state.GraphID),
		zap.String("mode", string(config.Mode)),
	)

	// Detect mode if not specified
	if config.Mode == "" {
		config.Mode = r.detectMode(config)
	}

	// Route based on mode
	var result *RoutingResult
	var err error

	switch config.Mode {
	case ModeDeterministic:
		result, err = r.routeDeterministic(ctx, state, config)
	case ModeLLM:
		result, err = r.routeLLM(ctx, state, config)
	case ModeHybrid:
		result, err = r.routeHybrid(ctx, state, config)
	default:
		return nil, fmt.Errorf("unknown routing mode: %s", config.Mode)
	}

	if err != nil {
		r.logger.Error("routing failed",
			zap.String("graph_id", state.GraphID),
			zap.String("mode", string(config.Mode)),
			zap.Error(err),
		)
		return nil, err
	}

	r.logger.Info("routing decision",
		zap.String("graph_id", state.GraphID),
		zap.String("mode", string(config.Mode)),
		zap.String("target", result.TargetNode),
		zap.String("path", result.PathTaken),
		zap.String("reasoning", result.Reasoning),
	)

	return result, nil
}

// detectMode detects the routing mode from configuration
func (r *Router) detectMode(config *NodeConfig) RoutingMode {
	// Hybrid mode: has fast_rules and llm_fallback
	if len(config.FastRules) > 0 && config.LLMFallback != nil {
		return ModeHybrid
	}

	// LLM mode: has llm_config
	if config.LLMConfig != nil {
		return ModeLLM
	}

	// Deterministic mode: has rules
	if len(config.Rules) > 0 {
		return ModeDeterministic
	}

	// Default to deterministic
	return ModeDeterministic
}

// validateConfig validates the routing configuration
func (r *Router) validateConfig(config *NodeConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	if config.Fallback == "" {
		return fmt.Errorf("fallback route is required")
	}

	switch config.Mode {
	case ModeDeterministic:
		if len(config.Rules) == 0 {
			return fmt.Errorf("deterministic mode requires rules")
		}
		for i, rule := range config.Rules {
			if rule.Condition == "" {
				return fmt.Errorf("rule %d: condition is required", i)
			}
			if rule.Target == "" {
				return fmt.Errorf("rule %d: target is required", i)
			}
		}

	case ModeLLM:
		if config.LLMConfig == nil {
			return fmt.Errorf("llm mode requires llm_config")
		}
		if config.LLMConfig.PromptTemplate == "" {
			return fmt.Errorf("llm_config.prompt_template is required")
		}
		if len(config.LLMConfig.Routes) == 0 {
			return fmt.Errorf("llm_config.routes is required")
		}

	case ModeHybrid:
		if len(config.FastRules) == 0 {
			return fmt.Errorf("hybrid mode requires fast_rules")
		}
		if config.LLMFallback == nil {
			return fmt.Errorf("hybrid mode requires llm_fallback")
		}
		if config.LLMFallback.PromptTemplate == "" {
			return fmt.Errorf("llm_fallback.prompt_template is required")
		}
		if len(config.LLMFallback.Routes) == 0 {
			return fmt.Errorf("llm_fallback.routes is required")
		}
	}

	return nil
}
