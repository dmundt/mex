package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/dmundt/mex/internal/explorer"
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

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    int
		wantMessage string
	}{
		{name: "nil", err: nil, wantCode: 0, wantMessage: ""},
		{name: "usage", err: &explorer.UsageError{Message: "Incorrect number of arguments."}, wantCode: 2, wantMessage: "Incorrect number of arguments."},
		{name: "exit code", err: &explorer.ExitCodeError{Code: 1}, wantCode: 1, wantMessage: ""},
		{name: "flag", err: errors.New("unknown flag: --bogus"), wantCode: 2, wantMessage: "unknown flag: --bogus"},
		{name: "generic", err: errors.New("connection refused"), wantCode: 1, wantMessage: "connection refused"},
	}
	for _, tt := range tests {
		code, message := classifyError(tt.err)
		if code != tt.wantCode || message != tt.wantMessage {
			t.Errorf("classifyError(%v) = (%d, %q), want (%d, %q)", tt.err, code, message, tt.wantCode, tt.wantMessage)
		}
	}
}
