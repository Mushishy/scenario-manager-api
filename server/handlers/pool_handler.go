package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func PostPool(c *gin.Context) {
	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_schema.json")
	if !ok {
		return
	}

	// Check that we can pull the userID and apikey from what the user provided
	APIKey := c.Request.Header.Get("X-API-Key")
	userID, ok := utils.ExtractUserIDFromAPIKey(c, APIKey)
	if !ok {
		return
	}

	// Add createdBy to input
	input["createdBy"] = userID

	// Validate TopologyId
	topologyId := input["topologyId"].(string)
	if _, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId); !ok {
		return
	}

	// Process UsersAndTeams to add userId
	if usersAndTeams, ok := input["usersAndTeams"].([]interface{}); ok && len(usersAndTeams) > 0 {
		if err := utils.ValidateUsersAndTeams(usersAndTeams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
		input["usersAndTeams"] = utils.ProcessUsersAndTeams(usersAndTeams)
	}

	// Validate MainUser for SHARED or INDIVIDUAL types
	if input["type"] == "SHARED" || input["type"] == "INDIVIDUAL" {
		if mainUser, ok := input["mainUser"].(string); !ok || mainUser == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
	}

	// Generate pool id and create folder
	poolId, err := utils.GenerateUniqueID(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	poolPath := filepath.Join(config.PoolFolder, poolId)
	if err := os.MkdirAll(poolPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Save pool data
	if !utils.WritePoolDataWithResponse(c, poolPath, input) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": poolId})
}

// TODO
// when I add users I have to remove flags.json
func PatchPoolUsers(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_users_schema.json")
	if !ok {
		return
	}

	// Process UsersAndTeams
	if usersAndTeams, ok := input["usersAndTeams"].([]interface{}); ok && len(usersAndTeams) > 0 {
		if err := utils.ValidateUsersAndTeams(usersAndTeams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
		input["usersAndTeams"] = utils.ProcessUsersAndTeams(usersAndTeams)
	}

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	poolData["usersAndTeams"] = input["usersAndTeams"]

	if !utils.WritePoolDataWithResponse(c, poolPath, poolData) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolTopology(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_topology_schema.json")
	if !ok {
		return
	}

	// Validate TopologyId exists
	topologyId := input["topologyId"].(string)
	if _, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId); !ok {
		return
	}

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	poolData["topologyId"] = topologyId

	if !utils.WritePoolDataWithResponse(c, poolPath, poolData) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolNote(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_note_schema.json")
	if !ok {
		return
	}

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	poolData["note"] = input["note"]

	if !utils.WritePoolDataWithResponse(c, poolPath, poolData) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func GetPool(c *gin.Context) {
	poolId := utils.GetOptionalQueryParam(c, "poolId")
	userIds := utils.GetOptionalQueryParam(c, "userIds")

	if poolId != "" {
		poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
		if !ok {
			return
		}

		poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
		if !ok {
			return
		}

		if userIds == "true" {
			userIdList := utils.ExtractUserIds(poolData)
			c.JSON(http.StatusOK, gin.H{"poolId": poolId, "userIds": userIdList})
			return
		}

		poolData["poolId"] = poolId
		c.JSON(http.StatusOK, poolData)
		return
	}

	// Return all pools
	pools, err := utils.GetAllPools(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, pools)
}

func DeletePool(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	if err := os.RemoveAll(poolPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}
