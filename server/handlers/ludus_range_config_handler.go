package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetRangeConfig(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	topologyId, ok := utils.GetRequiredQueryParam(c, "topologyId")
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

	// Validate topology exists
	topologyPath, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId)
	if !ok {
		return
	}

	// Read topology file
	fileInfo, err := utils.ReadFirstFileInDir(topologyPath)
	if utils.HandleFileReadError(c, err) {
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Upload to all users
	responses := utils.MakeConcurrentFileUploads(userIds, fileInfo.Content, true, apiKey, config.MaxConcurrentRequests)

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
			allSame = false
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
