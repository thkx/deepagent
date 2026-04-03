//go:build !redis
// +build !redis

package memory

import "fmt"

func newRedisStore(dsn string) (Store, error) {
    return nil, fmt.Errorf("redis backend not enabled: build with -tags=redis to enable")
}
