package explorer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// discoverResponse is the shape of a server/discover result.
type discoverResponse struct {
	SupportedVersions []string                `json:"supportedVersions"`
	Capabilities      *mcp.ServerCapabilities `json:"capabilities"`
	Instructions      string                  `json:"instructions"`
}

// fetchServerInfo returns protocol and server metadata for the selected mode.
func fetchServerInfo(ctx context.Context, url string, protocolVersion string) (Info, error) {
	client, err := NewClient(ClientOptions{URL: url, ProtocolVersion: protocolVersion, Context: ctx})
	if err != nil {
		return Info{}, err
	}
	defer client.Close()

	info := client.Info()
	if info.Mode == "stateless" {
		if supported, err := rawDiscoverSupportedVersions(ctx, url, protocolVersion); err == nil {
			info.SupportedVersions = supported
		}
	}
	return info, nil
}

// rawDiscoverSupportedVersions sends a bare server/discover request over the
// transport and returns the versions the server supports. Stateless servers
// answer such a request directly, without any session handshake.
func rawDiscoverSupportedVersions(ctx context.Context, url, requestedVersion string) ([]string, error) {
	transport, err := newTransport(url)
	if err != nil {
		return nil, err
	}
	conn, err := transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	params, err := json.Marshal(map[string]any{
		"_meta": map[string]any{
			"protocolVersion": requestedVersion,
			"clientInfo": map[string]any{
				"name":    packageName,
				"version": version,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	id, err := jsonrpc.MakeID(float64(1))
	if err != nil {
		return nil, err
	}
	req := &jsonrpc.Request{ID: id, Method: "server/discover", Params: params}
	if err := conn.Write(ctx, req); err != nil {
		return nil, fmt.Errorf("sending discover: %w", err)
	}

	for {
		msg, err := conn.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("reading discover response: %w", err)
		}
		resp, ok := msg.(*jsonrpc.Response)
		if !ok {
			// Ignore any server-initiated requests on this probing connection.
			continue
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		var result discoverResponse
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, fmt.Errorf("decoding discover result: %w", err)
		}
		return result.SupportedVersions, nil
	}
}
