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
