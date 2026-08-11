package explorer

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestShortSchemaType(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   string
	}{
		{name: "string", schema: map[string]any{"type": "string"}, want: "string"},
		{name: "nullable", schema: map[string]any{"type": []any{"string", "null"}}, want: "string | null"},
		{name: "array", schema: map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}, want: "array[integer]"},
		{name: "ref", schema: map[string]any{"$ref": "#/$defs/Foo"}, want: "Foo"},
		{name: "anyOf", schema: map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}}}, want: "string | number"},
		{name: "empty", schema: map[string]any{}, want: "value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortSchemaType(tt.schema); got != tt.want {
				t.Errorf("shortSchemaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSchemaTypeDetails(t *testing.T) {
	schema := map[string]any{
		"type":        "integer",
		"minimum":     float64(1),
		"maximum":     float64(10),
		"default":     float64(5),
		"enum":        []any{float64(1), float64(2)},
		"pattern":     "^[0-9]+$",
	}
	got := schemaType(schema)
	for _, want := range []string{"integer", ">= 1", "<= 10", "default 5", "one of 1, 2", "pattern constrained"} {
		if !contains(got, want) {
			t.Errorf("schemaType() = %q, want it to contain %q", got, want)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestSchemaAllowsString(t *testing.T) {
	root := map[string]any{"$defs": map[string]any{"code": map[string]any{"type": "string"}}}
	tests := []struct {
		name   string
		schema map[string]any
		want   bool
	}{
		{name: "string", schema: map[string]any{"type": "string"}, want: true},
		{name: "nullable string", schema: map[string]any{"type": []any{"string", "null"}}, want: true},
		{name: "integer", schema: map[string]any{"type": "integer"}, want: false},
		{name: "string const", schema: map[string]any{"const": "abc"}, want: true},
		{name: "number const", schema: map[string]any{"const": float64(2)}, want: false},
		{name: "enum strings", schema: map[string]any{"enum": []any{"a", "b"}}, want: true},
		{name: "enum numbers", schema: map[string]any{"enum": []any{float64(1)}}, want: false},
		{name: "anyOf", schema: map[string]any{"anyOf": []any{map[string]any{"type": "integer"}, map[string]any{"type": "string"}}}, want: true},
		{name: "allOf strings", schema: map[string]any{"allOf": []any{map[string]any{"type": "string"}, map[string]any{"minLength": 1}}}, want: false},
		{name: "local ref string", schema: map[string]any{"$ref": "#/$defs/code"}, want: true},
		{name: "empty", schema: map[string]any{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schemaAllowsString(tt.schema, root); got != tt.want {
				t.Errorf("schemaAllowsString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveLocalRef(t *testing.T) {
	root := map[string]any{
		"$defs": map[string]any{
			"a": map[string]any{"$ref": "#/$defs/b"},
			"b": map[string]any{"type": "string"},
		},
	}
	got := resolveLocalRef(map[string]any{"$ref": "#/$defs/a"}, root)
	if got["type"] != "string" {
		t.Errorf("resolveLocalRef() = %v, want resolved type string", got)
	}
}

func TestParseArgumentValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		schema  map[string]any
		want    any
		wantErr bool
	}{
		{name: "string stays literal", value: "hello", schema: map[string]any{"type": "string"}, want: "hello"},
		{name: "string that parses as json stays literal", value: "123", schema: map[string]any{"type": "string"}, want: "123"},
		{name: "integer parsed", value: "123", schema: map[string]any{"type": "integer"}, want: float64(123)},
		{name: "object parsed", value: `{"a": 1}`, schema: map[string]any{"type": "object"}, want: map[string]any{"a": float64(1)}},
		{name: "array parsed", value: `[1, 2]`, schema: map[string]any{"type": "array"}, want: []any{float64(1), float64(2)}},
		{name: "unparseable non-string errors", value: "notnum", schema: map[string]any{"type": "integer"}, wantErr: true},
		{name: "unparseable loose schema returns literal", value: "whatever", schema: map[string]any{}, want: "whatever"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgumentValue("arg", tt.value, tt.schema, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantJSON, _ := json.Marshal(tt.want)
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("value = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestBuildArguments(t *testing.T) {
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"count": map[string]any{"type": "integer"},
			"tags":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []any{"name"},
		"additionalProperties": false,
	}

	tests := []struct {
		name    string
		jsonArg *string
		pairs   [][2]string
		want    map[string]any
		wantErr string
	}{
		{
			name:  "pairs only",
			pairs: [][2]string{{"name", "bob"}, {"count", "3"}, {"tags", `["a"]`}},
			want:  map[string]any{"name": "bob", "count": float64(3), "tags": []any{"a"}},
		},
		{
			name:  "json overrides pairs",
			jsonArg: ptr(`{"name": "alice", "count": 2}`),
			pairs: [][2]string{{"count", "9"}},
			want:  map[string]any{"name": "alice", "count": float64(9)},
		},
		{
			name:    "missing required",
			pairs:   [][2]string{{"count", "3"}},
			wantErr: "name",
		},
		{
			name:    "invalid json object",
			jsonArg: ptr(`[1, 2]`),
			wantErr: "must be a JSON object",
		},
		{
			name:    "bad json",
			jsonArg: ptr(`{not json`),
			wantErr: "Raw arguments must be valid JSON",
		},
		{
			name:    "additional property rejected",
			pairs:   [][2]string{{"name", "bob"}, {"extra", "x"}},
			wantErr: "extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildArguments(inputSchema, tt.jsonArg, tt.pairs)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got none", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			wantJSON, _ := json.Marshal(tt.want)
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("arguments = %s, want %s", gotJSON, wantJSON)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func TestToolInputSchema(t *testing.T) {
	schema := map[string]any{"type": "object"}
	tool := &mcp.Tool{Name: "t", InputSchema: schema}
	if got := toolInputSchema(tool); got["type"] != "object" {
		t.Errorf("toolInputSchema() = %v, want type object", got)
	}

	raw, _ := json.Marshal(schema)
	tool2 := &mcp.Tool{Name: "t", InputSchema: json.RawMessage(raw)}
	if got := toolInputSchema(tool2); got["type"] != "object" {
		t.Errorf("toolInputSchema(raw) = %v, want type object", got)
	}

	if got := toolInputSchema(nil); got != nil {
		t.Errorf("toolInputSchema(nil) = %v, want nil", got)
	}

	noSchema := &mcp.Tool{Name: "t"}
	if got := toolInputSchema(noSchema); got == nil || len(got) != 0 {
		t.Errorf("toolInputSchema(no schema) = %v, want empty map", got)
	}

	badRaw := &mcp.Tool{Name: "t", InputSchema: json.RawMessage(`{not json`)}
	if got := toolInputSchema(badRaw); got == nil || len(got) != 0 {
		t.Errorf("toolInputSchema(bad raw) = %v, want empty map", got)
	}

	structSchema := &mcp.Tool{Name: "t", InputSchema: struct{ Type string `json:"type"` }{Type: "object"}}
	if got := toolInputSchema(structSchema); got["type"] != "object" {
		t.Errorf("toolInputSchema(struct) = %v, want type object", got)
	}

	unmarshalable := &mcp.Tool{Name: "t", InputSchema: make(chan int)}
	if got := toolInputSchema(unmarshalable); got == nil || len(got) != 0 {
		t.Errorf("toolInputSchema(unmarshalable) = %v, want empty map", got)
	}
}