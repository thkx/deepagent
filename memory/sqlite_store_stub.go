//go:build !sqlite
// +build !sqlite

package memory

import "fmt"

func newSQLiteStore(dsn string) (Store, error) {
    return nil, fmt.Errorf("sqlite backend not enabled: build with -tags=sqlite to enable")
}
