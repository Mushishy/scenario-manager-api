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

	// Get pool data to retrieve topology ID
	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	topologyId := poolData["topologyId"].(string)

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

	// Get pool data to retrieve topology ID
	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	topologyId := poolData["topologyId"].(string)

	// Get the expected topology configuration
	topologyPath, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId)
	if !ok {
		return
	}

	expectedTopologyFile, err := utils.ReadFirstFileInDir(topologyPath)
	if utils.HandleFileReadError(c, err) {
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

	// Check each user's config against the expected topology
	matchPoolTopology := true

	for _, userID := range userIds {
		request := utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/range/config/?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}

		response := utils.MakeConcurrentLudusRequests([]utils.LudusRequest{request}, apiKey, 1)[0]

		if response.Error != nil {
			matchPoolTopology = false
			break
		}

		// Extract the actual config content from the response
		var userConfigContent string
		if responseMap, ok := response.Response.(map[string]interface{}); ok {
			if result, exists := responseMap["result"]; exists {
				if resultStr, ok := result.(string); ok {
					userConfigContent = resultStr
				} else {
					matchPoolTopology = false
					break
				}
			} else {
				matchPoolTopology = false
				break
			}
		} else {
			matchPoolTopology = false
			break
		}

		// Compare the topology content with the user's config content
		if !utils.CompareConfigs(expectedTopologyFile.Content, userConfigContent) {
			matchPoolTopology = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"matchPoolTopology": matchPoolTopology,
	})
}
