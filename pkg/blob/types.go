package blob

import (
	"context"
	"io"
)

type BlobStore interface {
	// Put uploads content to the blob store.
	Put(ctx context.Context, key string, reader io.Reader) error

	// Get retrieves content from the blob store.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// List returns a list of keys matching the prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes a blob.
	Delete(ctx context.Context, key string) error
}
