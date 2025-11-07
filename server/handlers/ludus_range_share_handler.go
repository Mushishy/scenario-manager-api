package handlers

import (
	"archive/zip"
	"bytes"
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetRangeAccess(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, ok := utils.GetUserIdsFromPool(c, poolId, utils.SharedUsersAndTeamsOnly)
	if !ok {
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/user/wireguard?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

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
	folderName := "ludus-wireguard-configs-pool-" + string(poolId)

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

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	if pool.Type != "SHARED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pool must be of type SHARED"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// each user shares to their main user
	var requests []utils.LudusRequest
	for _, userTeam := range pool.UsersAndTeams {
		if userTeam.MainUserId != "" {
			payload := gin.H{
				"action":       "grant",
				"targetUserID": userTeam.MainUserId,
				"sourceUserID": userTeam.UserId,
				"force":        true,
			}
			requests = append(requests, utils.LudusRequest{
				Method:  "POST",
				URL:     config.LudusUrl + "/range/access",
				Payload: payload,
				UserID:  userTeam.UserId,
			})
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	results := utils.ConvertResponsesToResults(responses)

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func UnshareRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	if pool.Type != "SHARED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pool must be of type SHARED"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// each user unshares from their main user
	var requests []utils.LudusRequest
	for _, userTeam := range pool.UsersAndTeams {
		if userTeam.MainUserId != "" {
			payload := gin.H{
				"action":       "revoke",
				"targetUserID": userTeam.MainUserId,
				"sourceUserID": userTeam.UserId,
				"force":        true,
			}
			requests = append(requests, utils.LudusRequest{
				Method:  "POST",
				URL:     config.LudusUrl + "/range/access",
				Payload: payload,
				UserID:  userTeam.UserId,
			})
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	results := utils.ConvertResponsesToResults(responses)

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func GetSharedRanges(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	// Check if pool type is SHARED
	if pool.Type != "SHARED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pool must be of type SHARED"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")
	response, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/range/access", nil, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Parse response as generic interface
	rangeAccessList, ok := response.([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response format"})
		return
	}

	allShared := true
	anyShared := false

	// Check each user-mainUser pair in the pool
	for _, userTeam := range pool.UsersAndTeams {
		if userTeam.MainUserId == "" {
			continue // Skip users without mainUserId
		}

		isShared := false

		// Check if this user is shared to their main user
		for _, item := range rangeAccessList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if targetUserID, exists := itemMap["targetUserID"]; exists && targetUserID == userTeam.MainUserId {
					if sourceUserIDs, exists := itemMap["sourceUserIDs"]; exists {
						if sourceList, ok := sourceUserIDs.([]interface{}); ok {
							for _, sourceID := range sourceList {
								if sourceID == userTeam.UserId {
									isShared = true
									anyShared = true
									break
								}
							}
						}
					}
				}
			}
			if isShared {
				break
			}
		}

		if !isShared {
			allShared = false
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"shared":   allShared,
		"unshared": !anyShared,
	})
}
