package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// localURLPrefix is the route the API serves stored files under in disk mode.
const localURLPrefix = "/uploads"

// localStorage writes objects to a directory on disk. Files are served by the
// API at localURLPrefix.
type localStorage struct {
	root string
}

func newLocal(dir string) (*localStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &localStorage{root: dir}, nil
}

func (l *localStorage) Put(_ context.Context, key, _ string, data io.Reader, _ int64) (string, error) {
	path := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create object dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create object file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("write object: %w", err)
	}
	return localURLPrefix + "/" + strings.TrimPrefix(key, "/"), nil
}

func (l *localStorage) Delete(_ context.Context, key string) error {
	path := filepath.Join(l.root, filepath.FromSlash(key))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// Root is the base directory, exposed so the API can mount a file server on it.
func (l *localStorage) Root() string { return l.root }
