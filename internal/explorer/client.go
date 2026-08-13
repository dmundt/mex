package explorer

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Protocol versions requested by the two connection modes. Stateless mode
// requests the latest modern (MCP 2) version, which the SDK negotiates via the
// server/discover RPC, falling back to the legacy initialize handshake when the
// server does not support it. Legacy mode requests the latest legacy version,
// which forces the initialize handshake.
const (
	statelessProtocolVersion = "2026-07-28"
	legacyProtocolVersion    = "2025-11-25"
)

// httpClient is the shared HTTP client used for communicating with local
// streamable HTTP MCP servers. Keeping idle connections alive allows a single
// invocation to complete multiple requests over one connection.
var httpClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

// Info describes a connected MCP server and the protocol mode used to reach it.
type Info struct {
	URL               string                  `json:"url"`
	Mode              string                  `json:"mode"`
	Negotiation       string                  `json:"negotiation"`
	ProtocolVersion   string                  `json:"protocolVersion"`
	SupportedVersions []string                `json:"supportedVersions"`
	ServerInfo        *mcp.Implementation     `json:"serverInfo"`
	Capabilities      *mcp.ServerCapabilities `json:"capabilities"`
	Instructions      string                  `json:"instructions"`
}

// ClientOptions configures how a client connects to an MCP server.
type ClientOptions struct {
	// URL is a streamable HTTP URL or a local stdio command line.
	URL string
	// ProtocolVersion is the MCP protocol version to request. If empty, the
	// SDK's latest version is used.
	ProtocolVersion string
	// Context is the connection's context, used for the connect handshake.
	// If nil, context.Background is used.
	Context context.Context
}

// MCPClient is the interface used by commands to interact with an MCP server.
// It is abstracted so that tests can substitute a fake implementation.
type MCPClient interface {
	// Info returns protocol and server metadata for the connected session.
	Info() Info
	// ListToolsPage returns one page of tools starting at the given cursor. The
	// second return value is the next cursor, or "" when there are no more pages.
	ListToolsPage(ctx context.Context, cursor string) ([]*mcp.Tool, string, error)
	// ListPromptsPage returns one page of prompts starting at the given cursor.
	ListPromptsPage(ctx context.Context, cursor string) ([]*mcp.Prompt, string, error)
	// ListResourcesPage returns one page of resources starting at the given cursor.
	ListResourcesPage(ctx context.Context, cursor string) ([]*mcp.Resource, string, error)
	// FindTool returns the named tool, or nil if it does not exist.
	FindTool(ctx context.Context, name string) (*mcp.Tool, error)
	// CallTool invokes the named tool with the given arguments.
	CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error)
	// Close terminates the connection.
	Close() error
}

// mcpClient is an MCPClient backed by the go-sdk.
type mcpClient struct {
	session *mcp.ClientSession
	info    Info
}

// Connect establishes a session with an MCP server over the given transport.
func Connect(ctx context.Context, transport mcp.Transport, opts ClientOptions) (MCPClient, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    packageName,
		Version: version,
	}, nil)

	session, err := client.Connect(ctx, transport, &mcp.ClientSessionOptions{
		ProtocolVersion: opts.ProtocolVersion,
	})
	if err != nil {
		return nil, err
	}

	mc := &mcpClient{session: session}
	mc.info = buildInfo(mc.session, opts.URL, opts.ProtocolVersion)
	return mc, nil
}

// buildInfo derives the Info snapshot after a successful connection.
func buildInfo(session *mcp.ClientSession, url, requestedVersion string) Info {
	info := Info{
		URL:             url,
		Mode:            "legacy",
		Negotiation:     "initialize",
		ProtocolVersion: requestedVersion,
	}
	if ir := session.InitializeResult(); ir != nil {
		info.ProtocolVersion = ir.ProtocolVersion
		info.ServerInfo = ir.ServerInfo
		info.Capabilities = ir.Capabilities
		info.Instructions = ir.Instructions
	}
	// The mode reflects what was actually negotiated, not just what was
	// requested: a stateless request falls back to the legacy initialize
	// handshake when the server does not support server/discover.
	info.Mode, info.Negotiation = modeFromVersion(info.ProtocolVersion)
	return info
}

// modeFromVersion returns the mode and negotiation mechanism corresponding to a
// negotiated protocol version.
func modeFromVersion(version string) (mode, negotiation string) {
	if version >= statelessProtocolVersion {
		return "stateless", "server/discover"
	}
	return "legacy", "initialize"
}

// Info implements MCPClient.
func (c *mcpClient) Info() Info { return c.info }

// ListToolsPage implements MCPClient.
func (c *mcpClient) ListToolsPage(ctx context.Context, cursor string) ([]*mcp.Tool, string, error) {
	res, err := c.session.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
	if err != nil {
		return nil, "", err
	}
	return res.Tools, res.NextCursor, nil
}

// ListPromptsPage implements MCPClient.
func (c *mcpClient) ListPromptsPage(ctx context.Context, cursor string) ([]*mcp.Prompt, string, error) {
	res, err := c.session.ListPrompts(ctx, &mcp.ListPromptsParams{Cursor: cursor})
	if err != nil {
		return nil, "", err
	}
	return res.Prompts, res.NextCursor, nil
}

// ListResourcesPage implements MCPClient.
func (c *mcpClient) ListResourcesPage(ctx context.Context, cursor string) ([]*mcp.Resource, string, error) {
	res, err := c.session.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
	if err != nil {
		return nil, "", err
	}
	return res.Resources, res.NextCursor, nil
}

// FindTool implements MCPClient. It follows pagination through listAllTools,
// which fails fast if the server repeats a cursor.
func (c *mcpClient) FindTool(ctx context.Context, name string) (*mcp.Tool, error) {
	tools, err := listAllTools(ctx, c)
	if err != nil {
		return nil, err
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool, nil
		}
	}
	return nil, nil
}

// CallTool implements MCPClient.
func (c *mcpClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*mcp.CallToolResult, error) {
	return c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
}

// Close implements MCPClient.
func (c *mcpClient) Close() error { return c.session.Close() }

// NewClient is the constructor used by all commands. It is a package-level
// variable so that tests can substitute a fake client factory.
var NewClient = func(opts ClientOptions) (MCPClient, error) {
	transport, err := newTransport(opts.URL)
	if err != nil {
		return nil, err
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return Connect(ctx, transport, opts)
}

// isHTTPURL reports whether url identifies a streamable HTTP server.
func isHTTPURL(url string) bool {
	u, err := parseURL(url)
	return err == nil && (u == "http" || u == "https")
}

func parseURL(url string) (scheme string, err error) {
	colon := strings.Index(url, ":")
	if colon < 1 {
		return "", fmt.Errorf("no scheme in URL %q", url)
	}
	scheme = url[:colon]
	for _, r := range scheme {
		if !isSchemeChar(r) {
			return "", fmt.Errorf("invalid scheme in URL %q", url)
		}
	}
	return scheme, nil
}

// isSchemeChar reports whether r is valid in a URL scheme per RFC 3986.
func isSchemeChar(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.'
}

// newTransport builds the SDK transport for the given server, which is either a
// streamable HTTP URL or a local stdio command line.
func newTransport(url string) (mcp.Transport, error) {
	if isHTTPURL(url) {
		return &mcp.StreamableClientTransport{
			Endpoint:   url,
			HTTPClient: httpClient,
		}, nil
	}
	return newCommandTransport(url)
}

func newCommandTransport(commandLine string) (mcp.Transport, error) {
	parts, err := splitCommandLine(commandLine)
	if err != nil || len(parts) == 0 {
		return nil, fmt.Errorf("invalid stdio command %q", commandLine)
	}
	cmd := newCommand(parts[0], parts[1:]...)
	return &mcp.CommandTransport{Command: cmd}, nil
}
