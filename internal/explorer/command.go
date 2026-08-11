package explorer

import (
	"fmt"
	"os/exec"
	"strings"
)

// newCommand is a thin wrapper over exec.Command so tests can override it.
var newCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// splitCommandLine splits a command line into tokens, honoring single and
// double quotes.
func splitCommandLine(line string) ([]string, error) {
	var parts []string
	var current strings.Builder
	var quote rune
	inWord := false

	for _, r := range line {
		switch quote {
		case 0:
			switch r {
			case ' ', '\t', '\r', '\n':
				if inWord {
					parts = append(parts, current.String())
					current.Reset()
					inWord = false
				}
			case '\'', '"':
				quote = r
				inWord = true
			default:
				current.WriteRune(r)
				inWord = true
			}
		case '\'':
			if r == '\'' {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case '"':
			if r == '"' {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if inWord {
		parts = append(parts, current.String())
	}
	return parts, nil
}