// Package cel provides a CEL (Common Expression Language) evaluator for deterministic routing.
//
// CEL is a non-Turing complete expression language that provides fast, safe evaluation
// of conditions for routing decisions.
//
// Example usage:
//
//	evaluator := cel.NewEvaluator()
//
//	vars := map[string]interface{}{
//	    "state": map[string]interface{}{
//	        "priority": "high",
//	        "score": 0.95,
//	    },
//	}
//
//	result, err := evaluator.Evaluate(ctx, "state.priority == 'high'", vars)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	matched := result.(bool) // true
//
// Supported operations:
//   - Comparisons: ==, !=, <, <=, >, >=
//   - Boolean logic: &&, ||, !
//   - String operations: contains, startsWith, endsWith, matches
//   - Arithmetic: +, -, *, /, %
//   - List operations: in, size
//   - Map access: state.field, state["field"]
package cel
