package explorer

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// failingListClient fails every list call.
type failingListClient struct {
	repeatingCursorClient
	err error
}

func (f failingListClient) ListToolsPage(context.Context, string) ([]*mcp.Tool, string, error) {
	return nil, "", f.err
}

// errReader is an io.Reader that always fails.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("stdin boom") }

func TestRawDiscoverSupportedVersions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"supportedVersions":["2026-07-28","2025-11-25"],"capabilities":{}}}`))
		}))
		defer srv.Close()

		got, err := rawDiscoverSupportedVersions(context.Background(), srv.URL, statelessProtocolVersion)
		if err != nil {
			t.Fatalf("rawDiscoverSupportedVersions: %v", err)
		}
		want := []string{"2026-07-28", "2025-11-25"}
		if len(got) != len(want) {
			t.Fatalf("versions = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("versions = %v, want %v", got, want)
			}
		}
	})

	t.Run("jsonrpc error response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"nope"}}`))
		}))
		defer srv.Close()

		_, err := rawDiscoverSupportedVersions(context.Background(), srv.URL, statelessProtocolVersion)
		if err == nil || !strings.Contains(err.Error(), "nope") {
			t.Fatalf("error = %v, want JSON-RPC error", err)
		}
	})

	t.Run("malformed result", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"supportedVersions":42}}`))
		}))
		defer srv.Close()

		_, err := rawDiscoverSupportedVersions(context.Background(), srv.URL, statelessProtocolVersion)
		if err == nil || !strings.Contains(err.Error(), "decoding discover result") {
			t.Fatalf("error = %v, want decoding error", err)
		}
	})

	t.Run("invalid transport", func(t *testing.T) {
		if _, err := rawDiscoverSupportedVersions(context.Background(), "", statelessProtocolVersion); err == nil {
			t.Fatal("expected error for empty URL")
		}
	})
}

func TestValidateArguments(t *testing.T) {
	valid := func(schema map[string]any) map[string]any {
		schema["type"] = "object"
		schema["properties"] = map[string]any{"x": map[string]any{"type": "integer"}}
		return schema
	}

	t.Run("does not mutate caller schema", func(t *testing.T) {
		input := valid(map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema#"})
		if err := validateArguments(input, map[string]any{"x": float64(1)}); err != nil {
			t.Fatal(err)
		}
		if input["$schema"] != "https://json-schema.org/draft/2020-12/schema#" {
			t.Errorf("caller $schema was mutated: %v", input["$schema"])
		}
	})

	for name, uri := range map[string]string{
		"draft07-http":  "http://json-schema.org/draft-07/schema#",
		"draft07-https": "https://json-schema.org/draft-07/schema#",
		"2020-12":       "https://json-schema.org/draft/2020-12/schema",
		"2020-12-hash":  "https://json-schema.org/draft/2020-12/schema#",
		"unknown":       "https://example.com/weird/schema",
		"absent":        "",
	} {
		t.Run(name, func(t *testing.T) {
			input := map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "integer"}}}
			if uri != "" {
				input["$schema"] = uri
			}
			if err := validateArguments(input, map[string]any{"x": float64(1)}); err != nil {
				t.Errorf("validateArguments(%s): %v", name, err)
			}
		})
	}

	t.Run("schema mismatch", func(t *testing.T) {
		input := valid(map[string]any{})
		input["required"] = []any{"x"}
		err := validateArguments(input, map[string]any{})
		if err == nil || !strings.Contains(err.Error(), "do not match input schema") {
			t.Fatalf("error = %v, want schema mismatch", err)
		}
	})

	t.Run("unmarshalable schema", func(t *testing.T) {
		err := validateArguments(map[string]any{"bad": make(chan int)}, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("error = %v, want schema invalid", err)
		}
	})

	t.Run("dangling ref", func(t *testing.T) {
		err := validateArguments(map[string]any{"$ref": "#/$defs/missing"}, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("error = %v, want schema invalid", err)
		}
	})
}

func TestResolveLocalRefExtended(t *testing.T) {
	root := map[string]any{
		"$defs": map[string]any{
			"a/b":  map[string]any{"type": "string"},
			"a~b":  map[string]any{"type": "number"},
			"x":    "not a map",
			"loop": map[string]any{"$ref": "#/$defs/loop"},
		},
	}

	// No $ref: returned as-is.
	if got := resolveLocalRef(map[string]any{"type": "integer"}, root); got["type"] != "integer" {
		t.Errorf("no-ref schema not returned as-is: %v", got)
	}

	// Non-local ref: returned as-is.
	external := map[string]any{"$ref": "https://example.com/schema.json#/x"}
	if got := resolveLocalRef(external, root); got["$ref"] != external["$ref"] {
		t.Errorf("external ref not preserved: %v", got)
	}

	// Escaped pointer segments (~1 and ~0).
	if got := resolveLocalRef(map[string]any{"$ref": "#/$defs/a~1b"}, root); got["type"] != "string" {
		t.Errorf("~1 escape not resolved: %v", got)
	}
	if got := resolveLocalRef(map[string]any{"$ref": "#/$defs/a~0b"}, root); got["type"] != "number" {
		t.Errorf("~0 escape not resolved: %v", got)
	}

	// Missing target: returns the original ref schema.
	missing := map[string]any{"$ref": "#/$defs/nope"}
	if got := resolveLocalRef(missing, root); got["$ref"] != "#/$defs/nope" {
		t.Errorf("missing ref not preserved: %v", got)
	}

	// Target is not a map: returns the original ref schema.
	notMap := map[string]any{"$ref": "#/$defs/x"}
	if got := resolveLocalRef(notMap, root); got["$ref"] != "#/$defs/x" {
		t.Errorf("non-map target not preserved: %v", got)
	}

	// Circular ref: returns the original ref schema instead of looping.
	loop := map[string]any{"$ref": "#/$defs/loop"}
	if got := resolveLocalRef(loop, root); got["$ref"] != "#/$defs/loop" {
		t.Errorf("circular ref not guarded: %v", got)
	}
}

func TestNumberAndInteger(t *testing.T) {
	for name, v := range map[string]any{
		"float64": float64(1),
		"float32": float32(2),
		"int":     int(3),
		"int64":   int64(4),
	} {
		if got, ok := number(v); !ok || got == 0 {
			t.Errorf("number(%s) = %v, %v", name, got, ok)
		}
	}
	if _, ok := number("x"); ok {
		t.Error("number(string) should be false")
	}
	if _, ok := number(nil); ok {
		t.Error("number(nil) should be false")
	}

	if got, ok := integer(float64(5)); !ok || got != 5 {
		t.Errorf("integer(5) = %d, %v", got, ok)
	}
	if _, ok := integer("x"); ok {
		t.Error("integer(string) should be false")
	}
	if _, ok := integer(float64(5.5)); ok {
		t.Error("integer(5.5) should be false")
	}
}

func TestSchemaTypeConstraints(t *testing.T) {
	schema := map[string]any{
		"type":             "integer",
		"exclusiveMinimum": float64(1),
		"exclusiveMaximum": float64(10),
		"minItems":         int(2),
		"maxItems":         int64(5),
		"pattern":          "^x",
		"const":            "fixed",
		"default":          "d",
	}
	got := schemaType(schema)
	for _, want := range []string{"> 1", "< 10", "at least 2", "at most 5", "pattern constrained", `must be "fixed"`, `default "d"`} {
		if !contains(got, want) {
			t.Errorf("schemaType() = %q, want it to contain %q", got, want)
		}
	}
}

func TestMarshalJSONErrorBranches(t *testing.T) {
	if _, err := marshalJSONIndent(make(chan int)); err == nil {
		t.Error("marshalJSONIndent should error on channel")
	}
	if got := marshalJSONCompact(make(chan int)); got == "" {
		t.Error("marshalJSONCompact should return a fallback string")
	}
}

func TestFetchServerInfoLegacySkipsDiscover(t *testing.T) {
	fake := newFakeClient(nil, 10)
	fake.setInfo(Info{
		URL:             "http://fake/mcp",
		Mode:            "legacy",
		Negotiation:     "initialize",
		ProtocolVersion: legacyProtocolVersion,
	})
	orig := NewClient
	NewClient = func(opts ClientOptions) (MCPClient, error) { return fake, nil }
	t.Cleanup(func() { NewClient = orig })

	info, err := fetchServerInfo(context.Background(), "http://fake/mcp", legacyProtocolVersion)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode != "legacy" {
		t.Errorf("mode = %q, want legacy", info.Mode)
	}
	if len(info.SupportedVersions) != 0 {
		t.Errorf("legacy mode should not probe discover, got %v", info.SupportedVersions)
	}
}

func TestFetchServerInfoClientError(t *testing.T) {
	installNewClientErr(t, errors.New("connection refused"))
	if _, err := fetchServerInfo(context.Background(), "http://bad/mcp", statelessProtocolVersion); err == nil {
		t.Fatal("expected client error")
	}
}

func TestPaginateError(t *testing.T) {
	_, err := listAllTools(context.Background(), failingListClient{err: errors.New("boom")})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want list error", err)
	}
}

func TestArgumentPairFlag(t *testing.T) {
	var f argumentPairFlag
	if err := f.Set("x"); err != nil {
		t.Fatal(err)
	}
	if f.String() != "" {
		t.Errorf("String() = %q, want empty", f.String())
	}
	if f.Type() != "NAME VALUE" {
		t.Errorf("Type() = %q, want NAME VALUE", f.Type())
	}
}

func TestWriteCallJSONNoContent(t *testing.T) {
	var sb strings.Builder
	cmd := newRootCommand()
	cmd.SetOut(&sb)
	if err := writeCallJSON(cmd, &mcp.CallToolResult{}); err != nil {
		t.Fatal(err)
	}
	if sb.String() != "null\n" {
		t.Errorf("output = %q, want null", sb.String())
	}
}

func TestInvokeToolErrors(t *testing.T) {
	ctx := context.Background()

	fake := newFakeClient(nil, 10)
	fake.findError = errors.New("boom")
	if _, err := invokeTool(ctx, fake, "x", nil, nil); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want list error", err)
	}

	fake.findError = nil
	if _, err := invokeTool(ctx, fake, "missing", nil, nil); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestConnectError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	transport, err := newTransport(url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(context.Background(), transport, ClientOptions{ProtocolVersion: statelessProtocolVersion}); err == nil {
		t.Fatal("expected connect error against closed server")
	}
}
