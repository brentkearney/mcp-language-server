package utilities

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath checks that a file path is within the workspace boundary.
// It resolves the path to an absolute path, resolves symlinks (if the path
// exists), and confirms the result is under workspaceDir.
//
// Returns the cleaned absolute path if valid, or an error if the path
// escapes the workspace.
func ValidatePath(workspaceDir, filePath string) (string, error) {
	if workspaceDir == "" {
		return filePath, nil // no workspace configured — skip validation
	}

	// Resolve workspace to absolute + resolve symlinks
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}
	realWorkspace, err := filepath.EvalSymlinks(absWorkspace)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace symlinks: %w", err)
	}

	// Resolve the target path to absolute
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// If the file exists, resolve symlinks to get the real path.
	// If it doesn't exist (e.g., CreateFile), resolve the parent directory
	// and append the filename.
	var realPath string
	if _, err := os.Stat(absPath); err == nil {
		realPath, err = filepath.EvalSymlinks(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlinks: %w", err)
		}
	} else {
		// File doesn't exist — resolve parent dir + filename
		parentDir := filepath.Dir(absPath)
		if _, err := os.Stat(parentDir); err == nil {
			realParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				return "", fmt.Errorf("failed to resolve parent symlinks: %w", err)
			}
			realPath = filepath.Join(realParent, filepath.Base(absPath))
		} else {
			// Parent doesn't exist either — just use the cleaned abs path
			realPath = absPath
		}
	}

	// Check containment: realPath must be under realWorkspace
	if !isSubpath(realWorkspace, realPath) {
		return "", fmt.Errorf("path %q is outside workspace %q", filePath, workspaceDir)
	}

	return absPath, nil
}

// ValidateURI extracts a file path from a file:// URI and validates it
// against the workspace boundary. Returns the cleaned absolute path.
func ValidateURI(workspaceDir, uri string) (string, error) {
	path := strings.TrimPrefix(uri, "file://")
	return ValidatePath(workspaceDir, path)
}

// isSubpath checks if child is under parent (or is parent itself).
func isSubpath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	if child == parent {
		return true
	}

	// Ensure parent ends with separator for prefix check
	parentWithSep := parent + string(filepath.Separator)
	return strings.HasPrefix(child, parentWithSep)
}
