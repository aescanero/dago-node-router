package template

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
)

// Engine renders Handlebars templates
type Engine struct {
	cache map[string]*raymond.Template
	mu    sync.RWMutex
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	engine := &Engine{
		cache: make(map[string]*raymond.Template),
	}

	// Register custom helpers
	engine.registerHelpers()

	return engine
}

// Render renders a template with the given data
func (e *Engine) Render(templateStr string, data interface{}) (string, error) {
	// Get or compile template
	tmpl, err := e.getTemplate(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to compile template: %w", err)
	}

	// Execute the template
	result, err := tmpl.Exec(data)
	if err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return result, nil
}

// getTemplate gets a compiled template from cache or compiles it
func (e *Engine) getTemplate(templateStr string) (*raymond.Template, error) {
	// Check cache first (read lock)
	e.mu.RLock()
	if tmpl, ok := e.cache[templateStr]; ok {
		e.mu.RUnlock()
		return tmpl, nil
	}
	e.mu.RUnlock()

	// Compile the template (write lock)
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check again in case another goroutine compiled it
	if tmpl, ok := e.cache[templateStr]; ok {
		return tmpl, nil
	}

	// Parse and compile the template
	tmpl, err := raymond.Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Cache the template
	e.cache[templateStr] = tmpl

	return tmpl, nil
}

// ValidateTemplate validates a template without rendering it
func (e *Engine) ValidateTemplate(templateStr string) error {
	_, err := raymond.Parse(templateStr)
	return err
}

// ClearCache clears the compiled template cache
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = make(map[string]*raymond.Template)
}

// registerHelpers registers custom Handlebars helpers
func (e *Engine) registerHelpers() {
	// uppercase helper
	raymond.RegisterHelper("uppercase", func(str string) string {
		return strings.ToUpper(str)
	})

	// lowercase helper
	raymond.RegisterHelper("lowercase", func(str string) string {
		return strings.ToLower(str)
	})

	// trim helper
	raymond.RegisterHelper("trim", func(str string) string {
		return strings.TrimSpace(str)
	})

	// default helper - return default value if first arg is empty
	raymond.RegisterHelper("default", func(value interface{}, defaultValue interface{}) interface{} {
		if value == nil || value == "" {
			return defaultValue
		}
		return value
	})

	// eq helper - equality comparison
	raymond.RegisterHelper("eq", func(a, b interface{}) bool {
		return a == b
	})

	// ne helper - inequality comparison
	raymond.RegisterHelper("ne", func(a, b interface{}) bool {
		return a != b
	})

	// gt helper - greater than (for numbers)
	raymond.RegisterHelper("gt", func(a, b float64) bool {
		return a > b
	})

	// lt helper - less than (for numbers)
	raymond.RegisterHelper("lt", func(a, b float64) bool {
		return a < b
	})

	// contains helper - check if string contains substring
	raymond.RegisterHelper("contains", func(str, substr string) bool {
		return strings.Contains(str, substr)
	})

	// join helper - join array elements with separator
	raymond.RegisterHelper("join", func(arr []interface{}, sep string) string {
		strs := make([]string, len(arr))
		for i, v := range arr {
			strs[i] = fmt.Sprint(v)
		}
		return strings.Join(strs, sep)
	})

	// len helper - get length of array/string
	raymond.RegisterHelper("len", func(value interface{}) int {
		switch v := value.(type) {
		case string:
			return len(v)
		case []interface{}:
			return len(v)
		case map[string]interface{}:
			return len(v)
		default:
			return 0
		}
	})
}
