package utils

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var validFolderIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateUniqueID generates a unique 6-character alphanumeric ID and ensures it's unique in the given folder.
func GenerateUniqueID(basePath string) (string, error) {
	for {
		id := randomString(6)
		if _, err := os.Stat(filepath.Join(basePath, id)); os.IsNotExist(err) {
			return id, nil
		}
	}
}

// randomString generates a random alphanumeric string of the given length.
func randomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic("Failed to generate random string")
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// ValidateFolderID ensures that the folderID is valid and exists within the baseFolder.
func ValidateFolderID(baseFolder, folderID string) (string, error) {
	// Check if folderID is empty
	if folderID == "" {
		return "", os.ErrInvalid
	}

	// Validate folderID against the regex
	if !validFolderIDRegex.MatchString(folderID) {
		return "", os.ErrInvalid
	}

	// Resolve the full path
	folderPath := filepath.Join(baseFolder, folderID)

	// Ensure the resolved path is within the base folder
	baseFolderAbs, err := filepath.Abs(baseFolder)
	if err != nil {
		return "", os.ErrInvalid
	}

	folderPathAbs, err := filepath.Abs(folderPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	// Use filepath.Rel to check if folderPathAbs is within baseFolderAbs
	relPath, err := filepath.Rel(baseFolderAbs, folderPathAbs)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", os.ErrInvalid
	}

	// Check if the folder exists (only after validating it's within bounds)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return "", os.ErrNotExist
	}

	// Resolve symlinks only after we know the folder exists and is valid
	resolvedPath, err := filepath.EvalSymlinks(folderPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	return resolvedPath, nil
}
