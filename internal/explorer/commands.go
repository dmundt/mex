package explorer

import (
	"errors"
	"fmt"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// writeJSON writes v as indented JSON followed by a newline.
func writeJSON(cmd *cobra.Command, v any) error {
	data, err := marshalJSONIndent(v)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// writeText writes text to the command's output.
func writeText(cmd *cobra.Command, text string) error {
	fmt.Fprint(cmd.OutOrStdout(), text)
	return nil
}

func newListCommand() *cobra.Command {
	var common commonOptions
	var noTruncate bool
	cmd := &cobra.Command{
		Use:   "list URL",
		Short: "List the tools exposed by an MCP server at URL.",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: common.protocolVersion(), Context: cmd.Context()})
			if err != nil {
				return err
			}
			defer client.Close()
			tools, err := listAllTools(cmd.Context(), client)
			if err != nil {
				return err
			}
			if common.jsonOutput {
				return writeJSON(cmd, tools)
			}
			return writeText(cmd, renderToolList(tools, noTruncate))
		},
	}
	cmd.Flags().BoolVarP(&noTruncate, "no-truncate", "N", false, "Show full descriptions and detailed parameters.")
	addCommonFlags(cmd, &common)
	return cmd
}

func newPromptsCommand() *cobra.Command {
	var common commonOptions
	cmd := &cobra.Command{
		Use:   "prompts URL",
		Short: "List the prompts exposed by an MCP server at URL.",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: common.protocolVersion(), Context: cmd.Context()})
			if err != nil {
				return err
			}
			defer client.Close()
			prompts, err := listAllPrompts(cmd.Context(), client)
			if err != nil {
				return err
			}
			if common.jsonOutput {
				return writeJSON(cmd, prompts)
			}
			return writeText(cmd, renderPromptList(prompts))
		},
	}
	addCommonFlags(cmd, &common)
	return cmd
}

func newResourcesCommand() *cobra.Command {
	var common commonOptions
	cmd := &cobra.Command{
		Use:   "resources URL",
		Short: "List the resources exposed by an MCP server at URL.",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: common.protocolVersion(), Context: cmd.Context()})
			if err != nil {
				return err
			}
			defer client.Close()
			resources, err := listAllResources(cmd.Context(), client)
			if err != nil {
				return err
			}
			if common.jsonOutput {
				return writeJSON(cmd, resources)
			}
			return writeText(cmd, renderResourceList(resources))
		},
	}
	addCommonFlags(cmd, &common)
	return cmd
}

func newInspectCommand() *cobra.Command {
	var common commonOptions
	cmd := &cobra.Command{
		Use:   "inspect URL TOOL_NAME",
		Short: "Inspect one tool exposed by an MCP server at URL.",
		Args:  exactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, name := args[0], args[1]
			client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: common.protocolVersion(), Context: cmd.Context()})
			if err != nil {
				return err
			}
			defer client.Close()
			tool, err := client.FindTool(cmd.Context(), name)
			if err != nil {
				return err
			}
			if tool == nil {
				return fmt.Errorf("Tool '%s' not found.", name)
			}
			if common.jsonOutput {
				return writeJSON(cmd, tool)
			}
			return writeText(cmd, renderToolInspect(tool))
		},
	}
	addCommonFlags(cmd, &common)
	return cmd
}

func newCallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call URL TOOL_NAME [ARGUMENTS_JSON]",
		Short: "Call a tool with optional JSON and individual arguments.",
		// Argument parsing and flag handling are done by parseCallArgs so that
		// -a can consume two values per occurrence. For that reason flag parsing
		// is disabled; the flags are still registered below so that they appear
		// in help output.
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseCallArgs(args)
			if err != nil {
				return &UsageError{Message: err.Error()}
			}
			if parsed.help {
				cmd.HelpFunc()(cmd, nil)
				return nil
			}
			if parsed.jsonOutput && parsed.raw {
				return &UsageError{Message: "--json and --raw cannot be used together."}
			}
			if len(parsed.positionals) < 2 || len(parsed.positionals) > 3 {
				return &UsageError{Message: "Incorrect number of arguments."}
			}

			url, toolName := parsed.positionals[0], parsed.positionals[1]
			var argumentsJSON *string
			if len(parsed.positionals) == 3 {
				value := parsed.positionals[2]
				if value == "-" {
					data, err := io.ReadAll(cmd.InOrStdin())
					if err != nil {
						return err
					}
					value = string(data)
				}
				argumentsJSON = &value
			}

			protocolVersion := statelessProtocolVersion
			if parsed.legacy {
				protocolVersion = legacyProtocolVersion
			}

			client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: protocolVersion, Context: cmd.Context()})
			if err != nil {
				return err
			}
			defer client.Close()

			result, err := invokeTool(cmd.Context(), client, toolName, argumentsJSON, parsed.pairs)
			if err != nil {
				var usageErr *UsageError
				if errors.As(err, &usageErr) {
					return usageErr
				}
				return err
			}

			switch {
			case parsed.raw:
				if err := writeJSON(cmd, result); err != nil {
					return err
				}
			case parsed.jsonOutput:
				if err := writeCallJSON(cmd, result); err != nil {
					return err
				}
			default:
				if err := writeText(cmd, renderCallResult(result)); err != nil {
					return err
				}
			}
			if result.IsError {
				return &ExitCodeError{Code: 1}
			}
			return nil
		},
	}

	// Flags are registered only for help output; parseCallArgs performs the
	// actual parsing (and would be bypassed if cobra parsed these).
	cmd.Flags().VarP(&argumentPairFlag{}, "argument", "a",
		"Set one tool argument; repeat for multiple arguments.")
	cmd.Flags().Bool("raw", false, "Output the complete MCP CallToolResult as JSON.")
	cmd.Flags().Bool("json", false, "Output JSON.")
	cmd.Flags().Bool("legacy", false, "Force the legacy initialize handshake protocol.")
	cmd.Flags().Bool("stateless", false, "Force the stateless MCP server/discover protocol (default).")
	return cmd
}

// argumentPairFlag is a pflag value used only to render the -a/--argument
// option in help output with a NAME VALUE metavar. It is never parsed: flag
// parsing is disabled on the call command, and parseCallArgs handles -a.
type argumentPairFlag struct{}

func (argumentPairFlag) String() string { return "" }

func (argumentPairFlag) Set(string) error { return nil }

func (argumentPairFlag) Type() string { return "NAME VALUE" }

func newInfoCommand() *cobra.Command {
	var common commonOptions
	cmd := &cobra.Command{
		Use:   "info URL",
		Short: "Show protocol and metadata for an MCP server at URL.",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := fetchServerInfo(cmd.Context(), args[0], common.protocolVersion())
			if err != nil {
				return err
			}
			if common.jsonOutput {
				return writeJSON(cmd, info)
			}
			return writeText(cmd, renderInfo(info))
		},
	}
	addCommonFlags(cmd, &common)
	return cmd
}

func newDoctorCommand() *cobra.Command {
	var common commonOptions
	cmd := &cobra.Command{
		Use:   "doctor URL",
		Short: "Check stateless and legacy compatibility for an MCP server at URL.",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			report := doctorServer(cmd.Context(), args[0], common.isStateless())
			if common.jsonOutput {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeText(cmd, renderDoctor(report)); err != nil {
					return err
				}
			}
			if !report.Healthy {
				return &ExitCodeError{Code: 1}
			}
			return nil
		},
	}
	addCommonFlags(cmd, &common)
	return cmd
}

// writeCallJSON writes the --json form of a call result.
func writeCallJSON(cmd *cobra.Command, result *mcp.CallToolResult) error {
	if result.StructuredContent != nil {
		return writeJSON(cmd, result.StructuredContent)
	}
	if len(result.Content) > 0 {
		if text, ok := result.Content[0].(*mcp.TextContent); ok {
			fmt.Fprintln(cmd.OutOrStdout(), text.Text)
			return nil
		}
	}
	fmt.Fprintln(cmd.OutOrStdout(), "null")
	return nil
}
