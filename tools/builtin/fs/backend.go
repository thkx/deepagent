package fs

import "context"

type Backend interface {
	List(ctx context.Context, path string) ([]string, error)
	Read(ctx context.Context, path string) (string, error)
	Write(ctx context.Context, path, content string) error
	Edit(ctx context.Context, path, instructions string) error
}
