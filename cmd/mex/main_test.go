package main

import (
	"strings"
	"testing"
)

func TestIsFlagError(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{in: "unknown flag: --bogus", want: true},
		{in: "flag needs an argument: --json", want: true},
		{in: "flag provided but not defined: -x", want: true},
		{in: "invalid argument \"x\" for \"-a\" flag", want: true},
		{in: "connection refused", want: false},
		{in: "Tool 'add' not found.", want: false},
		{in: "", want: false},
	}
	for _, tt := range tests {
		if got := isFlagError(tt.in); got != tt.want {
			t.Errorf("isFlagError(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	if !strings.Contains("unknown flag: --x", "unknown flag") {
		t.Error("sanity")
	}
}
