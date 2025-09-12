package utils

import (
	"crypto/rand"
	"dulus/server/config"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/crypto/bcrypt"
)

type UserRetrievalOption int

const (
	SharedMainUserOnly UserRetrievalOption = iota
	SharedUsersAndTeamsOnly
	SharedAllUsers // Both main user and users from usersAndTeams
)

var validFolderIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateUniqueID generates a unique 6-character alphanumeric ID and ensures it's unique in the given folder.
func GenerateUniqueID(basePath string) (string, error) {
	for {
		id := RandomString(6)
		if _, err := os.Stat(filepath.Join(basePath, id)); os.IsNotExist(err) {
			return id, nil
		}
	}
}

// randomString generates a random alphanumeric string of the given length.
func RandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic("Failed to generate random string")
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// ValidateFolderID ensures that the folderID is valid and exists within the baseFolder.
func ValidateFolderID(baseFolder, folderID string) (string, error) {
	// Check if folderID is empty
	if folderID == "" {
		return "", os.ErrInvalid
	}

	// Validate folderID against the regex
	if !validFolderIDRegex.MatchString(folderID) {
		return "", os.ErrInvalid
	}

	// Resolve the full path
	folderPath := filepath.Join(baseFolder, folderID)

	// Ensure the resolved path is within the base folder
	baseFolderAbs, err := filepath.Abs(baseFolder)
	if err != nil {
		return "", os.ErrInvalid
	}

	folderPathAbs, err := filepath.Abs(folderPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	// Use filepath.Rel to check if folderPathAbs is within baseFolderAbs
	relPath, err := filepath.Rel(baseFolderAbs, folderPathAbs)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", os.ErrInvalid
	}

	// Check if the folder exists (only after validating it's within bounds)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return "", os.ErrNotExist
	}

	// Resolve symlinks only after we know the folder exists and is valid
	resolvedPath, err := filepath.EvalSymlinks(folderPath)
	if err != nil {
		return "", os.ErrInvalid
	}

	return resolvedPath, nil
}

// CheckPasswordHash compares password with hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Extracts userID from APIKey, returns userID and error if malformed
func ExtractUserIDFromAPIKey(c *gin.Context, APIKey string) (string, bool) {
	apiKeySplit := strings.Split(APIKey, ".")
	if len(apiKeySplit) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Malformed API Key provided"})
		c.Abort()
		return "", false
	}
	return apiKeySplit[0], true
}

// ValidateJSONSchema validates JSON input against a schema and returns the parsed data
func ValidateJSONSchema(c *gin.Context, schemaPath string) (map[string]interface{}, bool) {
	schemaLoader := gojsonschema.NewReferenceLoader(schemaPath)

	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return nil, false
	}

	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return nil, false
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return nil, false
	}

	return input, true
}

// ValidateFolderWithResponse validates folder ID and handles HTTP responses
func ValidateFolderWithResponse(c *gin.Context, baseFolder, folderId string) (string, bool) {
	folderPath, err := ValidateFolderID(baseFolder, folderId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return "", false
	default:
		return folderPath, true
	}
}

func GetUserIdsFromPool(poolId string, option UserRetrievalOption) ([]string, error) {
	poolPath, err := ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		return nil, fmt.Errorf("invalid pool ID")
	case os.ErrNotExist:
		return nil, fmt.Errorf("pool not found")
	case nil:
		// Success, continue
	default:
		return nil, fmt.Errorf("failed to validate pool: %w", err)
	}

	poolJsonPath := filepath.Join(poolPath, "pool.json")

	data, err := os.ReadFile(poolJsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool file: %w", err)
	}

	var pool Pool

	if err := json.Unmarshal(data, &pool); err != nil {
		return nil, fmt.Errorf("failed to parse pool file: %w", err)
	}

	var userIds []string

	if pool.Type == "SHARED" || pool.Type == "CTFD" {
		// For SHARED/CTFD pools: include MainUser based on option
		if pool.MainUser != "" && (option == SharedMainUserOnly || option == SharedAllUsers) {
			userIds = append(userIds, pool.MainUser)
		}
		// Include usersAndTeams based on option
		if option == SharedAllUsers || option == SharedUsersAndTeamsOnly {
			for _, userTeam := range pool.UsersAndTeams {
				userIds = append(userIds, userTeam.UserId)
			}
		}
	} else {
		// For other pool types: always include usersAndTeams
		for _, userTeam := range pool.UsersAndTeams {
			userIds = append(userIds, userTeam.UserId)
		}
	}

	return userIds, nil
}

// IsMainUserAlreadyUsed checks if a main user is already assigned to any pool
// It checks both the mainUser field and iterates through usersAndTeams to find if the userId exists
func IsMainUserAlreadyUsed(mainUser string) (bool, error) {
	if mainUser == "" {
		return false, fmt.Errorf("main user cannot be empty")
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

		// Check if this pool uses the main user in the mainUser field
		if pool.MainUser == mainUser {
			return true, nil
		}

		// Check if the mainUser exists as a userId in any of the usersAndTeams
		for _, userTeam := range pool.UsersAndTeams {
			if userTeam.UserId == mainUser {
				return true, nil
			}
		}
	}

	return false, nil
}

func GetMainUserFromPool(poolId string) (string, error) {
	poolPath, err := ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		return "", fmt.Errorf("invalid pool ID")
	case os.ErrNotExist:
		return "", fmt.Errorf("pool not found")
	case nil:
		// Success, continue
	default:
		return "", fmt.Errorf("failed to validate pool: %w", err)
	}

	poolJsonPath := filepath.Join(poolPath, "pool.json")

	data, err := os.ReadFile(poolJsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to read pool file: %w", err)
	}

	var pool Pool

	if err := json.Unmarshal(data, &pool); err != nil {
		return "", fmt.Errorf("failed to parse pool file: %w", err)
	}

	if pool.Type == "SHARED" || pool.Type == "CTFD" {
		if pool.MainUser == "" {
			return "", fmt.Errorf("no main user found in pool")
		}
		return pool.MainUser, nil
	}

	return "", nil
}
