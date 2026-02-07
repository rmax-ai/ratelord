package blob

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBlobStore(t *testing.T) {
	// Setup temp dir
	tmpDir, err := ioutil.TempDir("", "blob-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewLocalBlobStore(tmpDir)
	ctx := context.Background()

	// 1. Test Put
	key := "folder/test.txt"
	content := "hello world"
	err = store.Put(ctx, key, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify file exists on disk
	expectedPath := filepath.Join(tmpDir, key)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("File was not created at expected path: %s", expectedPath)
	}

	// 2. Test Get
	reader, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from reader: %v", err)
	}
	if string(data) != content {
		t.Errorf("Get content mismatch. Got %s, want %s", string(data), content)
	}

	// 3. Test List
	// Add another file
	key2 := "folder/other.txt"
	store.Put(ctx, key2, strings.NewReader("other"))

	keys, err := store.List(ctx, "folder")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List returned %d keys, want 2", len(keys))
	}

	// 4. Test Delete
	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("Get should fail after delete")
	}

	// Verify key2 still there
	_, err = store.Get(ctx, key2)
	if err != nil {
		t.Error("Other file should still exist")
	}
}
