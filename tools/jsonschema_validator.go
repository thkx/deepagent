//go:build jsonschema
// +build jsonschema

package tools

import (
    "bytes"
    "encoding/json"
    "fmt"

    jsonschema "github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidateAgainstJSONSchema validates args against a full JSON Schema using
// github.com/santhosh-tekuri/jsonschema/v5. This file is built only when the
// `jsonschema` build tag is set. Without that tag, a stub implementation is
// used (see jsonschema_validator_stub.go).
func ValidateAgainstJSONSchema(schema map[string]any, args map[string]any) error {
    if schema == nil {
        return nil
    }

    b, err := json.Marshal(schema)
    if err != nil {
        return fmt.Errorf("failed to marshal schema: %w", err)
    }

    compiler := jsonschema.NewCompiler()
    const schemaName = "__tool_schema.json"
    if err := compiler.AddResource(schemaName, bytes.NewReader(b)); err != nil {
        return fmt.Errorf("failed to add schema resource: %w", err)
    }

    sch, err := compiler.Compile(schemaName)
    if err != nil {
        return fmt.Errorf("failed to compile json schema: %w", err)
    }

    if err := sch.ValidateInterface(args); err != nil {
        return fmt.Errorf("json schema validation error: %w", err)
    }
    return nil
}
