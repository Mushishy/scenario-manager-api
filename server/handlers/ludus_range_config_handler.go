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

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	userIds, ok := utils.GetUserIdsFromPool(c, poolId, utils.SharedMainUserOnly)
	if !ok {
		return
	}

	topologyPath, ok := utils.ValidateFolderId(c, config.TopologyConfigFolder, pool.TopologyId)
	if !ok {
		return
	}

	// Read topology file
	fileInfo, err := utils.ReadFirstFileInDir(topologyPath)
	if utils.HandleFileReadError(c, err) {
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	responses := utils.MakeConcurrentFileUploads(userIds, fileInfo.Content, true, apiKey, config.MaxConcurrentRequests)

	results := utils.ConvertResponsesToResults(responses)

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func GetRangeConfig(c *gin.Context) {
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

	topologyPath, ok := utils.ValidateFolderId(c, config.TopologyConfigFolder, pool.TopologyId)
	if !ok {
		return
	}

	expectedTopologyFile, err := utils.ReadFirstFileInDir(topologyPath)
	if utils.HandleFileReadError(c, err) {
		return
	}

	userIds, ok := utils.GetUserIdsFromPool(c, poolId, utils.SharedMainUserOnly)
	if !ok {
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
