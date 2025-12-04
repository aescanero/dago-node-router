# Routing Strategies Guide

## Overview

This guide provides detailed information about routing strategies in the DA Node Router. It covers when to use each mode, configuration patterns, and real-world examples.

## Routing Modes

### Deterministic Routing (CEL)

Fast, rule-based routing using Common Expression Language (CEL).

#### When to Use

✅ **Good For:**
- Clear, objective criteria (numeric thresholds, string matching)
- High-performance requirements (1000+ routes/sec)
- Predictable, testable routing logic
- No semantic understanding needed

❌ **Not Good For:**
- Natural language understanding
- Complex semantic classification
- Ambiguous or subjective criteria

#### Configuration

```json
{
  "node_id": "classifier",
  "type": "router",
  "config": {
    "mode": "deterministic",
    "rules": [
      {
        "condition": "state.priority == 'critical'",
        "target": "emergency_handler"
      },
      {
        "condition": "state.score > 0.9",
        "target": "premium_flow"
      },
      {
        "condition": "state.category == 'refund' && state.amount > 100",
        "target": "manager_approval"
      }
    ],
    "fallback": "standard_flow"
  }
}
```

#### CEL Expression Examples

**String Operations:**
```javascript
// Exact match
state.status == "approved"

// Contains
state.message.contains("urgent")

// Starts with
state.email.startsWith("admin@")

// Ends with
state.filename.endsWith(".pdf")

// Regex match
state.code.matches("^[A-Z]{3}-\\d{4}$")
```

**Numeric Operations:**
```javascript
// Comparisons
state.amount > 1000
state.temperature <= 32.0
state.count >= 5

// Arithmetic
state.total - state.discount > 100
state.price * state.quantity > 500
```

**Boolean Logic:**
```javascript
// AND
state.verified && state.approved

// OR
state.priority == "high" || state.urgent == true

// NOT
!state.suspended

// Complex
(state.age >= 18 && state.country == "US") || state.override == true
```

**Lists and Maps:**
```javascript
// List membership
state.category in ["tech", "billing", "support"]

// List size
size(state.items) > 0

// Map access
state.metadata["region"] == "us-west"

// Map has key
has(state.flags.premium)
```

**Type Checking:**
```javascript
// Type checks
type(state.value) == int
type(state.data) == string

// Null checks
state.optional != null
```

#### Best Practices

1. **Order rules by specificity** - Most specific rules first
2. **Keep expressions simple** - Complex logic is hard to debug
3. **Use appropriate types** - `1` vs `1.0` vs `"1"`
4. **Always provide fallback** - Handle unmatched cases
5. **Test edge cases** - Null values, empty strings, boundary conditions

#### Performance

- **Evaluation time:** < 1ms per rule
- **Throughput:** 1000+ routes/sec per worker
- **Memory:** Minimal (rules compiled once)
- **Latency p99:** < 10ms

---

### LLM Routing

Semantic routing using Large Language Models for natural language understanding.

#### When to Use

✅ **Good For:**
- Natural language classification
- Intent detection
- Semantic similarity matching
- Complex, subjective criteria
- Handling ambiguity

❌ **Not Good For:**
- High-throughput scenarios (> 100 routes/sec)
- Strict latency requirements (< 100ms)
- Simple rule-based logic
- Cost-sensitive applications

#### Configuration

```json
{
  "node_id": "support_classifier",
  "type": "router",
  "config": {
    "mode": "llm",
    "prompt_template": "Classify the following customer message into one of these categories:\n\nCategories:\n- technical: Technical issues, bugs, errors\n- billing: Payments, invoices, subscriptions\n- general: General questions, feedback\n\nMessage: {{state.message}}\n\nRespond with only the category name.",
    "routes": {
      "technical": "tech_support_queue",
      "billing": "billing_department",
      "general": "general_inquiry"
    },
    "fallback": "human_review"
  }
}
```

#### Prompt Templates

Uses Handlebars syntax for variable substitution:

**Basic substitution:**
```handlebars
User message: {{state.message}}
Priority: {{state.priority}}
```

**Conditionals:**
```handlebars
{{#if state.premium}}
Priority customer detected.
{{else}}
Standard customer.
{{/if}}
```

**Loops:**
```handlebars
Previous messages:
{{#each state.history}}
- {{this.text}}
{{/each}}
```

**Helpers:**
```handlebars
// Built-in helpers
{{uppercase state.code}}
{{lowercase state.email}}
```

#### Prompt Engineering Tips

**1. Be specific and clear:**
```
❌ "Classify this message"
✅ "Classify the customer message into exactly one of these categories: technical, billing, general"
```

**2. Provide examples:**
```
Classify the message:

Examples:
- "My payment failed" → billing
- "App crashes on startup" → technical
- "How do I contact support?" → general

Message: {{state.message}}
```

**3. Constrain output format:**
```
Respond with ONLY the category name, nothing else.
Valid responses: technical, billing, general
```

**4. Include context:**
```
Customer: {{state.customer_id}}
History: {{state.interaction_count}} previous interactions
Message: {{state.message}}

Based on the customer's history and current message, classify...
```

#### Response Parsing

The router expects the LLM to return a string matching one of the route keys:

```json
{
  "routes": {
    "yes": "approval_flow",
    "no": "rejection_flow",
    "unclear": "human_review"
  }
}
```

LLM response is trimmed and lowercased before matching.

#### Best Practices

1. **Keep prompts concise** - LLMs perform better with focused prompts
2. **Provide examples** - Few-shot learning improves accuracy
3. **Constrain output** - Explicitly state valid responses
4. **Handle uncertainty** - Include an "unclear" or "other" route
5. **Test thoroughly** - LLM responses can be non-deterministic
6. **Monitor costs** - Track LLM API usage and costs
7. **Set timeouts** - Don't block indefinitely on LLM calls

#### Performance

- **Evaluation time:** 100-500ms (LLM dependent)
- **Throughput:** 10-50 routes/sec per worker
- **Latency p99:** 200-1000ms
- **Cost:** ~$0.001-0.01 per route (model dependent)

---

### Hybrid Routing

Combines fast CEL rules with LLM fallback for optimal performance and flexibility.

#### When to Use

✅ **Good For:**
- Optimizing common cases with rules
- Handling edge cases with LLM
- Balancing performance and flexibility
- Cost optimization (use LLM only when needed)

❌ **Not Good For:**
- Simple scenarios (use deterministic)
- Always need semantic understanding (use LLM)

#### Configuration

```json
{
  "node_id": "smart_router",
  "type": "router",
  "config": {
    "mode": "hybrid",
    "fast_rules": [
      {
        "condition": "state.message.contains('refund')",
        "target": "refund_flow"
      },
      {
        "condition": "state.priority == 'critical'",
        "target": "urgent_queue"
      },
      {
        "condition": "state.verified == false",
        "target": "verification_required"
      }
    ],
    "llm_fallback": {
      "prompt_template": "Classify this message:\n{{state.message}}\n\nCategories: technical, billing, general",
      "routes": {
        "technical": "tech_support",
        "billing": "billing_dept",
        "general": "general_inquiry"
      }
    },
    "fallback": "default_handler"
  }
}
```

#### Execution Flow

```
1. Try fast_rules (CEL) in order
   ├─ Match found → Return target (fast path)
   └─ No match → Continue to step 2

2. Execute llm_fallback
   ├─ LLM returns valid category → Return target (slow path)
   └─ No match or error → Continue to step 3

3. Use fallback route
```

#### Optimization Strategy

**Analyze your routing patterns:**
```
Common patterns:
- 40% → Refund requests (fast rule)
- 30% → Priority escalations (fast rule)
- 20% → Technical issues (needs LLM)
- 10% → General inquiries (needs LLM)
```

**Optimize fast_rules to cover 70-80% of cases:**
- Keep fast_rules focused on high-volume patterns
- Don't try to cover 100% with rules
- Let LLM handle the "long tail"

#### Best Practices

1. **Profile your traffic** - Know your common patterns
2. **Order fast_rules by frequency** - Most common first
3. **Keep fast_rules simple** - Complex rules defeat the purpose
4. **Monitor fast path hit rate** - Aim for 70-80%
5. **Log which path was taken** - For optimization

#### Performance

**Fast path (CEL match):**
- Evaluation time: < 1ms
- Same as deterministic mode

**Slow path (LLM fallback):**
- Evaluation time: 100-500ms
- Same as LLM mode

**Overall (70% fast path):**
- Average latency: ~50ms
- Throughput: 200+ routes/sec
- Cost: 30% of pure LLM mode

---

## Real-World Examples

### Example 1: Customer Support Triage

**Scenario:** Route customer support tickets to appropriate teams.

**Strategy:** Hybrid (fast rules for obvious cases, LLM for ambiguous)

```json
{
  "mode": "hybrid",
  "fast_rules": [
    {
      "condition": "state.message.contains('refund') || state.message.contains('charge')",
      "target": "billing_team"
    },
    {
      "condition": "state.message.contains('bug') || state.message.contains('error')",
      "target": "engineering_team"
    },
    {
      "condition": "state.customer.lifetime_value > 10000",
      "target": "vip_support"
    }
  ],
  "llm_fallback": {
    "prompt_template": "Classify support ticket:\n\n{{state.message}}\n\nCategories: technical, billing, account, general",
    "routes": {
      "technical": "engineering_team",
      "billing": "billing_team",
      "account": "account_management",
      "general": "general_support"
    }
  },
  "fallback": "general_support"
}
```

### Example 2: Content Moderation

**Scenario:** Route content for review based on risk level.

**Strategy:** Deterministic (clear rules for risk scoring)

```json
{
  "mode": "deterministic",
  "rules": [
    {
      "condition": "state.risk_score > 0.9",
      "target": "immediate_removal"
    },
    {
      "condition": "state.risk_score > 0.7",
      "target": "manual_review_high"
    },
    {
      "condition": "state.risk_score > 0.5",
      "target": "manual_review_low"
    },
    {
      "condition": "state.user.trusted == true",
      "target": "auto_approve"
    }
  ],
  "fallback": "default_review"
}
```

### Example 3: Lead Qualification

**Scenario:** Route sales leads to appropriate sales reps.

**Strategy:** LLM (requires understanding of business context)

```json
{
  "mode": "llm",
  "prompt_template": "Analyze this lead and classify their intent:\n\nCompany: {{state.company}}\nMessage: {{state.message}}\nBudget: {{state.budget}}\n\nClassify as:\n- enterprise: Large company, enterprise features, high budget\n- mid-market: Medium company, standard features\n- small-business: Small company or startup, basic needs\n- not-qualified: Not a good fit\n\nProvide only the classification.",
  "routes": {
    "enterprise": "enterprise_sales",
    "mid-market": "mid_market_sales",
    "small-business": "smb_sales",
    "not-qualified": "nurture_campaign"
  },
  "fallback": "general_sales"
}
```

### Example 4: Workflow Routing

**Scenario:** Route tasks in an approval workflow.

**Strategy:** Deterministic (clear business rules)

```json
{
  "mode": "deterministic",
  "rules": [
    {
      "condition": "state.amount > 10000",
      "target": "cfo_approval"
    },
    {
      "condition": "state.amount > 5000 && state.department == 'engineering'",
      "target": "vp_engineering_approval"
    },
    {
      "condition": "state.amount > 1000",
      "target": "manager_approval"
    },
    {
      "condition": "state.amount <= 1000",
      "target": "auto_approve"
    }
  ],
  "fallback": "manual_review"
}
```

## Troubleshooting

### Rule Not Matching

**Problem:** CEL rule should match but doesn't.

**Solutions:**
1. Check data types: `1` vs `"1"`
2. Check null values: use `state.field != null`
3. Test rule in isolation
4. Log state variables

### LLM Returns Unexpected Response

**Problem:** LLM response doesn't match any route.

**Solutions:**
1. Make prompt more specific
2. Add examples to prompt
3. Explicitly list valid responses
4. Check response parsing (trimming, lowercasing)
5. Add logging to see actual LLM response

### Slow Performance

**Problem:** Routing takes too long.

**Solutions:**
- **Deterministic:** Check rule complexity
- **LLM:** Reduce prompt size, check LLM API latency
- **Hybrid:** Increase fast path coverage

### High Costs

**Problem:** LLM routing costs too high.

**Solutions:**
1. Switch to hybrid mode
2. Optimize fast_rules to cover more cases
3. Use cheaper LLM model
4. Cache frequent routing patterns

## Testing

### Testing Deterministic Rules

```go
func TestPriorityRouting(t *testing.T) {
    router := NewRouter(config)

    state := &domain.GraphState{
        Variables: map[string]interface{}{
            "priority": "high",
        },
    }

    target, err := router.Route(ctx, state, nodeConfig)
    assert.NoError(t, err)
    assert.Equal(t, "urgent_handler", target)
}
```

### Testing LLM Routing

```go
func TestLLMRouting(t *testing.T) {
    mockLLM := &MockLLMClient{
        Response: "technical",
    }
    router := NewRouter(config, WithLLMClient(mockLLM))

    state := &domain.GraphState{
        Variables: map[string]interface{}{
            "message": "App crashes on startup",
        },
    }

    target, err := router.Route(ctx, state, nodeConfig)
    assert.NoError(t, err)
    assert.Equal(t, "tech_support", target)
}
```

## Monitoring and Observability

### Key Metrics

1. **Route distribution** - Which routes are taken most
2. **Mode usage** - Fast path vs slow path (hybrid)
3. **Latency** - p50, p95, p99 routing times
4. **Error rate** - Failed routing attempts
5. **Fallback usage** - How often fallback is used
6. **LLM costs** - Track API usage and costs

### Logging

Log routing decisions with context:

```json
{
  "level": "info",
  "msg": "routing_decision",
  "execution_id": "exec-123",
  "node_id": "router-1",
  "mode": "hybrid",
  "path": "fast",
  "target": "refund_flow",
  "reasoning": "matched rule: state.message.contains('refund')",
  "latency_ms": 2
}
```

## Migration Guide

### From Deterministic to Hybrid

1. Keep existing rules as `fast_rules`
2. Add LLM fallback for unmatched cases
3. Monitor fast path hit rate
4. Optimize rules based on patterns

### From LLM to Hybrid

1. Analyze LLM routing patterns
2. Extract common patterns as CEL rules
3. Move rules to `fast_rules`
4. Keep LLM as fallback
5. Monitor cost reduction

## Best Practices Summary

### General
- ✅ Always provide fallback route
- ✅ Log routing decisions with reasoning
- ✅ Monitor routing accuracy
- ✅ Test edge cases thoroughly

### Deterministic
- ✅ Keep rules simple and readable
- ✅ Order rules by specificity
- ✅ Use appropriate data types
- ✅ Handle null values explicitly

### LLM
- ✅ Craft clear, specific prompts
- ✅ Provide examples in prompts
- ✅ Constrain output format
- ✅ Set appropriate timeouts
- ✅ Monitor costs

### Hybrid
- ✅ Profile traffic patterns
- ✅ Optimize fast path coverage (70-80%)
- ✅ Keep fast rules simple
- ✅ Monitor fast vs slow path usage
