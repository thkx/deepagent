package memory

import (
    "fmt"
)

// NewMemoryStore creates a memory store of the requested kind.
// Supported kinds: "file" (default), "sqlite", "redis".
// For backends that are not built into the binary, an explanatory error is returned.
func NewMemoryStore(kind string, dsnOrDir string) (Store, error) {
    switch kind {
    case "", "file":
        return NewFileMemoryStore(dsnOrDir), nil
    case "sqlite":
        return newSQLiteStore(dsnOrDir)
    case "redis":
        return newRedisStore(dsnOrDir)
    case "redis-inmem":
        return NewInMemoryRedisStore(), nil
    default:
        return nil, fmt.Errorf("unknown memory backend kind: %s", kind)
    }
}
