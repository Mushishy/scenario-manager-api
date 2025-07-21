package utils

import (
	"crypto/rand"
	"errors"
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
		return "", errors.New("invalid folder ID")
	}

	// Resolve the full path
	folderPath := filepath.Join(baseFolder, folderID)
	resolvedPath, err := filepath.EvalSymlinks(folderPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	// Ensure the resolved path is within the base folder
	baseFolderAbs, err := filepath.Abs(baseFolder)
	if err != nil {
		return "", os.ErrInvalid
	}
	resolvedPathAbs, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	// Use filepath.Rel to check if resolvedPathAbs is within baseFolderAbs
	relPath, err := filepath.Rel(baseFolderAbs, resolvedPathAbs)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", os.ErrInvalid
	}

	// Check if the folder exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", os.ErrNotExist
	}

	return resolvedPath, nil
}
