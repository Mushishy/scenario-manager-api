package utils

import "fmt"

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
