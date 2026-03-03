package utilities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath(t *testing.T) {
	// Create a temp workspace
	workspace := t.TempDir()
	subdir := filepath.Join(workspace, "src")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	// Create a test file
	testFile := filepath.Join(subdir, "main.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0644))

	t.Run("valid path within workspace", func(t *testing.T) {
		result, err := ValidatePath(workspace, testFile)
		assert.NoError(t, err)
		assert.Equal(t, testFile, result)
	})

	t.Run("valid relative path within workspace", func(t *testing.T) {
		// Change to workspace dir for relative path test
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(workspace)

		result, err := ValidatePath(workspace, "src/main.go")
		assert.NoError(t, err)
		assert.Equal(t, testFile, result)
	})

	t.Run("workspace root itself is valid", func(t *testing.T) {
		result, err := ValidatePath(workspace, workspace)
		assert.NoError(t, err)
		assert.Equal(t, workspace, result)
	})

	t.Run("rejects absolute path outside workspace", func(t *testing.T) {
		_, err := ValidatePath(workspace, "/etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside workspace")
	})

	t.Run("rejects relative traversal outside workspace", func(t *testing.T) {
		_, err := ValidatePath(workspace, filepath.Join(workspace, "..", "..", "etc", "passwd"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside workspace")
	})

	t.Run("rejects dot-dot traversal", func(t *testing.T) {
		_, err := ValidatePath(workspace, filepath.Join(workspace, "src", "..", "..", "outside"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside workspace")
	})

	t.Run("allows nonexistent file within workspace", func(t *testing.T) {
		newFile := filepath.Join(subdir, "new.go")
		result, err := ValidatePath(workspace, newFile)
		assert.NoError(t, err)
		assert.Equal(t, newFile, result)
	})

	t.Run("empty workspace skips validation", func(t *testing.T) {
		result, err := ValidatePath("", "/any/path")
		assert.NoError(t, err)
		assert.Equal(t, "/any/path", result)
	})

	t.Run("rejects symlink pointing outside workspace", func(t *testing.T) {
		// Create a symlink inside workspace pointing outside
		outsideDir := t.TempDir()
		outsideFile := filepath.Join(outsideDir, "secret.txt")
		require.NoError(t, os.WriteFile(outsideFile, []byte("secret"), 0644))

		symlink := filepath.Join(subdir, "sneaky_link")
		require.NoError(t, os.Symlink(outsideFile, symlink))

		_, err := ValidatePath(workspace, symlink)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside workspace")
	})

	t.Run("allows symlink within workspace", func(t *testing.T) {
		symlink := filepath.Join(subdir, "link_to_main")
		os.Remove(symlink) // clean up if exists
		require.NoError(t, os.Symlink(testFile, symlink))

		result, err := ValidatePath(workspace, symlink)
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

func TestValidateURI(t *testing.T) {
	workspace := t.TempDir()
	testFile := filepath.Join(workspace, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package test"), 0644))

	t.Run("valid file URI", func(t *testing.T) {
		result, err := ValidateURI(workspace, "file://"+testFile)
		assert.NoError(t, err)
		assert.Equal(t, testFile, result)
	})

	t.Run("rejects file URI outside workspace", func(t *testing.T) {
		_, err := ValidateURI(workspace, "file:///etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside workspace")
	})

	t.Run("handles path without file prefix", func(t *testing.T) {
		result, err := ValidateURI(workspace, testFile)
		assert.NoError(t, err)
		assert.Equal(t, testFile, result)
	})
}

func TestIsSubpath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{"child under parent", "/workspace", "/workspace/src/file.go", true},
		{"child is parent", "/workspace", "/workspace", true},
		{"child outside parent", "/workspace", "/etc/passwd", false},
		{"partial prefix match", "/workspace", "/workspace-extra/file.go", false},
		{"parent with trailing slash", "/workspace/", "/workspace/file.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSubpath(tt.parent, tt.child))
		})
	}
}
