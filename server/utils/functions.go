package utils

import (
	"os"

	"golang.org/x/crypto/bcrypt"
)

// ensureDirectoryExists creates the directory if it doesn't exist
func EnsureDirectoryExists(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic("Failed to create directory: " + path)
	}
}

func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
