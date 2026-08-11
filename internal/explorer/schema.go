package explorer

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func toolInputSchema(tool *mcp.Tool) map[string]any {
	if tool == nil {
		return nil
	}
	switch v := tool.InputSchema.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return v
	case json.RawMessage:
		var m map[string]any
		if err := json.Unmarshal(v, &m); err == nil {
			return m
		}
		return map[string]any{}
	default:
		data, err := json.Marshal(v)
		if err == nil {
			var m map[string]any
			if err := json.Unmarshal(data, &m); err == nil {
				return m
			}
		}
		return map[string]any{}
	}
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

// firstList returns a if it is a list, otherwise b.
func firstList(a, b any) []any {
	if l := asList(a); l != nil {
		return l
	}
	return asList(b)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func requiredSet(schema map[string]any) map[string]bool {
	required := map[string]bool{}
	for _, r := range asList(schema["required"]) {
		if s, ok := r.(string); ok {
			required[s] = true
		}
	}
	return required
}

// resolveLocalRef resolves local "#/..." references within a schema.
func resolveLocalRef(schema, root map[string]any) map[string]any {
	current := schema
	seen := map[string]bool{}

	for current != nil {
		ref, _ := current["$ref"].(string)
		if ref == "" || !strings.HasPrefix(ref, "#/") {
			return current
		}
		if seen[ref] {
			return current
		}
		seen[ref] = true

		resolved := root
		ok := true
		for _, part := range strings.Split(ref[2:], "/") {
			part = strings.ReplaceAll(part, "~1", "/")
			part = strings.ReplaceAll(part, "~0", "~")
			next, exists := resolved[part]
			if !exists {
				ok = false
				break
			}
			resolved = asMap(next)
			if resolved == nil {
				ok = false
				break
			}
		}
		if !ok {
			return current
		}
		current = resolved
	}
	return schema
}

// schemaAllowsString reports whether a schema can accept a literal string value.
func schemaAllowsString(schema, root map[string]any) bool {
	schema = resolveLocalRef(schema, root)

	switch t := schema["type"].(type) {
	case string:
		if t == "string" {
			return true
		}
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && s == "string" {
				return true
			}
		}
	}
	if _, ok := schema["const"].(string); ok {
		return true
	}
	if enum := asList(schema["enum"]); enum != nil {
		for _, v := range enum {
			if _, ok := v.(string); ok {
				return true
			}
		}
	}

	alternatives := firstList(schema["anyOf"], schema["oneOf"])
	if alternatives != nil {
		for _, item := range alternatives {
			if schemaAllowsString(asMap(item), root) {
				return true
			}
		}
		return false
	}
	if allOf := asList(schema["allOf"]); allOf != nil {
		for _, item := range allOf {
			if !schemaAllowsString(asMap(item), root) {
				return false
			}
		}
		return true
	}
	return false
}

// schemaRequiresJSON reports whether a schema cannot accept a literal string.
func schemaRequiresJSON(schema, root map[string]any) bool {
	schema = resolveLocalRef(schema, root)
	if len(schema) == 0 || schemaAllowsString(schema, root) {
		return false
	}
	for _, key := range []string{"type", "const", "enum", "anyOf", "oneOf", "allOf"} {
		if _, ok := schema[key]; ok {
			return true
		}
	}
	return false
}
