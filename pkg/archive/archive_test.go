package archive

import (
	"os"
	"testing"
)

func TestNewWriter(t *testing.T) {
	tmpfile := "/tmp/test-archive.zip"
	defer os.Remove(tmpfile)

	arc, err := NewWriter(tmpfile)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	// Add a test file
	testContent := []byte("test content")
	err = arc.Add("test.json", testContent)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Close the archive
	err = arc.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpfile); os.IsNotExist(err) {
		t.Fatal("Archive file was not created")
	}
}
