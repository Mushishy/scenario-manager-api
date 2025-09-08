package handlers

import (
	"archive/zip"
	"bytes"
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RangeAccessItem represents a single range access entry from the Ludus API
type RangeAccessItem struct {
	TargetUserID  string   `json:"targetUserID"`
	SourceUserIDs []string `json:"sourceUserIDs"`
}

// RangeAccessResponse represents the response from /range/access endpoint
type RangeAccessResponse []RangeAccessItem

func GetRangeAccess(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedUsersAndTeamsOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/user/wireguard?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Collect valid configs first
	var validConfigs []struct {
		userID  string
		content string
	}

	for _, resp := range responses {
		if resp.Error != nil {
			continue // Skip failed requests
		}

		// Extract WireGuard config from nested response structure
		var configContent string
		if wireguardResp, ok := resp.Response.(map[string]interface{}); ok {
			if result, exists := wireguardResp["result"]; exists {
				if resultMap, ok := result.(map[string]interface{}); ok {
					if config, exists := resultMap["wireGuardConfig"]; exists {
						if configStr, ok := config.(string); ok {
							configContent = configStr
						} else {
							configContent = fmt.Sprintf("%v", config)
						}
					}
				}
			}
		}

		if configContent != "" {
			validConfigs = append(validConfigs, struct {
				userID  string
				content string
			}{
				userID:  resp.UserID,
				content: configContent,
			})
		}
	}

	if len(validConfigs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No configs found"})
		return
	}

	// Create ZIP file in memory
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// Create folder structure and add files
	folderName := "wireguard-configs/"

	for _, config := range validConfigs {
		fileName := folderName + config.userID + ".conf"

		// Create file in ZIP
		fileWriter, err := zipWriter.Create(fileName)
		if err != nil {
			zipWriter.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ZIP file"})
			return
		}

		// Write content to file
		_, err = fileWriter.Write([]byte(config.content))
		if err != nil {
			zipWriter.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to ZIP file"})
			return
		}
	}

	// Close ZIP writer properly
	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize ZIP file"})
		return
	}

	// Get the ZIP data and encode as base64
	zipData := zipBuffer.Bytes()
	base64Data := base64.StdEncoding.EncodeToString(zipData)
	filename := string(poolId) + ".zip"

	// Return JSON response with base64 encoded ZIP
	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"data":     base64Data,
		"size":     len(zipData),
	})
}

func ShareRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	targetId, ok := utils.GetRequiredQueryParam(c, "targetId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedUsersAndTeamsOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		payload := gin.H{
			"action":       "grant",
			"targetUserID": targetId,
			"sourceUserID": userID,
			"force":        true,
		}
		requests[i] = utils.LudusRequest{
			Method:  "POST",
			URL:     config.LudusUrl + "/range/access",
			Payload: payload,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

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

func UnshareRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	targetId, ok := utils.GetRequiredQueryParam(c, "targetId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedUsersAndTeamsOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		payload := gin.H{
			"action":       "revoke",
			"targetUserID": targetId,
			"sourceUserID": userID,
			"force":        true,
		}
		requests[i] = utils.LudusRequest{
			Method:  "POST",
			URL:     config.LudusUrl + "/range/access",
			Payload: payload,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

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

func GetSharedRanges(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	targetId, ok := utils.GetRequiredQueryParam(c, "targetId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedUsersAndTeamsOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")
	response, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/range/access", nil, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Parse response into our struct
	var rangeAccess RangeAccessResponse
	responseBytes, _ := json.Marshal(response)
	json.Unmarshal(responseBytes, &rangeAccess)

	shared := false
	unshared := true

	// Find the target user and check if all pool users are in sourceUserIDs
	for _, item := range rangeAccess {
		if item.TargetUserID == targetId {
			unshared = false
			if len(userIds) > 0 && utils.ContainsAll(item.SourceUserIDs, userIds) {
				shared = true
			}
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"shared":   shared,
		"unshared": unshared,
	})
}
