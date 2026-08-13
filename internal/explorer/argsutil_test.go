package explorer

import (
	"strings"
	"testing"
)

func TestParseCallArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantPos    []string
		wantPairs  [][2]string
		wantJSON   bool
		wantRaw    bool
		wantLegacy bool
		wantHelp   bool
		wantErr    string
	}{
		{
			name:    "positionals only",
			args:    []string{"http://x", "tool"},
			wantPos: []string{"http://x", "tool"},
		},
		{
			name:     "json flag",
			args:     []string{"http://x", "tool", "--json"},
			wantPos:  []string{"http://x", "tool"},
			wantJSON: true,
		},
		{
			name:      "two arguments",
			args:      []string{"-a", "name", "value", "--argument", "x", "1", "http://x", "tool"},
			wantPos:   []string{"http://x", "tool"},
			wantPairs: [][2]string{{"name", "value"}, {"x", "1"}},
		},
		{
			name:      "option value looks like flag",
			args:      []string{"--argument", "query", "--yes", "http://x", "tool"},
			wantPos:   []string{"http://x", "tool"},
			wantPairs: [][2]string{{"query", "--yes"}},
		},
		{
			name:       "legacy flag",
			args:       []string{"--legacy", "http://x", "tool"},
			wantPos:    []string{"http://x", "tool"},
			wantLegacy: true,
		},
		{
			name:    "double dash stops parsing",
			args:    []string{"http://x", "tool", "--", "--json"},
			wantPos: []string{"http://x", "tool", "--json"},
		},
		{
			name:    "negative number positional",
			args:    []string{"http://x", "tool", "-5"},
			wantPos: []string{"http://x", "tool", "-5"},
		},
		{
			name:    "unknown option",
			args:    []string{"--bogus", "http://x", "tool"},
			wantErr: "No such option: --bogus",
		},
		{
			name:    "argument missing two values",
			args:    []string{"-a", "onlyone"},
			wantErr: "requires 2 arguments",
		},
		{
			name:     "help flag",
			args:     []string{"-h"},
			wantHelp: true,
		},
		{
			name:    "stateless and legacy conflict",
			args:    []string{"--stateless", "--legacy", "http://x", "tool"},
			wantErr: "cannot be used together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCallArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCallArgs(%q) error = %v, want contains %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCallArgs(%q) unexpected error: %v", tt.args, err)
			}
			if len(got.positionals) != len(tt.wantPos) {
				t.Fatalf("positionals = %v, want %v", got.positionals, tt.wantPos)
			}
			for i := range tt.wantPos {
				if got.positionals[i] != tt.wantPos[i] {
					t.Errorf("positionals[%d] = %q, want %q", i, got.positionals[i], tt.wantPos[i])
				}
			}
			if len(got.pairs) != len(tt.wantPairs) {
				t.Fatalf("pairs = %v, want %v", got.pairs, tt.wantPairs)
			}
			for i := range tt.wantPairs {
				if got.pairs[i] != tt.wantPairs[i] {
					t.Errorf("pairs[%d] = %v, want %v", i, got.pairs[i], tt.wantPairs[i])
				}
			}
			if got.jsonOutput != tt.wantJSON {
				t.Errorf("jsonOutput = %v, want %v", got.jsonOutput, tt.wantJSON)
			}
			if got.raw != tt.wantRaw {
				t.Errorf("raw = %v, want %v", got.raw, tt.wantRaw)
			}
			if got.legacy != tt.wantLegacy {
				t.Errorf("legacy = %v, want %v", got.legacy, tt.wantLegacy)
			}
			if got.help != tt.wantHelp {
				t.Errorf("help = %v, want %v", got.help, tt.wantHelp)
			}
		})
	}
}

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: `npx @modelcontextprotocol/server-everything`, want: []string{"npx", "@modelcontextprotocol/server-everything"}},
		{in: `"C:\Program Files\node\npx.cmd" --arg "a b"`, want: []string{`C:\Program Files\node\npx.cmd`, "--arg", "a b"}},
		{in: `cmd 'single quotes'`, want: []string{"cmd", "single quotes"}},
	}
	for _, tt := range tests {
		got, err := splitCommandLine(tt.in)
		if err != nil {
			t.Fatalf("splitCommandLine(%q) error: %v", tt.in, err)
		}
		if len(got) != len(tt.want) {
			t.Fatalf("splitCommandLine(%q) = %v, want %v", tt.in, got, tt.want)
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("splitCommandLine(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}

	if _, err := splitCommandLine(`cmd "unterminated`); err == nil {
		t.Error("expected error for unterminated quote")
	}
}
