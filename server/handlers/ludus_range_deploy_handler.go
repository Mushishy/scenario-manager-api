package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func DeployRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}
	concurrentRequestsStr, ok := utils.GetRequiredQueryParam(c, "concurrentRequests")
	if !ok {
		return
	}

	concurrentRequests, err := strconv.Atoi(concurrentRequestsStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid concurrentRequests value"})
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedMainUserOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Set pool as deploying
	utils.SetPoolDeploymentState(poolId, utils.DeploymentStateDeploying)

	// Start deployment in background goroutine
	go func() {
		defer utils.ClearPoolDeploymentState(poolId) // Clean up when done

		payload := gin.H{"tags": "all", "force": true}

		// Process users in batches
		batchSize := concurrentRequests
		for i := 0; i < len(userIds); i += batchSize {
			end := i + batchSize
			if end > len(userIds) {
				end = len(userIds)
			}

			batch := userIds[i:end]

			// 1. Send deploy requests for this batch
			requests := make([]utils.LudusRequest, len(batch))
			for j, userID := range batch {
				requests[j] = utils.LudusRequest{
					Method:  "POST",
					URL:     config.LudusUrl + "/range/deploy/?userID=" + userID,
					Payload: payload,
					UserID:  userID,
				}
			}

			utils.MakeConcurrentLudusRequests(requests, apiKey, concurrentRequests)

			// Note: In async mode, we don't collect/return results
			// Use CheckRangeStatus to monitor deployment progress

			// 2. Wait for this batch to actually finish deploying (or get aborted)
			if !utils.WaitForBatchDeployment(poolId, batch, apiKey, 30*time.Second) {
				// Batch was aborted, stop processing remaining batches
				return
			}
		}
	}() // Close the goroutine

	// Return immediate response
	c.JSON(http.StatusOK, gin.H{
		"poolId":             poolId,
		"userCount":          len(userIds),
		"concurrentRequests": concurrentRequests,
	})
}

func CheckRangeStatus(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedAllUsers)

	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	mainUser, _ := utils.GetMainUserFromPool(poolId)

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
		// Skip if mainUser is specified and this response is not for the main user
		if mainUser != "" && resp.UserID != mainUser {
			if resp.Error != nil {
				results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
			} else {
				state := "unknown"
				if resp.Response != nil {
					if rangeState, exists := resp.Response.(map[string]interface{})["rangeState"]; exists {
						state = rangeState.(string)
					}
				}
				results = append(results, gin.H{"userId": resp.UserID, "state": state})
			}
			continue
		}

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

			if state != "SUCCESS" && state != "DEPLOYED" {
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
	concurrentRequestsStr, ok := utils.GetRequiredQueryParam(c, "concurrentRequests")
	if !ok {
		return
	}

	concurrentRequests, err := strconv.Atoi(concurrentRequestsStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid concurrentRequests value"})
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedMainUserOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Set pool as deploying
	utils.SetPoolDeploymentState(poolId, utils.DeploymentStateDeploying)

	// Start redeployment in background goroutine
	go func() {
		defer utils.ClearPoolDeploymentState(poolId) // Clean up when done

		// Process users in batches
		batchSize := concurrentRequests
		for i := 0; i < len(userIds); i += batchSize {
			end := i + batchSize
			if end > len(userIds) {
				end = len(userIds)
			}

			batch := userIds[i:end]

			// Step 1: Check current states for this batch
			checkRequests := make([]utils.LudusRequest, len(batch))
			for j, userID := range batch {
				checkRequests[j] = utils.LudusRequest{
					Method:  "GET",
					URL:     config.LudusUrl + "/range/?userID=" + userID,
					Payload: nil,
					UserID:  userID,
				}
			}

			checkResponses := utils.MakeConcurrentLudusRequests(checkRequests, apiKey, concurrentRequests)

			var usersToDestroy []string
			var usersToRedeploy []string

			// Step 2: Process states and determine actions
			for _, resp := range checkResponses {
				if resp.Error != nil {
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
					usersToDestroy = append(usersToDestroy, resp.UserID)
				case "DESTROYING":
					// Wait for these to finish destroying, then they'll be redeployed
				case "DESTROYED":
					usersToRedeploy = append(usersToRedeploy, resp.UserID)
				default:
					// Skip users in other states
				}
			}

			// Step 3: Destroy ranges that need destroying
			if len(usersToDestroy) > 0 {
				destroyRequests := make([]utils.LudusRequest, len(usersToDestroy))
				for j, userID := range usersToDestroy {
					destroyRequests[j] = utils.LudusRequest{
						Method:  "DELETE",
						URL:     config.LudusUrl + "/range/?userID=" + userID,
						Payload: nil,
						UserID:  userID,
					}
				}
				utils.MakeConcurrentLudusRequests(destroyRequests, apiKey, concurrentRequests)
			}

			// Step 4: Wait for all ranges in this batch to be destroyed
			allUsersInBatch := append(usersToDestroy, usersToRedeploy...)
			if len(allUsersInBatch) > 0 {
				if !utils.WaitForBatchDestroyed(poolId, allUsersInBatch, apiKey, 30*time.Second) {
					// Pool was aborted during destroy phase
					return
				}
			}

			// Step 5: Redeploy all ranges that were destroyed
			if len(allUsersInBatch) > 0 {
				redeployRequests := make([]utils.LudusRequest, len(allUsersInBatch))
				payload := gin.H{"tags": "all", "force": true}
				for j, userID := range allUsersInBatch {
					redeployRequests[j] = utils.LudusRequest{
						Method:  "POST",
						URL:     config.LudusUrl + "/range/deploy/?userID=" + userID,
						Payload: payload,
						UserID:  userID,
					}
				}
				utils.MakeConcurrentLudusRequests(redeployRequests, apiKey, concurrentRequests)

				// Step 6: Wait for this batch to finish deploying (or get aborted)
				if !utils.WaitForBatchDeployment(poolId, allUsersInBatch, apiKey, 30*time.Second) {
					// Some ranges were aborted, stop processing remaining batches
					return
				}
			}
		}
	}() // Close the goroutine

	// Return immediate response
	c.JSON(http.StatusOK, gin.H{
		"poolId":             poolId,
		"userCount":          len(userIds),
		"concurrentRequests": concurrentRequests,
	})
}

func AbortRange(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedMainUserOnly)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Set pool as aborted to stop any ongoing deployment processes
	utils.SetPoolDeploymentState(poolId, utils.DeploymentStateAborted)

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

	userIds, err := utils.GetUserIdsFromPool(poolId, utils.SharedMainUserOnly)
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

	utils.DeleteCtfdData(poolId)
	c.JSON(http.StatusOK, gin.H{"results": results})
}
