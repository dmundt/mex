package explorer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// marshalJSONIndent marshals v as indented JSON, without HTML escaping, like
// Python's json.dumps(v, indent=2).
func marshalJSONIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// marshalJSONCompact marshals v as compact JSON, without HTML escaping.
func marshalJSONCompact(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimRight(buf.String(), "\n")
}

// firstDescriptionLine returns the first non-empty line of a description.
func firstDescriptionLine(description string) string {
	for _, line := range fullDescriptionLines(description) {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// fullDescriptionLines returns the non-empty-trimmed lines of a description.
func fullDescriptionLines(description string) []string {
	description = strings.ReplaceAll(description, "\r\n", "\n")
	description = strings.TrimSpace(description)
	if description == "" {
		return nil
	}
	return strings.Split(description, "\n")
}

// splitLines splits text on newlines, dropping any trailing empty segment, in
// the spirit of Python's str.splitlines.
func splitLines(description string) []string {
	description = strings.ReplaceAll(description, "\r\n", "\n")
	lines := strings.Split(description, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// toolHeading renders the display heading for a tool.
func toolHeading(tool *mcp.Tool, includeSignature bool) string {
	heading := tool.Name
	if includeSignature {
		schema := toolInputSchema(tool)
		properties := asMap(schema["properties"])
		required := requiredSet(schema)
		var parameters []string
		for _, name := range sortedKeys(properties) {
			optional := ""
			if !required[name] {
				optional = "?"
			}
			parameters = append(parameters, fmt.Sprintf("%s%s: %s", name, optional, shortSchemaType(asMap(properties[name]))))
		}
		heading = fmt.Sprintf("%s(%s)", heading, strings.Join(parameters, ", "))
	}
	if tool.Title != "" {
		heading = heading + " - " + tool.Title
	}
	return heading
}

// renderParameterDetails renders a parameter's type label, requirement, and
// full description.
func renderParameters(inputSchema map[string]any) string {
	var b strings.Builder
	properties := asMap(inputSchema["properties"])
	if len(properties) == 0 {
		b.WriteString("  Parameters: none\n")
		return b.String()
	}

	b.WriteString("  Parameters:\n")
	required := requiredSet(inputSchema)
	for _, name := range sortedKeys(properties) {
		requirement := "optional"
		if required[name] {
			requirement = "required"
		}
		schema := asMap(properties[name])
		fmt.Fprintf(&b, "    %s (%s, %s)\n", name, schemaType(schema), requirement)
		desc, _ := schema["description"].(string)
		for _, line := range fullDescriptionLines(desc) {
			if strings.TrimSpace(line) == "" {
				b.WriteString("\n")
			} else {
				fmt.Fprintf(&b, "      %s\n", line)
			}
		}
	}
	return b.String()
}

// renderJSONSection renders a labeled indented-JSON section, or "" if value is
// nil.
func renderJSONSection(label string, value any) string {
	if value == nil {
		return ""
	}
	data, err := marshalJSONIndent(value)
	if err != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(label + ":\n")
	for _, line := range strings.Split(string(data), "\n") {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	return b.String()
}

// renderToolList renders the human-readable output of the list command.
func renderToolList(tools []*mcp.Tool, noTruncate bool) string {
	if len(tools) == 0 {
		return "No tools available.\n"
	}

	var b strings.Builder
	for index, tool := range tools {
		if index > 0 {
			b.WriteString("\n")
		}
		if noTruncate {
			b.WriteString(toolHeading(tool, false) + "\n")
			for _, line := range fullDescriptionLines(tool.Description) {
				if strings.TrimSpace(line) == "" {
					b.WriteString("\n")
				} else {
					fmt.Fprintf(&b, "  %s\n", line)
				}
			}
			b.WriteString(renderParameters(toolInputSchema(tool)))
		} else {
			b.WriteString(toolHeading(tool, true) + "\n")
			if summary := firstDescriptionLine(tool.Description); summary != "" {
				fmt.Fprintf(&b, "  %s\n", summary)
			}
		}
	}
	return b.String()
}

// renderPromptList renders the human-readable output of the prompts command.
func renderPromptList(prompts []*mcp.Prompt) string {
	if len(prompts) == 0 {
		return "No prompts available.\n"
	}

	var b strings.Builder
	for index, prompt := range prompts {
		if index > 0 {
			b.WriteString("\n")
		}
		heading := prompt.Name
		if prompt.Title != "" {
			heading = heading + " - " + prompt.Title
		}
		b.WriteString(heading + "\n")
		if summary := firstDescriptionLine(prompt.Description); summary != "" {
			fmt.Fprintf(&b, "  %s\n", summary)
		}

		if len(prompt.Arguments) == 0 {
			b.WriteString("  Arguments: none\n")
			continue
		}
		b.WriteString("  Arguments:\n")
		for _, argument := range prompt.Arguments {
			requirement := "optional"
			if argument.Required {
				requirement = "required"
			}
			argumentHeading := argument.Name
			if argument.Title != "" {
				argumentHeading = argumentHeading + " - " + argument.Title
			}
			fmt.Fprintf(&b, "    %s (%s)\n", argumentHeading, requirement)
			if description := firstDescriptionLine(argument.Description); description != "" {
				fmt.Fprintf(&b, "      %s\n", description)
			}
		}
	}
	return b.String()
}

// renderResourceList renders the human-readable output of the resources
// command.
func renderResourceList(resources []*mcp.Resource) string {
	if len(resources) == 0 {
		return "No resources available.\n"
	}

	var b strings.Builder
	for index, resource := range resources {
		if index > 0 {
			b.WriteString("\n")
		}
		heading := resource.Name
		if resource.Title != "" {
			heading = heading + " - " + resource.Title
		}
		b.WriteString(heading + "\n")
		fmt.Fprintf(&b, "  %s\n", resource.URI)
		if summary := firstDescriptionLine(resource.Description); summary != "" {
			fmt.Fprintf(&b, "  %s\n", summary)
		}
		if resource.MIMEType != "" {
			fmt.Fprintf(&b, "  MIME type: %s\n", resource.MIMEType)
		}
		if resource.Size != 0 {
			fmt.Fprintf(&b, "  Size: %d bytes\n", resource.Size)
		}
	}
	return b.String()
}

// renderToolInspect renders the human-readable output of the inspect command.
func renderToolInspect(tool *mcp.Tool) string {
	var b strings.Builder
	b.WriteString(toolHeading(tool, false) + "\n")
	for _, line := range fullDescriptionLines(tool.Description) {
		if strings.TrimSpace(line) == "" {
			b.WriteString("\n")
		} else {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	schema := toolInputSchema(tool)
	b.WriteString(renderJSONSection("Input schema", schema))
	b.WriteString(renderJSONSection("Output schema", tool.OutputSchema))
	b.WriteString(renderJSONSection("Annotations", tool.Annotations))
	b.WriteString(renderJSONSection("Icons", tool.Icons))
	b.WriteString(renderJSONSection("Metadata", tool.Meta))
	return b.String()
}

// renderInfo renders the human-readable output of the info command.
func renderInfo(info Info) string {
	var b strings.Builder
	fmt.Fprintf(&b, "URL: %s\n", info.URL)
	fmt.Fprintf(&b, "Mode: %s\n", info.Mode)
	fmt.Fprintf(&b, "Negotiation: %s\n", info.Negotiation)
	fmt.Fprintf(&b, "Protocol version: %s\n", info.ProtocolVersion)
	if len(info.SupportedVersions) > 0 {
		fmt.Fprintf(&b, "Supported versions: %s\n", strings.Join(info.SupportedVersions, ", "))
	}
	b.WriteString(renderJSONSection("Server info", info.ServerInfo))
	b.WriteString(renderJSONSection("Capabilities", info.Capabilities))
	if info.Instructions != "" {
		b.WriteString("Instructions:\n")
		for _, line := range splitLines(info.Instructions) {
			if line == "" {
				b.WriteString("\n")
			} else {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}
	return b.String()
}

// renderCallResult renders the human-readable output of the call command.
func renderCallResult(result *mcp.CallToolResult) string {
	var b strings.Builder
	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			b.WriteString(c.Text + "\n")
		case *mcp.ImageContent:
			fmt.Fprintf(&b, "[image: %s, %d base64 characters]\n", c.MIMEType, len(c.Data))
		case *mcp.AudioContent:
			fmt.Fprintf(&b, "[audio: %s, %d base64 characters]\n", c.MIMEType, len(c.Data))
		case *mcp.ResourceLink:
			fmt.Fprintf(&b, "[resource: %s (%s)]\n", c.Name, c.URI)
		case *mcp.EmbeddedResource:
			resource := c.Resource
			if resource == nil {
				b.WriteString("[resource: (unknown)]\n")
			} else if len(resource.Blob) == 0 {
				b.WriteString(resource.Text + "\n")
			} else {
				mimeType := resource.MIMEType
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
				fmt.Fprintf(&b, "[resource: %s, %s, %d base64 characters]\n", resource.URI, mimeType, len(resource.Blob))
			}
		default:
			b.WriteString(renderJSONSection("Content", content))
		}
	}
	if result.StructuredContent != nil {
		b.WriteString(renderJSONSection("Structured content", result.StructuredContent))
	}
	return b.String()
}
