package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"dulus/server/config"
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

// ValidateUsersNotMainUsers validates that no userId in usersAndTeams is already being used as a mainUser in any pool
func ValidateUsersNotMainUsers(usersAndTeams []interface{}) error {
	if len(usersAndTeams) == 0 {
		return nil
	}

	// Get all main users from all pools
	mainUsers, err := getAllMainUsersFromPools()
	if err != nil {
		return fmt.Errorf("failed to get main users from pools: %w", err)
	}

	// Check each userId in usersAndTeams
	for _, item := range usersAndTeams {
		userMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		userId, exists := userMap["userId"].(string)
		if !exists {
			continue
		}
		fmt.Println("Checking userId:", userId)
		fmt.Println("Main users:", mainUsers)
		fmt.Println("User exists as main user:", mainUsers[userId])

		if mainUsers[userId] {
			return fmt.Errorf("user ID '%s' is already being used as a main user in another pool", userId)
		}
	}

	return nil
}

// getAllMainUsersFromPools gets all main users (as userIds) from all pools
func getAllMainUsersFromPools() (map[string]bool, error) {
	mainUsers := make(map[string]bool)

	// Read all directories in the pool folder
	poolDirs, err := os.ReadDir(config.PoolFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool folder: %w", err)
	}

	for _, dir := range poolDirs {
		if !dir.IsDir() {
			continue
		}

		poolJsonPath := filepath.Join(config.PoolFolder, dir.Name(), "pool.json")

		// Check if pool.json exists
		if _, err := os.Stat(poolJsonPath); os.IsNotExist(err) {
			continue
		}

		// Read and parse pool.json
		data, err := os.ReadFile(poolJsonPath)
		if err != nil {
			// Skip this pool if we can't read it, don't fail the entire operation
			continue
		}

		var poolData map[string]interface{}
		if err := json.Unmarshal(data, &poolData); err != nil {
			// Skip this pool if we can't parse it, don't fail the entire operation
			continue
		}

		// Extract mainUser and convert to userId if exists
		if mainUser, exists := poolData["mainUser"].(string); exists && mainUser != "" {
			mainUsers[mainUser] = true
		}
	}

	return mainUsers, nil
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

// GetUserDetailsAndExtractFlags combines GetUserDetailsFromPool and ExtractFlagsFromLogs
// For shared pools, it retrieves flags only from mainUser and applies them to all users
// For individual pools, it retrieves flags separately for each user
func GetUserDetailsAndExtractFlags(poolPath string, apiKey string) ([]CtfdUser, error) {
	// Read pool data
	poolJsonPath := filepath.Join(poolPath, "pool.json")
	poolData, err := os.ReadFile(poolJsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool data: %w", err)
	}

	var pool Pool
	if err := json.Unmarshal(poolData, &pool); err != nil {
		return nil, fmt.Errorf("failed to parse pool data: %w", err)
	}

	var ctfdUsers []CtfdUser

	// Handle shared pools - get flags from mainUser only
	if pool.Type == "SHARED" && pool.MainUser != "" {
		// Get flags from main user
		mainUserFlags, err := getMainUserFlags(pool.MainUser, apiKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get main user flags: %w", err)
		}

		// Create CtfdUser entries for all users with same flags
		for _, userTeam := range pool.UsersAndTeams {
			ctfdUser := CtfdUser{
				User:     userTeam.User,
				Password: RandomString(6),
				Team:     userTeam.Team,
				Flags:    mainUserFlags,
			}
			ctfdUsers = append(ctfdUsers, ctfdUser)
		}
	} else {
		// Handle individual pools - get flags for each user separately
		userIds := make([]string, len(pool.UsersAndTeams))
		userDetailMap := make(map[string]UserDetails)

		for i, userTeam := range pool.UsersAndTeams {
			userIds[i] = userTeam.UserId
			userDetailMap[userTeam.UserId] = UserDetails{
				Username: userTeam.User,
				Team:     userTeam.Team,
			}
		}

		ctfdUsers = ExtractFlagsFromLogs(userIds, userDetailMap, apiKey)
	}

	return ctfdUsers, nil
}

// getMainUserFlags retrieves flags for the main user
func getMainUserFlags(mainUser string, apiKey string) ([]Flag, error) {
	// Make request to get logs for main user
	response, err := MakeLudusRequest("GET", config.LudusUrl+"/range/logs/?userID="+mainUser, nil, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for main user: %w", err)
	}

	// Extract flags using the same logic as ExtractUserFlags
	ludusResponse := LudusResponse{
		UserID:   mainUser,
		Response: response,
		Error:    nil,
	}

	flagPattern := regexp.MustCompile(`&%&(.*?)&%&`)
	flags := ExtractUserFlags(ludusResponse, flagPattern)

	return flags, nil
}
