package objectstore

import (
	"context"
	"io"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error)
	Stat(ctx context.Context, key string) (size int64, etag string, contentType string, err error)
	List(ctx context.Context, prefix string) ([]string, error)
}
