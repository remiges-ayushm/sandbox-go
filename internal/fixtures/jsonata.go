package fixtures

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"

	"github.com/jsonata-go/jsonata"
)

const jsonataPrefix = "jsonata:"

var (
	jsonataOnce     sync.Once
	jsonataInstance jsonata.JSONataInstance
	jsonataOpenErr  error
)

func getJsonataInstance() (jsonata.JSONataInstance, error) {
	jsonataOnce.Do(func() {
		jsonataInstance, jsonataOpenErr = jsonata.OpenLatest()
	})
	return jsonataInstance, jsonataOpenErr
}

func evaluateJsonataString(value string, requestBody interface{}) interface{} {
	expression := strings.TrimPrefix(value, jsonataPrefix)

	instance, err := getJsonataInstance()
	if err != nil {
		slog.Error("applyJsonata: failed to open jsonata instance", "error", err)
		return value
	}

	expr, err := instance.Compile(expression, false)
	if err != nil {
		slog.Error("applyJsonata: failed to compile expression", "expression", expression, "error", err)
		return value
	}

	inputJSON, err := json.Marshal(requestBody)
	if err != nil {
		slog.Error("applyJsonata: failed to marshal request body", "expression", expression, "error", err)
		return value
	}

	resultJSON, err := expr.Evaluate(inputJSON, nil)
	if err != nil {
		slog.Error("applyJsonata: failed to evaluate expression", "expression", expression, "error", err)
		return value
	}
	if instance.IsUndefined(resultJSON) {
		return value
	}

	var result interface{}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		slog.Error("applyJsonata: failed to unmarshal result", "expression", expression, "error", err)
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
