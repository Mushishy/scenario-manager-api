package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ImportUsers(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, ok := utils.GetUserIdsFromPool(c, poolId, utils.SharedAllUsers)
	if !ok {
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		payload := gin.H{
			"name":    userID,
			"userID":  userID,
			"isAdmin": false,
		}
		requests[i] = utils.LudusRequest{
			Method:  "POST",
			URL:     config.LudusAdminUrl + "/user",
			Payload: payload,
			UserID:  userID,
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	results := utils.ConvertResponsesToResults(responses)

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func DeleteUsers(c *gin.Context) {
	poolId := utils.GetOptionalQueryParam(c, "poolId")

	var userIds []string

	if poolId != "" {
		// Delete users by poolId
		var ok bool
		userIds, ok = utils.GetUserIdsFromPool(c, poolId, utils.SharedAllUsers)
		if !ok {
			return
		}
	} else {
		// Delete users by userIds from request body
		var requestBody struct {
			UserIds []string `json:"userIds" binding:"required"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}

		userIds = requestBody.UserIds
	}

	if len(userIds) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No users to delete"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = utils.LudusRequest{
			Method:  "DELETE",
			URL:     config.LudusAdminUrl + "/user/" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	results := utils.ConvertResponsesToResults(responses)

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func CheckUsers(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, ok := utils.GetUserIdsFromPool(c, poolId, utils.SharedAllUsers)
	if !ok {
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/user?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Collect IDs of users that don't exist
	var missingUserIds []string
	for _, resp := range responses {
		exists := false

		if resp.Error == nil && resp.Response != nil {
			// Check if response is an empty array
			if respArray, ok := resp.Response.([]interface{}); ok {
				exists = len(respArray) > 0
			} else {
				// If it's not an array, assume user exists
				exists = true
			}
		}

		if !exists {
			missingUserIds = append(missingUserIds, resp.UserID)
		}
	}

	allExist := len(missingUserIds) == 0

	c.JSON(http.StatusOK, gin.H{
		"missingUserIds": missingUserIds,
		"allExist":       allExist,
	})
}

func GetAllMainUsers(c *gin.Context) {
	current := utils.GetOptionalQueryParam(c, "current")
	apiKey := c.Request.Header.Get("X-API-Key")

	// Get all users from Ludus API
	response, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/user/all", nil, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users from Ludus"})
		return
	}

	// Parse response to extract userIDs
	var allUserIds []string
	if userArray, ok := response.([]interface{}); ok {
		for _, user := range userArray {
			if userMap, ok := user.(map[string]interface{}); ok {
				if userID, exists := userMap["userID"].(string); exists {
					allUserIds = append(allUserIds, userID)
				}
			}
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid response format from Ludus"})
		return
	}

	// Get main users from pools
	existingMainUsers, err := utils.GetAllMainUsersFromPools()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve main users"})
		return
	}

	// Convert main users map to array
	var mainUserIds []string
	for userId := range existingMainUsers {
		mainUserIds = append(mainUserIds, userId)
	}

	// If current parameter is set, return just the main users
	if current != "" {
		c.JSON(http.StatusOK, gin.H{"userIds": mainUserIds})
		return
	}

	// Get all users (both userIds and mainUserIds) from all pools
	allPoolUsers, err := utils.GetAllUserIdsFromPools(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pool users"})
		return
	}

	// Filter out ALL pool users (regular + main) and ROOT from all users
	var filteredUserIds []string
	for _, userID := range allUserIds {
		if !allPoolUsers[userID] && userID != "ROOT" {
			filteredUserIds = append(filteredUserIds, userID)
		}
	}

	// Check for optional poolId parameter
	poolId := utils.GetOptionalQueryParam(c, "poolId")
	if poolId != "" {
		// Validate poolId and get pool path
		poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
		if !ok {
			return
		}

		// Get pool data to extract main users for this specific pool
		pool, ok := utils.ReadPoolWithResponse(c, poolPath)
		if !ok {
			return
		}

		// Extract both userIds and mainUserIds from this specific pool
		// so users already assigned to this pool are available for re-selection
		poolUserIds, poolMainUserIds := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
		allPoolSpecificUsers := append(poolUserIds, poolMainUserIds...)

		for _, userId := range allPoolSpecificUsers {
			// Avoid duplicates
			found := false
			for _, existing := range filteredUserIds {
				if existing == userId {
					found = true
					break
				}
			}
			if !found {
				filteredUserIds = append(filteredUserIds, userId)
			}
		}
	}

	isCtfd := utils.GetOptionalQueryParam(c, "isCtfd")

	// If isCtfd parameter is set, filter to only users starting with "CTFD"
	if isCtfd != "" {
		var ctfdUserIds []string
		for _, userID := range filteredUserIds {
			if len(userID) >= 4 && userID[:4] == "CTFD" {
				ctfdUserIds = append(ctfdUserIds, userID)
			}
		}
		filteredUserIds = ctfdUserIds
	}

	c.JSON(http.StatusOK, gin.H{"userIds": filteredUserIds})
}
