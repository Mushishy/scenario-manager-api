package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// generateUserId generates userId from user name
func generateUserId(user string) string {
	// Remove spaces, convert to lowercase, and add BATCH prefix
	userId := strings.ToLower(strings.ReplaceAll(user, " ", ""))
	return "BATCH" + userId
}

// ProcessUsersAndTeams processes usersAndTeams and adds userId
func ProcessUsersAndTeams(usersAndTeams []interface{}) []interface{} {
	var processed []interface{}

	for _, item := range usersAndTeams {
		if itemMap, ok := item.(map[string]interface{}); ok {
			// Create a copy of the item
			newItem := make(map[string]interface{})
			for k, v := range itemMap {
				newItem[k] = v
			}

			// Add userId based on user field
			if user, exists := itemMap["user"].(string); exists {
				newItem["userId"] = generateUserId(user)
			}

			processed = append(processed, newItem)
		}
	}

	return processed
}

// ValidateUsersAndTeams validates team field consistency
func ValidateUsersAndTeams(ctfdData []interface{}) error {
	teamSet := false

	// Check if any 'team' field is set
	for _, item := range ctfdData {
		data, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid data format")
		}

		if team, exists := data["team"]; exists && team != "" {
			teamSet = true
			break
		}
	}

	// If one 'team' is set, ensure all 'team' fields are set
	if teamSet {
		for _, item := range ctfdData {
			data, ok := item.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid data format")
			}

			if _, exists := data["team"]; !exists || data["team"] == "" {
				return fmt.Errorf("missing or empty 'team' field")
			}
		}
	}

	return nil
}

// ReadCTFdJSON reads and parses CTFd JSON data
func ReadCTFdJSON(dataPath string) (map[string]interface{}, error) {
	filePath := filepath.Join(dataPath, "ctfd_data.json")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// GetAllCTFdData returns all CTFd data items
func GetAllCTFdData(baseFolder string) ([]map[string]interface{}, error) {
	var dataItems []map[string]interface{}

	files, err := os.ReadDir(baseFolder)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			dataPath := filepath.Join(baseFolder, file.Name())

			// Get file info for creation time
			fileInfo, err := os.Stat(dataPath)
			if err != nil {
				continue
			}

			// Try to read the data to get additional info
			data, err := ReadCTFdJSON(dataPath)
			var itemCount int
			if err == nil {
				if ctfdData, ok := data["ctfd_data"].([]interface{}); ok {
					itemCount = len(ctfdData)
				}
			}

			dataItems = append(dataItems, map[string]interface{}{
				"poolId":    file.Name(),
				"createdAt": fileInfo.ModTime().Format(time.RFC3339),
				"itemCount": itemCount,
			})
		}
	}

	return dataItems, nil
}

// ValidateFlagsConsistency ensures that if one user has flags set, all users must have flags set
func ValidateFlagsConsistency(ctfdData []interface{}) error {
	if len(ctfdData) == 0 {
		return nil
	}

	var hasFlags *bool = nil

	for _, item := range ctfdData {
		user, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		flags, flagsExist := user["flags"]
		userHasFlags := flagsExist && flags != nil

		// If flags is an array, check if it's not empty
		if userHasFlags {
			if flagsArray, ok := flags.([]interface{}); ok {
				userHasFlags = len(flagsArray) > 0
			}
		}

		if hasFlags == nil {
			hasFlags = &userHasFlags
		} else if *hasFlags != userHasFlags {
			return fmt.Errorf("inconsistent flags: if one user has flags, all users must have flags")
		}
	}

	return nil
}

// SaveCTFdData saves CTFd data to the specified path
func SaveCTFdData(dataPath string, data map[string]interface{}) error {
	filePath := filepath.Join(dataPath, "ctfd_data.json") // Changed from "data.json" to "ctfd_data.json"
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print JSON
	return encoder.Encode(data)
}
