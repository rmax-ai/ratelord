package blob

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// LocalBlobStore implements BlobStore using the local filesystem.
type LocalBlobStore struct {
	rootPath string
}

// NewLocalBlobStore creates a new LocalBlobStore with the given root directory.
func NewLocalBlobStore(rootPath string) *LocalBlobStore {
	return &LocalBlobStore{rootPath: rootPath}
}

// Put uploads content to the blob store.
// It creates the necessary directory structure and uses atomic writes via temp file + rename.
func (s *LocalBlobStore) Put(ctx context.Context, key string, reader io.Reader) error {
	fullPath := filepath.Join(s.rootPath, key)

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create a temporary file in the same directory for atomic write
	tempFile, err := ioutil.TempFile(dir, "temp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Copy data to temp file
	if _, err := io.Copy(tempFile, reader); err != nil {
		os.Remove(tempFile.Name()) // Clean up on error
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written
	if err := tempFile.Sync(); err != nil {
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file
	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile.Name(), fullPath); err != nil {
		os.Remove(tempFile.Name())
		return fmt.Errorf("failed to rename temp file to %s: %w", fullPath, err)
	}

	return nil
}

// Get retrieves content from the blob store.
func (s *LocalBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.rootPath, key)
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob %s not found", key)
		}
		return nil, fmt.Errorf("failed to open blob %s: %w", key, err)
	}
	return file, nil
}

// List returns a list of keys matching the prefix.
func (s *LocalBlobStore) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	root := filepath.Join(s.rootPath, prefix)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Get the key relative to rootPath
			relPath, err := filepath.Rel(s.rootPath, path)
			if err != nil {
				return err
			}
			keys = append(keys, relPath)
		}
		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			// If the prefix directory doesn't exist, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list blobs with prefix %s: %w", prefix, err)
	}

	return keys, nil
}

// Delete removes a blob.
func (s *LocalBlobStore) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.rootPath, key)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("blob %s not found", key)
		}
		return fmt.Errorf("failed to delete blob %s: %w", key, err)
	}
	return nil
}
