package utils

import (
	"os"
	"path/filepath"
	"time"
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
	return !os.IsNotExist(err)
}

// GetAllItems returns all items in a directory with their basic info
func GetAllItems(baseFolder string) ([]ItemInfo, error) {
	var items []ItemInfo

	files, err := os.ReadDir(baseFolder)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			itemPath := filepath.Join(baseFolder, file.Name())
			fileInfo, err := os.Stat(itemPath)
			if err != nil {
				continue // Skip items we can't stat
			}

			items = append(items, ItemInfo{
				ID:           file.Name(),
				CreationTime: fileInfo.ModTime(),
			})
		}
	}

	return items, nil
}
