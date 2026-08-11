package explorer

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCapabilityNames(t *testing.T) {
	if got := capabilityNames(nil); got != nil {
		t.Errorf("capabilityNames(nil) = %v, want nil", got)
	}
	all := &mcp.ServerCapabilities{
		Completions: &mcp.CompletionCapabilities{},
		Logging:     &mcp.LoggingCapabilities{},
		Prompts:     &mcp.PromptCapabilities{},
		Resources:   &mcp.ResourceCapabilities{},
		Tools:       &mcp.ToolCapabilities{},
	}
	got := capabilityNames(all)
	want := []string{"completions", "logging", "prompts", "resources", "tools"}
	if len(got) != len(want) {
		t.Fatalf("capabilityNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("capabilityNames() = %v, want %v", got, want)
		}
	}

	partial := &mcp.ServerCapabilities{Resources: &mcp.ResourceCapabilities{}}
	if got := capabilityNames(partial); len(got) != 1 || got[0] != "resources" {
		t.Errorf("capabilityNames(partial) = %v, want [resources]", got)
	}
}

func TestExceptionMessage(t *testing.T) {
	if got := exceptionMessage(nil); got != "" {
		t.Errorf("exceptionMessage(nil) = %q, want empty", got)
	}
	if got := exceptionMessage(errors.New("  boom  ")); got != "boom" {
		t.Errorf("exceptionMessage(err) = %q, want boom", got)
	}
}
