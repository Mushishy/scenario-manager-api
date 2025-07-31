package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

// Range endpoints
func SetRangeConfig(c *gin.Context) {
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ludus_users_schema.json")

	var input struct {
		UserIds []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Get topologyId from query parameter
	topologyId := c.Query("topologyId")
	if topologyId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate if topology exists and get the path
	topologyPath, err := utils.ValidateFolderID(config.TopologyConfigFolder, topologyId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Read the first file in the topology folder (same as GetTopology)
	files, err := os.ReadDir(topologyPath)
	if err != nil || len(files) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Find the first non-directory file
	var configFilePath string
	for _, file := range files {
		if !file.IsDir() {
			configFilePath = filepath.Join(topologyPath, file.Name())
			break
		}
	}

	if configFilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Read the config file content
	configContent, err := os.ReadFile(configFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests for file upload
	responses := utils.MakeConcurrentFileUploads(input.UserIds, string(configContent), true, apiKey, config.MaxConcurrentRequests)

	// Convert to results format
	var results []gin.H
	for _, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
		} else {
			results = append(results, gin.H{"userId": resp.UserID, "response": resp.Response})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func GetRangeConfig(c *gin.Context) {
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ludus_users_schema.json")

	var input struct {
		UserIds []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(input.UserIds))
	for i, userID := range input.UserIds {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/range/config/?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Convert to results format and check if all configs are the same
	var results []gin.H
	var firstConfig interface{}
	allSame := true

	for i, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
			allSame = false // Error means not all are the same
		} else {
			results = append(results, gin.H{"userId": resp.UserID, "config": resp.Response})

			// Check if all configs are the same
			if i == 0 {
				firstConfig = resp.Response
			} else if !utils.CompareConfigs(firstConfig, resp.Response) {
				allSame = false
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"allSame": allSame,
	})
}
