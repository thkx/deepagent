package tools

// This is the default (no-op / lightweight fallback) implementation of
// ValidateAgainstJSONSchema. It delegates to the existing lightweight
// ValidateAgainstSchema logic so builds without the `jsonschema` tag or
// without network access continue to work.
func ValidateAgainstJSONSchema(schema map[string]any, args map[string]any) error {
    return ValidateAgainstSchema(schema, args)
}
