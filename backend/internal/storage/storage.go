// Package storage abstracts object storage behind a small interface with two
// implementations: local disk (dev default) and S3-compatible (Cloudflare R2 by
// default). The rest of the app depends only on the interface.
package storage

import (
	"context"
	"io"

	"github.com/morfostech/morfos-finance/internal/config"
)

// Storage persists opaque objects addressed by key and returns a URL to reach
// them.
type Storage interface {
	// Put stores data under key and returns its retrievable URL.
	Put(ctx context.Context, key, contentType string, data io.Reader, size int64) (url string, err error)
	// Delete removes the object at key. Missing objects are not an error.
	Delete(ctx context.Context, key string) error
}

// New builds the configured backend: S3-compatible when configured, else local
// disk. LocalServePath (non-empty for disk) is the URL prefix the API must serve
// files under.
func New(cfg config.StorageConfig) (s Storage, localServePath string, err error) {
	if cfg.UseS3() {
		s, err = newS3(cfg)
		return s, "", err
	}
	local, err := newLocal(cfg.Dir)
	if err != nil {
		return nil, "", err
	}
	return local, localURLPrefix, nil
}
