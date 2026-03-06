package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"crypto/rand"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

var validFolderIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const lowercaseCharset = "abcdefghijklmnopqrstuvwxyz"

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

// RandomLowercaseString generates a random lowercase string of the given length.
func RandomLowercaseString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic("Failed to generate random string")
	}
	for i := range b {
		b[i] = lowercaseCharset[b[i]%byte(len(lowercaseCharset))]
	}
	return string(b)
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

// Compare two hashed passwords
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Compare two config strings
func CompareConfigs(config1, config2 interface{}) bool {
	return fmt.Sprintf("%v", config1) == fmt.Sprintf("%v", config2)
}

// ValidateFolderID ensures that the folderID is valid and exists within the baseFolder.
// Handles HTTP responses automatically and returns (string, bool).
func ValidateFolderId(c *gin.Context, baseFolder, folderID string) (string, bool) {
	// Check if folderID is empty
	if folderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	// Validate folderID against the regex
	if !validFolderIDRegex.MatchString(folderID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	// Resolve the full path
	folderPath := filepath.Join(baseFolder, folderID)

	// Ensure the resolved path is within the base folder
	baseFolderAbs, err := filepath.Abs(baseFolder)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	folderPathAbs, err := filepath.Abs(folderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	// Use filepath.Rel to check if folderPathAbs is within baseFolderAbs
	relPath, err := filepath.Rel(baseFolderAbs, folderPathAbs)
	if err != nil || strings.HasPrefix(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	// Check if the folder exists (only after validating it's within bounds)
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return "", false
	}

	// Resolve symlinks only after we know the folder exists and is valid
	resolvedPath, err := filepath.EvalSymlinks(folderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	return resolvedPath, true
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

// ValidateDateTimeRange validates date time format and ensures stop time is after start time
func ValidateDateTimeRange(startTime, stopTime string) bool {
	layout := "02/01/2006 15:04"

	// Both times must be provided or both must be empty
	if (startTime == "") != (stopTime == "") {
		return false // One exists and the other doesn't
	}

	// If both are empty, validation passes
	if startTime == "" && stopTime == "" {
		return true
	}

	// Both times are provided, validate formats
	parsedStartTime, err := time.Parse(layout, startTime)
	if err != nil {
		return false
	}

	// Additional validation to ensure the parsed time matches the input
	if parsedStartTime.Format(layout) != startTime {
		return false
	}

	parsedStopTime, err := time.Parse(layout, stopTime)
	if err != nil {
		return false
	}

	// Additional validation to ensure the parsed time matches the input
	if parsedStopTime.Format(layout) != stopTime {
		return false
	}

	// Validate that stop time is strictly greater than start time (not equal)
	if !parsedStopTime.After(parsedStartTime) {
		return false
	}

	return true
}
