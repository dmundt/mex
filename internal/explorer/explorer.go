// Package explorer implements the mcp-explorer command line tool.
//
// It explores, inspects, and invokes tools, prompts, and resources exposed by
// Model Context Protocol (MCP) servers.
package explorer

import (
	"github.com/spf13/cobra"
)

const (
	// packageName is the display name used in help output and as the client
	// identity announced to MCP servers.
	packageName = "mcp-explorer"
	// version is the client version announced to MCP servers.
	version = "0.1.0"
)

// UsageError describes invalid command-line usage. It exits with code 2.
type UsageError struct {
	Message string
}

// Error implements the error interface.
func (e *UsageError) Error() string { return e.Message }

// Execute runs the mcp-explorer command line interface.
func Execute() error {
	return newRootCommand().Execute()
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     packageName,
		Short:   "CLI tool for exploring MCP servers",
		Long:    "mcp-explorer explores, inspects, and invokes the tools, prompts, and resources exposed by Model Context Protocol (MCP) servers.",
		Version: version,
	}
	root.AddCommand(
		newListCommand(),
		newPromptsCommand(),
		newResourcesCommand(),
		newInspectCommand(),
		newCallCommand(),
		newInfoCommand(),
		newDoctorCommand(),
	)
	return root
}

// commonOptions holds the flags shared by every command.
type commonOptions struct {
	jsonOutput bool
	stateless  bool
	legacy     bool
}

// isStateless reports whether to force the stateless MCP 2 protocol. Stateless
// is the default; --legacy forces the older initialize handshake.
func (o *commonOptions) isStateless() bool {
	return !o.legacy
}

// protocolVersion returns the protocol version to request, based on the
// selected mode.
func (o *commonOptions) protocolVersion() string {
	if o.legacy {
		return legacyProtocolVersion
	}
	return statelessProtocolVersion
}

// addCommonFlags adds the flags shared by every command.
func addCommonFlags(cmd *cobra.Command, o *commonOptions) {
	cmd.Flags().BoolVar(&o.jsonOutput, "json", false, "Output JSON.")
	cmd.Flags().BoolVar(&o.stateless, "stateless", false, "Force the stateless MCP server/discover protocol (default).")
	cmd.Flags().BoolVar(&o.legacy, "legacy", false, "Force the legacy initialize handshake protocol.")
}

// exactArgs wraps cobra.ExactArgs so that argument-count misuse reports an
// exit code of 2, matching the usage-error convention.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return &UsageError{Message: err.Error()}
		}
		return nil
	}
}
