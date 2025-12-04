// Package router implements routing strategies for graph execution flow.
//
// The router supports three routing modes:
//   - Deterministic: Fast, rule-based routing using CEL expressions
//   - LLM: Semantic routing using Large Language Models
//   - Hybrid: Combines CEL rules with LLM fallback for optimal performance
//
// Example deterministic routing:
//
//	config := &NodeConfig{
//	    Mode: ModeDeterministic,
//	    Rules: []Rule{
//	        {Condition: "state.priority == 'high'", Target: "urgent_handler"},
//	        {Condition: "state.score > 0.8", Target: "premium_flow"},
//	    },
//	    Fallback: "default_handler",
//	}
//	result, err := router.Route(ctx, state, config)
//
// Example LLM routing:
//
//	config := &NodeConfig{
//	    Mode: ModeLLM,
//	    LLMConfig: &LLMConfig{
//	        PromptTemplate: "Classify: {{state.message}}",
//	        Routes: map[string]string{
//	            "technical": "tech_support",
//	            "billing": "billing_dept",
//	        },
//	    },
//	    Fallback: "default_handler",
//	}
//	result, err := router.Route(ctx, state, config)
//
// Example hybrid routing:
//
//	config := &NodeConfig{
//	    Mode: ModeHybrid,
//	    FastRules: []Rule{
//	        {Condition: "state.message.contains('refund')", Target: "refund_flow"},
//	    },
//	    LLMFallback: &LLMConfig{
//	        PromptTemplate: "Classify: {{state.message}}",
//	        Routes: map[string]string{"technical": "tech_support"},
//	    },
//	    Fallback: "default_handler",
//	}
//	result, err := router.Route(ctx, state, config)
package router
