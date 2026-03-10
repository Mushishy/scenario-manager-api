package utils

import (
	"dulus/server/config"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// ReadPoolInternal reads pool data without HTTP handling (for internal use)
func ReadPoolInternal(poolPath string) (Pool, error) {
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		return Pool{}, err
	}
	defer file.Close()

	var pool Pool
	if err := json.NewDecoder(file).Decode(&pool); err != nil {
		return Pool{}, err
	}
	return pool, nil
}

// ReadPoolWithResponse reads pool data and handles HTTP responses
func ReadPoolWithResponse(c *gin.Context, poolPath string) (Pool, bool) {
	pool, err := ReadPoolInternal(poolPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return Pool{}, false
	}
	return pool, true
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

// WritePoolDataWithResponse writes pool data and handles HTTP responses
func WritePoolDataWithResponse(c *gin.Context, poolPath string, poolData map[string]interface{}) bool {
	if err := WritePoolData(poolPath, poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return false
	}
	return true
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
			pool, err := ReadPoolInternal(poolPath)
			if err != nil {
				continue // Skip pools we can't read
			}

			// Create pool data map for list view (without sensitive data)
			poolData := map[string]interface{}{
				"poolId":     file.Name(),
				"createdBy":  pool.CreatedBy,
				"note":       pool.Note,
				"topologyId": pool.TopologyId,
				"type":       pool.Type,
				"ctfdData":   HasCtfdData(poolPath),
			}

			// Get creation time from pool.json file
			poolJsonPath := filepath.Join(poolPath, "pool.json")
			if fileInfo, err := os.Stat(poolJsonPath); err == nil {
				poolData["createdAt"] = fileInfo.ModTime().Format(config.TimestampFormat)
			}

			pools = append(pools, poolData)
		}
	}

	return pools, nil
}

// ExecuteTestingAction is a generic helper function for testing actions
func ExecuteTestingAction(c *gin.Context, endpoint string, payload interface{}) {
	poolId, ok := GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	var users []string
	if pool.Type == "SHARED" {
		_, mainUsers := ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = mainUsers
	} else {
		userIds, _ := ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = userIds
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]LudusRequest, len(users))
	for i, userID := range users {
		requests[i] = LudusRequest{
			Method:  "PUT",
			URL:     config.LudusUrl + endpoint + "?userID=" + userID,
			Payload: payload,
			UserID:  userID,
		}
	}

	responses := MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)
	results := ConvertResponsesToResults(responses)
	c.JSON(http.StatusOK, gin.H{"results": results})
}
