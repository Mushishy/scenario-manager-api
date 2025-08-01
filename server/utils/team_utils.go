package utils

import (
	"fmt"
	"strings"
)

// Helper function to generate userId from user name
func generateUserId(user string) string {
	// Remove spaces, convert to lowercase, and add BATCH prefix
	userId := strings.ToLower(strings.ReplaceAll(user, " ", ""))
	return "BATCH" + userId
}

// Helper function to process usersAndTeams and add userId
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

func ValidateTeamField(ctfdData []interface{}) error {
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

func ValidateUsersAndTeams(usersAndTeams []interface{}) error {
	teamSet := false

	// Check if any 'team' field is set
	for _, item := range usersAndTeams {
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
		for _, item := range usersAndTeams {
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
