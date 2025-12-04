// Package template provides a Handlebars template engine for rendering LLM prompts.
//
// The engine supports Handlebars syntax with custom helpers for common operations.
//
// Example usage:
//
//	engine := template.NewEngine()
//
//	data := map[string]interface{}{
//	    "state": map[string]interface{}{
//	        "message": "Hello World",
//	        "priority": "high",
//	    },
//	}
//
//	template := "Message: {{state.message}}\nPriority: {{uppercase state.priority}}"
//	result, err := engine.Render(template, data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Output: Message: Hello World
//	//         Priority: HIGH
//
// Built-in helpers:
//   - uppercase - Convert string to uppercase
//   - lowercase - Convert string to lowercase
//   - trim - Trim whitespace from string
//   - default - Return default value if first arg is empty
//   - eq - Equality comparison
//   - ne - Inequality comparison
//   - gt - Greater than (for numbers)
//   - lt - Less than (for numbers)
//   - contains - Check if string contains substring
//   - join - Join array elements with separator
//   - len - Get length of array/string/map
//
// Example with helpers:
//
//	{{uppercase name}}                     # "JOHN"
//	{{lowercase email}}                    # "user@example.com"
//	{{default value "N/A"}}                # "N/A" if value is empty
//	{{#if (eq status "active")}}...{{/if}} # Conditional
//	{{#if (gt score 0.8)}}...{{/if}}       # Numeric comparison
//	{{join items ", "}}                    # "a, b, c"
package template
