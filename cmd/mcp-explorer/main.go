package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dmundt/mcp-explorer/internal/explorer"
)

func main() {
	err := explorer.Execute()
	if err == nil {
		return
	}
	var usageErr *explorer.UsageError
	if errors.As(err, &usageErr) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", usageErr)
		os.Exit(2)
	}
	msg := err.Error()
	if isFlagError(msg) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}

// isFlagError reports whether the message looks like the error text produced by
// cobra/pflag for command-line misuse, which we classify as a usage error.
func isFlagError(msg string) bool {
	return strings.HasPrefix(msg, "unknown flag:") ||
		strings.HasPrefix(msg, "flag needs an argument:") ||
		strings.HasPrefix(msg, "flag provided but not defined:") ||
		strings.HasPrefix(msg, "invalid argument")
}