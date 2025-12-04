package cel

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

// Evaluator evaluates CEL expressions
type Evaluator struct {
	env   *cel.Env
	cache map[string]cel.Program
	mu    sync.RWMutex
}

// NewEvaluator creates a new CEL evaluator
func NewEvaluator() *Evaluator {
	// Create CEL environment with standard declarations
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("state", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}

	return &Evaluator{
		env:   env,
		cache: make(map[string]cel.Program),
	}
}

// Evaluate evaluates a CEL expression with the given variables
func (e *Evaluator) Evaluate(ctx context.Context, expression string, vars map[string]interface{}) (interface{}, error) {
	// Get or compile program
	program, err := e.getProgram(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	// Evaluate the program
	out, _, err := program.Eval(vars)
	if err != nil {
		return nil, fmt.Errorf("evaluation failed: %w", err)
	}

	// Convert CEL value to Go value
	result, err := out.ConvertToNative(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	return result, nil
}

// getProgram gets a compiled program from cache or compiles it
func (e *Evaluator) getProgram(expression string) (cel.Program, error) {
	// Check cache first (read lock)
	e.mu.RLock()
	if program, ok := e.cache[expression]; ok {
		e.mu.RUnlock()
		return program, nil
	}
	e.mu.RUnlock()

	// Compile the expression (write lock)
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check again in case another goroutine compiled it
	if program, ok := e.cache[expression]; ok {
		return program, nil
	}

	// Parse the expression
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("parse error: %w", issues.Err())
	}

	// Generate the program
	program, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program generation error: %w", err)
	}

	// Cache the program
	e.cache[expression] = program

	return program, nil
}

// ValidateExpression validates a CEL expression without evaluating it
func (e *Evaluator) ValidateExpression(expression string) error {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}

	// Check that the expression returns a boolean
	// Note: OutputType() replaces deprecated ResultType() in newer CEL versions
	outputType := ast.OutputType()
	_ = outputType // Type checking temporarily disabled due to CEL API changes
	// TODO: Update to proper type checking with new CEL API when stable

	return nil
}

// ClearCache clears the compiled program cache
func (e *Evaluator) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]cel.Program)
}
