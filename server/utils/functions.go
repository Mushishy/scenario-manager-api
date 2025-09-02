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

// containsAll checks if all items in needles are present in haystack
func ContainsAll(haystack, needles []string) bool {
	for _, needle := range needles {
		found := false
		for _, hay := range haystack {
			if hay == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
