package handlers

import (
	"archive/zip"
	"bytes"
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetRangeAccess(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId)
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

	// Create ZIP file in memory
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	var successCount int
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
						configContent = fmt.Sprintf("%v", config)
					}
				}
			}
		}

		if configContent != "" {
			// Create file in zip
			fileName := fmt.Sprintf("%s.conf", resp.UserID)
			file, err := zipWriter.Create(fileName)
			if err != nil {
				continue
			}

			_, err = file.Write([]byte(configContent))
			if err != nil {
				continue
			}
			successCount++
		}
	}

	if successCount == 0 {
		zipWriter.Close() // Close even if no files
		c.JSON(http.StatusNotFound, gin.H{"error": "No configs found"})
		return
	}

	// IMPORTANT: Close zip writer BEFORE checking for errors or sending response
	err = zipWriter.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=wireguard-configs.zip")
	c.Header("Content-Length", fmt.Sprintf("%d", zipBuffer.Len()))

	// Send zip file
	c.Data(http.StatusOK, "application/zip", zipBuffer.Bytes())
}

func ShareRange(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Get targetId from query parameter
	targetId := c.Query("targetId")
	if targetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId)
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
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Get targetId from query parameter
	targetId := c.Query("targetId")
	if targetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId)
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
	apiKey := c.Request.Header.Get("X-API-Key")
	response, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/range/access", nil, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shared_ranges": response})
}
