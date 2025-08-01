package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func CompareConfigs(config1, config2 interface{}) bool {
	return fmt.Sprintf("%v", config1) == fmt.Sprintf("%v", config2)
}

func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
