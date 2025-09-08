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

		// Validate mainUser is not in usersAndTeams (only if mainUser is provided)
		if mainUser, exists := input["mainUser"]; exists && mainUser != nil {
			if mainUserStr, ok := mainUser.(string); ok && mainUserStr != "" {
				if err := utils.ValidateMainUserNotInUsersAndTeams(mainUserStr, usersAndTeams); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
					return
				}
			}
		}

		input["usersAndTeams"] = utils.ProcessUsersAndTeams(usersAndTeams)
	}

	// Validate MainUser for SHARED or CTFD types
	if input["type"] == "SHARED" || input["type"] == "CTFD" {
		if mainUser, ok := input["mainUser"].(string); !ok || mainUser == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		} else {
			// Check if mainUser is already used in another pool
			isUsed, err := utils.IsMainUserAlreadyUsed(mainUser)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
				return
			}
			if isUsed {
				c.JSON(http.StatusConflict, gin.H{"error": "Main user is already assigned to another pool"})
				return
			}
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

	poolData, ok := utils.ReadPoolDataWithResponse(c, poolPath)
	if !ok {
		return
	}

	// Get new users from request
	newUsersAndTeams, ok := input["usersAndTeams"].([]interface{})
	if !ok || len(newUsersAndTeams) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Process new users to add userId (but don't validate yet)
	processedNewUsers := utils.ProcessUsersAndTeams(newUsersAndTeams)

	// Get existing users
	var existingUsersAndTeams []interface{}
	if existing, exists := poolData["usersAndTeams"]; exists {
		if existingUsers, ok := existing.([]interface{}); ok {
			existingUsersAndTeams = existingUsers
		}
	}

	// Combine existing and new users
	combinedUsers := append(existingUsersAndTeams, processedNewUsers...)

	// Validate the combined user list (includes team consistency check)
	if err := utils.ValidateUsersAndTeams(combinedUsers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate mainUser is not in combined users (only if mainUser exists)
	if mainUser, exists := poolData["mainUser"].(string); exists && mainUser != "" {
		if err := utils.ValidateMainUserNotInUsersAndTeams(mainUser, combinedUsers); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
	}

	// Update pool data with combined users
	poolData["usersAndTeams"] = combinedUsers

	if !utils.WritePoolDataWithResponse(c, poolPath, poolData) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Users added successfully"})
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
		poolData["ctfdData"] = utils.HasCtfdData(poolPath)

		// Get creation time from pool.json file (same logic as GetAllPools)
		poolJsonPath := filepath.Join(poolPath, "pool.json")
		if fileInfo, err := os.Stat(poolJsonPath); err == nil {
			poolData["createdAt"] = fileInfo.ModTime()
		}

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

func CheckUserIds(c *gin.Context) {
	input, ok := utils.ValidateJSONSchema(c, "file://schemas/check_userids_schema.json")
	if !ok {
		return
	}

	userIds, ok := input["userIds"].([]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Get all existing userIds from all pools
	existingUserIds, err := utils.GetAllUserIdsFromPools(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Check each userId
	var results []map[string]interface{}
	for _, userIdInterface := range userIds {
		if userId, ok := userIdInterface.(string); ok {
			exists := existingUserIds[userId]
			results = append(results, map[string]interface{}{
				"userId": userId,
				"exists": exists,
			})
		}
	}

	c.JSON(http.StatusOK, results)
}
