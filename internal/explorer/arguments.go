package explorer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// parseArgumentValue interprets a raw command-line value according to the
// parameter's JSON schema: strings remain literal, while other values are
// parsed as JSON when the schema requires it.
func parseArgumentValue(name, value string, schema, root map[string]any) (any, error) {
	if schemaAllowsString(schema, root) {
		return value, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(value), &parsed); err == nil {
		return parsed, nil
	}
	if schemaRequiresJSON(schema, root) {
		expected := shortSchemaType(resolveLocalRef(schema, root))
		return nil, &UsageError{
			Message: fmt.Sprintf("Argument '%s' must be valid JSON for type %s", name, expected),
		}
	}
	return value, nil
}

// buildArguments combines raw JSON arguments with individual -a arguments and
// validates the result against the tool's input schema.
func buildArguments(inputSchema map[string]any, argumentsJSON *string, argumentPairs [][2]string) (map[string]any, error) {
	var arguments map[string]any
	if argumentsJSON == nil {
		arguments = map[string]any{}
	} else {
		var parsed any
		if err := json.Unmarshal([]byte(*argumentsJSON), &parsed); err != nil {
			return nil, &UsageError{
				Message: fmt.Sprintf("Raw arguments must be valid JSON: %s", jsonErrorText(err)),
			}
		}
		var ok bool
		arguments, ok = parsed.(map[string]any)
		if !ok {
			return nil, &UsageError{Message: "Raw arguments must be a JSON object"}
		}
	}

	properties := asMap(inputSchema["properties"])
	additionalProperties := inputSchema["additionalProperties"]
	for _, pair := range argumentPairs {
		name, value := pair[0], pair[1]
		parameterSchema := asMap(properties[name])
		if parameterSchema == nil {
			parameterSchema = asMap(additionalProperties)
			if parameterSchema == nil {
				parameterSchema = map[string]any{}
			}
		}
		parsed, err := parseArgumentValue(name, value, parameterSchema, inputSchema)
		if err != nil {
			return nil, err
		}
		arguments[name] = parsed
	}

	if err := validateArguments(inputSchema, arguments); err != nil {
		return nil, err
	}
	return arguments, nil
}

// jsonErrorText returns the message portion of a json syntax error.
func jsonErrorText(err error) string {
	return err.Error()
}

// validateArguments checks arguments against the input schema.
func validateArguments(inputSchema map[string]any, arguments map[string]any) error {
	// Normalize an unsupported $schema URI so legacy server schemas still
	// validate. The jsonschema-go package understands draft-07 and draft
	// 2020-12; other URIs default to 2020-12 when omitted.
	if v, ok := inputSchema["$schema"].(string); ok {
		switch v {
		case "http://json-schema.org/draft-07/schema#",
			"https://json-schema.org/draft-07/schema#",
			"https://json-schema.org/draft/2020-12/schema":
			// supported as-is
		case "https://json-schema.org/draft/2020-12/schema#":
			inputSchema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		default:
			delete(inputSchema, "$schema")
		}
	}

	data, err := json.Marshal(inputSchema)
	if err != nil {
		return &UsageError{Message: fmt.Sprintf("Tool input schema is invalid: %s", err)}
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return &UsageError{Message: fmt.Sprintf("Tool input schema is invalid: %s", err)}
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return &UsageError{Message: fmt.Sprintf("Tool input schema is invalid: %s", err)}
	}
	if err := resolved.Validate(&arguments); err != nil {
		message := strings.TrimSpace(err.Error())
		return &UsageError{Message: "Arguments do not match input schema: " + message}
	}
	return nil
}
