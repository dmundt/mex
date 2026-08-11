# mcp-explorer

[![CI](https://github.com/dmundt/mcp-explorer/actions/workflows/ci.yml/badge.svg)](https://github.com/dmundt/mcp-explorer/actions/workflows/ci.yml)

A Go command-line tool for exploring, inspecting, and invoking the tools,
prompts, and resources exposed by [Model Context Protocol (MCP)][mcp] servers,
over Streamable HTTP or stdio.

This is a Go port of [Simon Willison's `mcp-explorer`][ref], aiming for parity
in output formatting and exit behavior.

## Installation

```bash
go install github.com/dmundt/mcp-explorer/cmd/mcp-explorer@latest
```

## Usage

Every command takes a server URL (or a local stdio command line) as its
first positional argument.

```
mcp-explorer list URL
mcp-explorer prompts URL
mcp-explorer resources URL
mcp-explorer inspect URL TOOL_NAME
mcp-explorer call URL TOOL_NAME [ARGUMENTS_JSON]
mcp-explorer info URL
mcp-explorer doctor URL
```

Common options:

- `--json` - Output JSON instead of the human-readable rendering.
- `--stateless` - Force the stateless MCP 2 `server/discover` protocol (default).
- `--legacy` - Force the legacy `initialize` handshake protocol.

### list

List the tools exposed by an MCP server.

```
mcp-explorer list https://example.com/mcp
```

Adding `-N`/`--no-truncate` shows full descriptions and detailed parameters.

### prompts

List every prompt exposed by an MCP server.

### resources

List every directly-addressable resource exposed by an MCP server.

### inspect

Inspect a single tool in detail, including its input schema, output schema,
annotations, icons, and metadata.

### call

Call a tool with arguments from a JSON object and/or individual `-a` pairs.

```
mcp-explorer call https://example.com/mcp my-tool '{"query": "hello"}'
mcp-explorer call URL my-tool -a query hello -a limit 5
```

If `ARGUMENTS_JSON` is `-`, the JSON is read from standard input. Arguments
provided with `-a NAME VALUE` override keys in the JSON object. The `--json`
flag prints only the call result; `--raw` prints the complete
`CallToolResult`. A tool result marked as an error exits with code 1.

### info

Show protocol and server metadata for the selected mode: mode, negotiation
mechanism, negotiated protocol version, supported versions, server identity,
capabilities, and instructions.

### doctor

Check whether an MCP server is compatible with both stateless and legacy
modes, comparing latency, tool counts, and pagination behavior for each.
Exits with code 1 when the selected mode reports an error.

## Development

```bash
go build ./...
go test ./...
go vet ./...
```

The integration tests exercise the client against an in-memory MCP server
from the Go SDK.

## License

MIT. See [LICENSE](LICENSE).

[mcp]: https://modelcontextprotocol.io/
[ref]: https://github.com/simonw/mcp-explorer