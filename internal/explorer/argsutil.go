package explorer

import "fmt"

// callArgs holds the parsed arguments of the call command, which parses its own
// flags because -a consumes two values per occurrence.
type callArgs struct {
	positionals []string
	pairs       [][2]string
	jsonOutput  bool
	raw         bool
	stateless   bool
	legacy      bool
	help        bool
}

// parseCallArgs parses the raw command-line arguments of the call command.
//
// Supported flags: -a/--argument NAME VALUE (repeatable), --json, --raw,
// --stateless, --legacy, -h/--help. Any other option is rejected as a usage
// error. Everything after the first non-option token (or after "--") that is
// not an option is treated as a positional argument.
func parseCallArgs(args []string) (callArgs, error) {
	var out callArgs
	var positionals []string
	onlyPositional := false

	i := 0
	for i < len(args) {
		arg := args[i]
		if onlyPositional || arg == "-" || !isOption(arg) {
			positionals = append(positionals, arg)
			i++
			continue
		}
		switch arg {
		case "--":
			onlyPositional = true
			i++
		case "--json":
			out.jsonOutput = true
			i++
		case "--raw":
			out.raw = true
			i++
		case "--stateless":
			out.stateless = true
			i++
		case "--legacy":
			out.legacy = true
			i++
		case "-h", "--help":
			out.help = true
			i++
		case "-a", "--argument":
			next := i + 3
			if next > len(args) {
				return callArgs{}, fmt.Errorf("Option '-a' requires 2 arguments")
			}
			out.pairs = append(out.pairs, [2]string{args[i+1], args[i+2]})
			i = next
		default:
			return callArgs{}, fmt.Errorf("No such option: %s", arg)
		}
	}

	if out.stateless && out.legacy {
		return callArgs{}, fmt.Errorf("--stateless and --legacy cannot be used together")
	}
	out.positionals = positionals
	return out, nil
}

// isOption reports whether a token looks like a command-line option. Tokens
// that start with a minus sign followed by a digit (negative numbers) are
// treated as positional values.
func isOption(arg string) bool {
	if len(arg) < 2 || arg[0] != '-' {
		return false
	}
	if arg[1] >= '0' && arg[1] <= '9' {
		return false
	}
	return true
}
