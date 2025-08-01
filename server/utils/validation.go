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

var validFolderIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateUniqueID generates a unique 6-character alphanumeric ID and ensures it's unique in the given folder.
func GenerateUniqueID(basePath string) (string, error) {
	for {
		id := randomString(6)
		if _, err := os.Stat(filepath.Join(basePath, id)); os.IsNotExist(err) {
			return id, nil
		}
	}
}

// randomString generates a random alphanumeric string of the given length.
func randomString(length int) string {
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

// GetUserIdsFromPool helper function to get userIds from pool
func GetUserIdsFromPool(poolId string) ([]string, error) {
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

	var pool struct {
		UsersAndTeams []struct {
			UserId string `json:"userId"`
		} `json:"usersAndTeams"`
	}

	if err := json.Unmarshal(data, &pool); err != nil {
		return nil, fmt.Errorf("failed to parse pool file: %w", err)
	}

	var userIds []string
	for _, userTeam := range pool.UsersAndTeams {
		userIds = append(userIds, userTeam.UserId)
	}

	return userIds, nil
}
