package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func DeployRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
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
	payload := gin.H{"tags": "all", "force": true}

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = utils.LudusRequest{
			Method:  "POST",
			URL:     config.LudusUrl + "/range/deploy/?userID=" + userID,
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

func CheckRangeStatus(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
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
			URL:     config.LudusUrl + "/range/?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Convert to results format and check if all are deployed
	var results []gin.H
	allDeployed := true

	for _, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
			allDeployed = false
		} else {
			state := "unknown"
			if resp.Response != nil {
				if rangeState, exists := resp.Response.(map[string]interface{})["rangeState"]; exists {
					state = rangeState.(string)
				}
			}

			if state != "DEPLOYED" {
				allDeployed = false
			}

			results = append(results, gin.H{"userId": resp.UserID, "state": state})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results":     results,
		"allDeployed": allDeployed,
	})
}

func RedeployRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
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

	// First, get current states for all users
	checkRequests := make([]utils.LudusRequest, len(userIds))
	for i, userID := range userIds {
		checkRequests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/range/?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	checkResponses := utils.MakeConcurrentLudusRequests(checkRequests, apiKey, config.MaxConcurrentRequests)

	// Process based on current state
	var results []gin.H
	for _, resp := range checkResponses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
			continue
		}

		state := "unknown"
		if resp.Response != nil {
			if rangeState, exists := resp.Response.(map[string]interface{})["rangeState"]; exists {
				state = rangeState.(string)
			}
		}

		switch state {
		case "ERROR", "ABORTED":
			// Destroy range
			utils.MakeLudusRequest("DELETE", config.LudusUrl+"/range/?userID="+resp.UserID, nil, apiKey)
			results = append(results, gin.H{"userId": resp.UserID, "action": "destroyed", "message": "Wait and redeploy"})
		case "DESTROYING":
			results = append(results, gin.H{"userId": resp.UserID, "action": "waiting", "message": "Wait until destroyed"})
		case "DESTROYED":
			// Redeploy
			payload := gin.H{"tags": "all", "force": true}
			utils.MakeLudusRequest("POST", config.LudusUrl+"/range/deploy/?userID="+resp.UserID, payload, apiKey)
			results = append(results, gin.H{"userId": resp.UserID, "action": "redeployed"})
		default:
			results = append(results, gin.H{"userId": resp.UserID, "action": "skipped", "state": state})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func AbortRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
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
			Method:  "POST",
			URL:     config.LudusUrl + "/range/abort/?userID=" + userID,
			Payload: nil,
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

func RemoveRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
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
			Method:  "DELETE",
			URL:     config.LudusUrl + "/range/?userID=" + userID,
			Payload: nil,
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
