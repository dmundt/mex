package explorer

import (
	"strconv"
	"strings"
)

// shortSchemaType returns a short human-readable type label for a schema.
func shortSchemaType(schema map[string]any) string {
	if ts := asList(schema["type"]); ts != nil {
		parts := make([]string, 0, len(ts))
		for _, t := range ts {
			if s, ok := t.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " | ")
	}
	if t, ok := schema["type"].(string); ok {
		if t == "array" {
			items := asMap(schema["items"])
			return "array[" + shortSchemaType(items) + "]"
		}
		return t
	}
	if ref, ok := schema["$ref"].(string); ok {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	if alternatives := firstList(schema["anyOf"], schema["oneOf"]); alternatives != nil {
		parts := make([]string, 0, len(alternatives))
		for _, item := range alternatives {
			parts = append(parts, shortSchemaType(asMap(item)))
		}
		return strings.Join(parts, " | ")
	}
	return "value"
}

// schemaType returns a longer human-readable type label for a schema, including
// constraint details.
func schemaType(schema map[string]any) string {
	label := shortSchemaType(schema)
	var details []string

	if enum := asList(schema["enum"]); enum != nil {
		vals := make([]string, 0, len(enum))
		for _, v := range enum {
			vals = append(vals, marshalJSONCompact(v))
		}
		details = append(details, "one of "+strings.Join(vals, ", "))
	}
	if _, ok := schema["const"]; ok {
		details = append(details, "must be "+marshalJSONCompact(schema["const"]))
	}
	if _, ok := schema["default"]; ok {
		details = append(details, "default "+marshalJSONCompact(schema["default"]))
	}
	if v, ok := number(schema["minimum"]); ok {
		details = append(details, ">= "+numString(v))
	}
	if v, ok := number(schema["exclusiveMinimum"]); ok {
		details = append(details, "> "+numString(v))
	}
	if v, ok := number(schema["maximum"]); ok {
		details = append(details, "<= "+numString(v))
	}
	if v, ok := number(schema["exclusiveMaximum"]); ok {
		details = append(details, "< "+numString(v))
	}
	if v, ok := integer(schema["minItems"]); ok {
		details = append(details, "at least "+strconv.FormatInt(v, 10)+" item(s)")
	}
	if v, ok := integer(schema["maxItems"]); ok {
		details = append(details, "at most "+strconv.FormatInt(v, 10)+" item(s)")
	}
	if _, ok := schema["pattern"]; ok {
		details = append(details, "pattern constrained")
	}

	if len(details) > 0 {
		return label + "; " + strings.Join(details, "; ")
	}
	return label
}

// number returns v as a float64 if it is numeric.
func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// integer returns v as an int64 if it is an integer-valued number.
func integer(v any) (int64, bool) {
	f, ok := number(v)
	if !ok {
		return 0, false
	}
	i := int64(f)
	if float64(i) != f {
		return 0, false
	}
	return i, true
}

// numString formats a float64 the way Python would print a JSON number.
func numString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
