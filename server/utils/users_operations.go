package utils

import (
	"dulus/server/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserRetrievalOption int

const (
	SharedMainUserOnly UserRetrievalOption = iota
	SharedUsersAndTeamsOnly
	SharedAllUsers // Both main user and users from usersAndTeams
)

const (
	OperationCreate UserRetrievalOption = iota
	OperationAdd
)

// replaceSpecialChars replaces special characters with their ASCII equivalents
func replaceSpecialChars(s string) string {
	replacements := map[rune]rune{
		'á': 'a', 'à': 'a', 'â': 'a', 'ä': 'a', 'ã': 'a', 'å': 'a', 'ā': 'a', 'ą': 'a',
		'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e', 'ē': 'e', 'ę': 'e', 'ě': 'e',
		'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i', 'ī': 'i', 'į': 'i',
		'ó': 'o', 'ò': 'o', 'ô': 'o', 'ö': 'o', 'õ': 'o', 'ø': 'o', 'ō': 'o', 'ő': 'o',
		'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u', 'ū': 'u', 'ů': 'u', 'ű': 'u', 'ų': 'u',
		'ý': 'y', 'ÿ': 'y', 'ỳ': 'y',
		'ñ': 'n', 'ň': 'n', 'ń': 'n',
		'ç': 'c', 'č': 'c', 'ć': 'c',
		'š': 's', 'ś': 's', 'ş': 's',
		'ž': 'z', 'ź': 'z', 'ż': 'z',
		'đ': 'd', 'ď': 'd',
		'ť': 't', 'ţ': 't',
		'ř': 'r', 'ŕ': 'r',
		'ľ': 'l', 'ł': 'l',
		'ğ': 'g',
		'ß': 's',
	}

	var result strings.Builder
	for _, char := range s {
		if replacement, exists := replacements[char]; exists {
			result.WriteRune(replacement)
		} else if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == ' ' {
			result.WriteRune(char)
		}
		// Skip any other special characters
	}
	return result.String()
}

// generateUserId generates userId from user name
func generateUserId(user string) string {
	// Remove spaces, convert to lowercase, replace special characters
	userId := strings.ToLower(strings.ReplaceAll(user, " ", ""))
	userId = replaceSpecialChars(userId)
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
				// Normalize user field: lowercase and replace special characters, keep spaces
				normalizedUser := strings.ToLower(user)
				cleanUser := replaceSpecialChars(normalizedUser)
				// Update the user field with cleaned version
				newItem["user"] = cleanUser
				newItem["userId"] = generateUserId(cleanUser)
			}

			processed = append(processed, newItem)
		}
	}

	return processed
}

// ValidateAndProcessUsersAndTeams validates and processes usersAndTeams in one go
// Schema already validates: mainUserId required for SHARED, not allowed for INDIVIDUAL
func ValidateAndProcessUsersAndTeams(usersAndTeams []interface{}, poolType string, operation UserRetrievalOption) ([]interface{}, error) {
	// Step 1: Process and add userIds
	processed := ProcessUsersAndTeams(usersAndTeams)

	// Step 2: Validate duplicates and collect IDs
	teamSet := false
	userSet := make(map[string]bool)
	userIdSet := make(map[string]bool)
	mainUserIds := make(map[string]bool)
	allUserIds := []string{}

	for _, item := range processed {
		data, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid data format")
		}

		// Check for duplicate usernames
		if user, exists := data["user"].(string); exists {
			if userSet[user] {
				return nil, fmt.Errorf("duplicate user found: %s", user)
			}
			userSet[user] = true
		}

		// Check for duplicate userIds and collect them
		if userId, exists := data["userId"].(string); exists && userId != "" {
			if userIdSet[userId] {
				return nil, fmt.Errorf("duplicate userId found: %s", userId)
			}
			// Validate userId length (must be less than 20 characters)
			if len(userId) >= 20 {
				return nil, fmt.Errorf("userId '%s' must be less than 20 characters (current: %d)", userId, len(userId))
			}
			userIdSet[userId] = true
			allUserIds = append(allUserIds, userId)
		}

		// Check if team is set
		if team, exists := data["team"]; exists && team != "" {
			teamSet = true
		}

		// Collect mainUserIds
		if mainUserId, exists := data["mainUserId"]; exists && mainUserId != "" {
			mainUserIds[mainUserId.(string)] = true
		}
	}

	// Validate team consistency: if one has team, all must have team
	if teamSet {
		for _, item := range processed {
			data := item.(map[string]interface{})
			if team, exists := data["team"]; !exists || team == "" {
				return nil, fmt.Errorf("if one user has team, all must have team")
			}
		}
	}

	// Validate userId cannot be mainUserId within the same request
	for _, userId := range allUserIds {
		if mainUserIds[userId] {
			return nil, fmt.Errorf("userId '%s' cannot also be a mainUserId in the same request", userId)
		}
	}

	// Step 3: Cross-pool validation
	existingMainUsers, err := GetAllMainUsersFromPools()
	if err != nil {
		return nil, fmt.Errorf("failed to validate against existing pools: %w", err)
	}

	// Check if any of our userIds are already main users in other pools
	for _, userId := range allUserIds {
		if existingMainUsers[userId] {
			return nil, fmt.Errorf("userId '%s' is already a main user in another pool", userId)
		}
	}

	// Check if any of our mainUserIds are already used in other pools
	if len(mainUserIds) > 0 && operation == OperationCreate {
		mainUserIdList := make([]string, 0, len(mainUserIds))
		for id := range mainUserIds {
			mainUserIdList = append(mainUserIdList, id)
		}

		isUsed, err := IsMainUserAlreadyUsed(mainUserIdList)
		if err != nil {
			return nil, fmt.Errorf("failed to validate main users: %w", err)
		}
		if isUsed {
			return nil, fmt.Errorf("one or more main users are already assigned to another pool")
		}
	}

	return processed, nil
}

// For shared pools, it retrieves flags only from mainUser and applies them to all users
// For individual pools, it retrieves flags separately for each user
func GetUserIdsFromPool(c *gin.Context, poolId string, option UserRetrievalOption) ([]string, bool) {
	poolPath, ok := ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return nil, false
	}

	poolJsonPath := filepath.Join(poolPath, "pool.json")

	data, err := os.ReadFile(poolJsonPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return nil, false
	}

	var pool Pool

	if err := json.Unmarshal(data, &pool); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return nil, false
	}

	var userIds []string

	if pool.Type == "SHARED" {
		// For SHARED pools: filter based on option
		switch option {
		case SharedMainUserOnly:
			// Collect unique mainUserId values - these are the actual main users
			mainUserIds := make(map[string]bool)
			for _, userTeam := range pool.UsersAndTeams {
				if userTeam.MainUserId != "" && !mainUserIds[userTeam.MainUserId] {
					mainUserIds[userTeam.MainUserId] = true
					userIds = append(userIds, userTeam.MainUserId)
				}
			}
		case SharedUsersAndTeamsOnly:
			// Only include regular users (all userId values)
			for _, userTeam := range pool.UsersAndTeams {
				userIds = append(userIds, userTeam.UserId)
			}
		case SharedAllUsers:
			// Include all users (both mainUserIds and userIds)
			allUsers := make(map[string]bool)
			for _, userTeam := range pool.UsersAndTeams {
				if userTeam.MainUserId != "" && !allUsers[userTeam.MainUserId] {
					allUsers[userTeam.MainUserId] = true
					userIds = append(userIds, userTeam.MainUserId)
				}
				if !allUsers[userTeam.UserId] {
					allUsers[userTeam.UserId] = true
					userIds = append(userIds, userTeam.UserId)
				}
			}
		}
	} else {
		// For other pool types: always include all usersAndTeams
		for _, userTeam := range pool.UsersAndTeams {
			userIds = append(userIds, userTeam.UserId)
		}
	}

	return userIds, true
}

// ExtractUserIdsAndMainUserIdsFromPool extracts distinct userIds and mainUserIds from pool's usersAndTeams
// Returns two arrays: userIds and mainUserIds (both with distinct values)
func ExtractUserIdsAndMainUserIdsFromPool(pool Pool) ([]string, []string) {
	userIdMap := make(map[string]bool)
	mainUserIdMap := make(map[string]bool)

	var userIds []string
	var mainUserIds []string

	for _, userTeam := range pool.UsersAndTeams {
		// Collect distinct userIds
		if userTeam.UserId != "" && !userIdMap[userTeam.UserId] {
			userIdMap[userTeam.UserId] = true
			userIds = append(userIds, userTeam.UserId)
		}

		// Collect distinct mainUserIds (only if not empty)
		if userTeam.MainUserId != "" && !mainUserIdMap[userTeam.MainUserId] {
			mainUserIdMap[userTeam.MainUserId] = true
			mainUserIds = append(mainUserIds, userTeam.MainUserId)
		}
	}

	return userIds, mainUserIds
}

// GetAllMainUsersFromPools gets all main users (mainUserId values) from all pools
func GetAllMainUsersFromPools() (map[string]bool, error) {
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

		var pool Pool
		if err := json.Unmarshal(data, &pool); err != nil {
			// Skip this pool if we can't parse it, don't fail the entire operation
			continue
		}

		// Extract all mainUserId values - these are the actual main users
		for _, userTeam := range pool.UsersAndTeams {
			if userTeam.MainUserId != "" {
				mainUsers[userTeam.MainUserId] = true
			}
		}
	}

	return mainUsers, nil
}

// GetAllUserIdsFromPools collects all userIds from usersAndTeams across all pools
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

		// Extract userIds and mainUserIds from usersAndTeams
		if usersAndTeams, exists := poolData["usersAndTeams"].([]interface{}); exists {
			for _, item := range usersAndTeams {
				if userMap, ok := item.(map[string]interface{}); ok {
					// Add userId if it exists
					if userId, exists := userMap["userId"].(string); exists && userId != "" {
						userIds[userId] = true
					}
					// Add mainUserId if it exists (distinct values)
					if mainUserId, exists := userMap["mainUserId"].(string); exists && mainUserId != "" {
						userIds[mainUserId] = true
					}
				}
			}
		}
	}

	return userIds, nil
}

// IsMainUserAlreadyUsed checks if any of the provided main user IDs are already assigned in any pool
// It checks if the mainUserIds exist in usersAndTeams either as userId or mainUserId
func IsMainUserAlreadyUsed(mainUserIds []string) (bool, error) {
	if len(mainUserIds) == 0 {
		return false, fmt.Errorf("main user IDs array cannot be empty")
	}

	// Create a map for quick lookup
	mainUserIdMap := make(map[string]bool)
	for _, id := range mainUserIds {
		if id != "" {
			mainUserIdMap[id] = true
		}
	}

	if len(mainUserIdMap) == 0 {
		return false, fmt.Errorf("no valid main user IDs provided")
	}

	// Read all directories in the pool folder
	poolDirs, err := os.ReadDir(config.PoolFolder)
	if err != nil {
		return false, fmt.Errorf("failed to read pool folder: %w", err)
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

		var pool Pool
		if err := json.Unmarshal(data, &pool); err != nil {
			// Skip this pool if we can't parse it, don't fail the entire operation
			continue
		}

		// Check if any of the mainUserIds exist as userId or mainUserId in usersAndTeams
		for _, userTeam := range pool.UsersAndTeams {
			if mainUserIdMap[userTeam.UserId] || mainUserIdMap[userTeam.MainUserId] {
				return true, nil
			}
		}
	}

	return false, nil
}
