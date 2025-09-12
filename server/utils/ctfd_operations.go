package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CtfdTopologyRequest represents the request structure for CTFd topology creation
type CtfdTopologyRequest struct {
	TopologyName           string `json:"topologyName"`
	ScenarioID             string `json:"scenarioId"`
	PoolID                 string `json:"poolId"`
	UsernameConfig         string `json:"usernameConfig"`
	PasswordConfig         string `json:"passwordConfig"`
	AdminUsername          string `json:"adminUsername"`
	AdminPassword          string `json:"adminPassword"`
	CtfName                string `json:"ctfName"`
	CtfDescription         string `json:"ctfDescription"`
	ChallengeVisibility    string `json:"challengeVisibility"`
	AccountVisibility      string `json:"accountVisibility"`
	ScoreVisibility        string `json:"scoreVisibility"`
	RegistrationVisibility string `json:"registrationVisibility"`
	AllowNameChanges       string `json:"allowNameChanges"`
	AllowTeamCreation      string `json:"allowTeamCreation"`
	AllowTeamDisbanding    string `json:"allowTeamDisbanding"`
	ConfStartTime          string `json:"confStartTime"`
	ConfStopTime           string `json:"confStopTime"`
	TimeZone               string `json:"timeZone"`
	AllowViewingAfter      string `json:"allowViewingAfter"`
}

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

// ValidateUsersAndTeams validates team field consistency and ensures unique users
func ValidateUsersAndTeams(ctfdData []interface{}) error {
	teamSet := false
	userSet := make(map[string]bool)

	// Check if any 'team' field is set and validate unique users
	for _, item := range ctfdData {
		data, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid data format")
		}

		// Check for duplicate users
		if user, exists := data["user"].(string); exists {
			if userSet[user] {
				return fmt.Errorf("duplicate user found: %s", user)
			}
			userSet[user] = true
		} else {
			return fmt.Errorf("missing 'user' field")
		}

		if team, exists := data["team"]; exists && team != "" {
			teamSet = true
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
	filePath := filepath.Join(dataPath, "ctfd_data.json")

	// Check if file already exists, if so, do nothing
	if _, err := os.Stat(filePath); err == nil {
		return nil // File exists, silently return
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// ValidateMainUserNotInUsersAndTeams ensures mainUser is not present in usersAndTeams
func ValidateMainUserNotInUsersAndTeams(mainUser string, usersAndTeams []interface{}) error {
	if mainUser == "" {
		return nil // No main user to validate
	}

	for _, item := range usersAndTeams {
		if userMap, ok := item.(map[string]interface{}); ok {
			if user, exists := userMap["user"].(string); exists && user == mainUser {
				return fmt.Errorf("main user '%s' cannot be included in usersAndTeams", mainUser)
			}
		}
	}

	return nil
}

// GetAllUserIdsFromPools collects all userIds from mainUser and usersAndTeams across all pools
func GetAllUserIdsFromPools(poolFolder string) (map[string]bool, error) {
	userIds := make(map[string]bool)

	pools, err := GetAllPools(poolFolder)
	if err != nil {
		return nil, err
	}

	for _, pool := range pools {
		poolMap := pool // pool is already map[string]interface{}

		poolId, ok := poolMap["poolId"].(string)
		if !ok {
			continue
		}

		// Read full pool data
		poolPath := filepath.Join(poolFolder, poolId)
		file, err := os.Open(filepath.Join(poolPath, "pool.json"))
		if err != nil {
			continue
		}

		var poolData map[string]interface{}
		if err := json.NewDecoder(file).Decode(&poolData); err != nil {
			file.Close()
			continue
		}
		file.Close()

		// Extract mainUser userId if exists
		if mainUser, exists := poolData["mainUser"].(string); exists && mainUser != "" {
			mainUserId := generateUserId(mainUser)
			userIds[mainUserId] = true
		}

		// Extract userIds from usersAndTeams
		if usersAndTeams, exists := poolData["usersAndTeams"].([]interface{}); exists {
			for _, item := range usersAndTeams {
				if userMap, ok := item.(map[string]interface{}); ok {
					if userId, exists := userMap["userId"].(string); exists {
						userIds[userId] = true
					}
				}
			}
		}
	}

	return userIds, nil
}
