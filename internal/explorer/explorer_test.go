package explorer

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestNewRootCommandStructure(t *testing.T) {
	root := newRootCommand()
	names := map[string]bool{}
	for _, sub := range root.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "prompts", "resources", "inspect", "call", "info", "doctor"} {
		if !names[want] {
			t.Errorf("root command missing subcommand %q (have %v)", want, names)
		}
	}
	if root.Version != version {
		t.Errorf("version = %q, want %q", root.Version, version)
	}
}

func TestExecuteHelp(t *testing.T) {
	var out strings.Builder
	root := newRootCommand()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "mex") {
		t.Errorf("help output missing tool name: %q", out.String())
	}
}

func TestUsageError(t *testing.T) {
	var out strings.Builder
	root := newRootCommand()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"list"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected usage error for list without server URL")
	}
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("error = %T %v, want *UsageError", err, err)
	}
	if ue.Message == "" {
		t.Error("usage error message is empty")
	}
}

func TestExecute(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"mex", "--version"}
	if err := Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestProtocolVersion(t *testing.T) {
	o := commonOptions{}
	if got := o.protocolVersion(); got != statelessProtocolVersion {
		t.Errorf("default protocol = %q, want %q", got, statelessProtocolVersion)
	}
	o.legacy = true
	if got := o.protocolVersion(); got != legacyProtocolVersion {
		t.Errorf("legacy protocol = %q, want %q", got, legacyProtocolVersion)
	}
	if o.isStateless() {
		t.Error("legacy isStateless() = true, want false")
	}
}
