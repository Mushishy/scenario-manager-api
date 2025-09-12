package utils

import (
	"dulus/server/config"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// ReadPoolData reads and parses pool.json file
func ReadPoolData(poolPath string) (map[string]interface{}, error) {
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var poolData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&poolData); err != nil {
		return nil, err
	}
	return poolData, nil
}

// WritePoolData writes pool data to pool.json file
func WritePoolData(poolPath string, poolData map[string]interface{}) error {
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(poolData)
}

// ReadPoolDataWithResponse reads pool data and handles HTTP responses
func ReadPoolDataWithResponse(c *gin.Context, poolPath string) (map[string]interface{}, bool) {
	poolData, err := ReadPoolData(poolPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return nil, false
	}
	return poolData, true
}

// WritePoolDataWithResponse writes pool data and handles HTTP responses
func WritePoolDataWithResponse(c *gin.Context, poolPath string, poolData map[string]interface{}) bool {
	if err := WritePoolData(poolPath, poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return false
	}
	return true
}

func HasCtfdData(poolPath string) bool {
	ctfdDataPath := filepath.Join(poolPath, "ctfd_data.json")
	_, err := os.Stat(ctfdDataPath)
	return err == nil
}

// GetAllPools returns all pools with basic information (excluding sensitive data)
func GetAllPools(poolFolder string) ([]map[string]interface{}, error) {
	var pools []map[string]interface{}

	files, err := os.ReadDir(poolFolder)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			poolPath := filepath.Join(poolFolder, file.Name())
			poolData, err := ReadPoolData(poolPath)
			if err != nil {
				continue // Skip pools we can't read
			}

			// Remove sensitive data for list view
			delete(poolData, "mainUser")
			delete(poolData, "usersAndTeams")
			poolData["poolId"] = file.Name()

			// Set ctfdData flag based on file existence
			poolData["ctfdData"] = HasCtfdData(poolPath)

			// Get creation time from pool.json file
			poolJsonPath := filepath.Join(poolPath, "pool.json")
			if fileInfo, err := os.Stat(poolJsonPath); err == nil {
				poolData["createdAt"] = fileInfo.ModTime()
			}

			pools = append(pools, poolData)
		}
	}

	return pools, nil
}

// ExtractUserIds extracts user IDs from pool data
func ExtractUserIds(poolData map[string]interface{}) []string {
	var userIds []string

	if usersAndTeams, exists := poolData["usersAndTeams"]; exists {
		if usersList, ok := usersAndTeams.([]interface{}); ok {
			for _, user := range usersList {
				if userMap, ok := user.(map[string]interface{}); ok {
					if userId, exists := userMap["userId"]; exists {
						if userIdStr, ok := userId.(string); ok {
							userIds = append(userIds, userIdStr)
						}
					}
				}
			}
		}
	}

	return userIds
}

// DeleteCtfdData deletes the ctfd_data.json file from a pool directory
// Does not return an error if the file doesn't exist
func DeleteCtfdData(poolId string) error {
	ctfdDataPath := filepath.Join(config.PoolFolder, poolId, "ctfd_data.json")

	err := os.Remove(ctfdDataPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
