package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dmundt/mex/internal/explorer"
)

func main() {
	err := explorer.Execute()
	if err == nil {
		return
	}
	code, message := classifyError(err)
	if message != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	os.Exit(code)
}

// classifyError maps an error to its exit code and the message to print ("" for
// errors that exit silently, like ExitCodeError).
func classifyError(err error) (code int, message string) {
	if err == nil {
		return 0, ""
	}
	var usageErr *explorer.UsageError
	if errors.As(err, &usageErr) {
		return 2, usageErr.Message
	}
	var exitCodeErr *explorer.ExitCodeError
	if errors.As(err, &exitCodeErr) {
		return exitCodeErr.Code, ""
	}
	msg := err.Error()
	if isFlagError(msg) {
		return 2, msg
	}
	return 1, msg
}

// isFlagError reports whether the message looks like the error text produced by
// cobra/pflag for command-line misuse, which we classify as a usage error.
func isFlagError(msg string) bool {
	return strings.HasPrefix(msg, "unknown flag:") ||
		strings.HasPrefix(msg, "flag needs an argument:") ||
		strings.HasPrefix(msg, "flag provided but not defined:") ||
		strings.HasPrefix(msg, "invalid argument")
}
