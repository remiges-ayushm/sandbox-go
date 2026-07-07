package fixtures

import (
	"log"
	"strings"

	jsonata "github.com/blues/jsonata-go"
)

const jsonataPrefix = "jsonata:"

func evaluateJsonataString(value string, requestBody interface{}) interface{} {
	expression := strings.TrimPrefix(value, jsonataPrefix)
	expr, err := jsonata.Compile(expression)
	if err != nil {
		log.Printf(`applyJsonata: failed to compile "%s": %v`, expression, err)
		return value
	}

	result, err := expr.Eval(requestBody)
	if err != nil {
		log.Printf(`applyJsonata: failed to evaluate "%s": %v`, expression, err)
		return value
	}
	if result == nil {
		return value
	}
	return result
}

// ApplyJsonata recursively walks template, evaluating any string value prefixed with
// "jsonata:" as a JSONata expression against requestBody, and leaving all other values as-is.
func ApplyJsonata(template interface{}, requestBody interface{}) interface{} {
	switch v := template.(type) {
	case string:
		if strings.HasPrefix(v, jsonataPrefix) {
			return evaluateJsonataString(v, requestBody)
		}
		return v
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = ApplyJsonata(item, requestBody)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, value := range v {
			result[key] = ApplyJsonata(value, requestBody)
		}
		return result
	default:
		return template
	}
}
