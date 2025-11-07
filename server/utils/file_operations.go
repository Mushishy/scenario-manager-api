package utils

import (
	"dulus/server/config"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// FileInfo represents file information
type FileInfo struct {
	Name         string    `json:"name"`
	Content      string    `json:"content"`
	CreationTime time.Time `json:"creationTime"`
}

// ItemInfo represents basic item information
type ItemInfo struct {
	ID           string    `json:"id"`
	CreationTime time.Time `json:"creationTime"`
}

// ReadFirstFileInDir reads the first file in a directory and returns its info
func ReadFirstFileInDir(dirPath string) (*FileInfo, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil || len(files) == 0 {
		return nil, os.ErrNotExist
	}

	// Find the first non-directory file
	var fileName string
	for _, file := range files {
		if !file.IsDir() {
			fileName = file.Name()
			break
		}
	}

	if fileName == "" {
		return nil, os.ErrNotExist
	}

	filePath := filepath.Join(dirPath, fileName)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Get file creation time
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Name:         fileName,
		Content:      string(content),
		CreationTime: fileInfo.ModTime(),
	}, nil
}

// EnsureDirectoryExists creates the directory if it doesn't exist
func EnsureDirectoryExists(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetSingleItemWithFile gets a single item by ID with its first file encoded as base64
func GetSingleItemWithFile(c *gin.Context, baseFolder, itemId, itemType string) {
	itemPath, ok := ValidateFolderId(c, baseFolder, itemId)
	if !ok {
		return
	}

	fileInfo, err := ReadFirstFileInDir(itemPath)
	if HandleFileReadError(c, err) {
		return
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(fileInfo.Content))

	c.JSON(http.StatusOK, gin.H{
		itemType + "Id":   itemId,
		itemType + "Name": fileInfo.Name,
		itemType + "File": encoded,
		"createdAt":       fileInfo.CreationTime.Format(config.TimestampFormat),
	})
}

// GetAllItemsWithFileNames gets all items in a folder with their file names
func GetAllItemsWithFileNames(c *gin.Context, baseFolder, itemType string) {
	items, err := os.ReadDir(baseFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	var itemList []gin.H
	for _, item := range items {
		if item.IsDir() {
			itemPath := filepath.Join(baseFolder, item.Name())

			// Try to read first file for name and creation time
			fileInfo, err := ReadFirstFileInDir(itemPath)
			var fileName string
			var createdAt string

			if err == nil {
				fileName = fileInfo.Name
				createdAt = fileInfo.CreationTime.Format(config.TimestampFormat)
			} else {
				// Fallback to directory modification time
				dirInfo, dirErr := os.Stat(itemPath)
				if dirErr == nil {
					createdAt = dirInfo.ModTime().Format(config.TimestampFormat)
				}
			}

			itemList = append(itemList, gin.H{
				itemType + "Id":   item.Name(),
				itemType + "Name": fileName,
				"createdAt":       createdAt,
			})
		}
	}

	c.JSON(http.StatusOK, itemList)
}

// SaveUploadedFile handles a single uploaded file for an item folder.
// If providedId is empty a new ID is generated. It validates the file extension,
// optionally removes existing content when updating, saves the uploaded file and
// returns the item ID on success.
func SaveUploadedFile(c *gin.Context, baseFolder, providedId, expectedExt string) (string, bool) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	if filepath.Ext(file.Filename) != expectedExt {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}

	var itemId string
	var itemPath string

	if providedId != "" {
		// Validate existing folder
		validatedPath, ok := ValidateFolderId(c, baseFolder, providedId)
		if !ok {
			return "", false
		}
		itemId = providedId
		itemPath = validatedPath
	} else {
		// Create new folder with generated id
		newId, err := GenerateUniqueID(baseFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return "", false
		}
		itemId = newId
		itemPath = filepath.Join(baseFolder, itemId)
		if err := os.MkdirAll(itemPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return "", false
		}
	}

	// If updating, clean the folder
	if providedId != "" {
		if err := os.RemoveAll(itemPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return "", false
		}
		if err := os.MkdirAll(itemPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return "", false
		}
	}

	// Save uploaded file
	destPath := filepath.Join(itemPath, file.Filename)
	if err := c.SaveUploadedFile(file, destPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return "", false
	}

	return itemId, true
}

// HandleFileReadError handles common file reading errors with HTTP responses
func HandleFileReadError(c *gin.Context, err error) bool {
	if err == os.ErrNotExist {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return true
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return true
	}
	return false
}
