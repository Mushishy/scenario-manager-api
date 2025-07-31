package utils

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func EnsureDirectoryExists(path string) {
	// ensureDirectoryExists creates the directory if it doesn't exist
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic("Failed to create directory: " + path)
	}
}

func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func CompareConfigs(config1, config2 interface{}) bool {
	return fmt.Sprintf("%v", config1) == fmt.Sprintf("%v", config2)
}
