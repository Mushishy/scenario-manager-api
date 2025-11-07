package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/json"
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
	poolType, _ := input["type"].(string)

	// Validate TopologyId
	topologyId := input["topologyId"].(string)
	if _, ok := utils.ValidateFolderId(c, config.TopologyConfigFolder, topologyId); !ok {
		return
	}

	// Validate and process UsersAndTeams
	if usersAndTeams, ok := input["usersAndTeams"].([]interface{}); ok && len(usersAndTeams) > 0 {
		processedUsers, err := utils.ValidateAndProcessUsersAndTeams(usersAndTeams, poolType, utils.OperationCreate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
		input["usersAndTeams"] = processedUsers
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

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_topology_schema.json")
	if !ok {
		return
	}

	// Validate TopologyId exists
	topologyId := input["topologyId"].(string)
	if _, ok := utils.ValidateFolderId(c, config.TopologyConfigFolder, topologyId); !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	pool.TopologyId = topologyId

	// Convert to map for existing write helper
	poolBytes, _ := json.Marshal(pool)
	var poolMap map[string]interface{}
	json.Unmarshal(poolBytes, &poolMap)

	if !utils.WritePoolDataWithResponse(c, poolPath, poolMap) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolNote(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_note_schema.json")
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	if noteStr, ok := input["note"].(string); ok {
		pool.Note = noteStr
	}

	poolBytes, _ := json.Marshal(pool)
	var poolMap map[string]interface{}
	json.Unmarshal(poolBytes, &poolMap)

	if !utils.WritePoolDataWithResponse(c, poolPath, poolMap) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolUsers(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/pool_users_schema.json")
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	// Get new users from request
	newUsersAndTeams, ok := input["usersAndTeams"].([]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Convert existing pool.UsersAndTeams into []interface{}
	existingBytes, _ := json.Marshal(pool.UsersAndTeams)
	var existingUsersAndTeams []interface{}
	json.Unmarshal(existingBytes, &existingUsersAndTeams)

	// For SHARED pools, validate that new users' mainUserIds match existing ones
	poolType := pool.Type
	if poolType == "SHARED" && len(existingUsersAndTeams) > 0 {
		// Create hashmap of existing mainUserIds
		existingMainUserIds := make(map[string]bool)
		for _, item := range existingUsersAndTeams {
			if userMap, ok := item.(map[string]interface{}); ok {
				if mainUserId, exists := userMap["mainUserId"].(string); exists && mainUserId != "" {
					existingMainUserIds[mainUserId] = true
				}
			}
		}

		// Check that all new users' mainUserIds are in existing mainUserIds
		for _, item := range newUsersAndTeams {
			if userMap, ok := item.(map[string]interface{}); ok {
				if mainUserId, exists := userMap["mainUserId"].(string); exists && mainUserId != "" {
					if !existingMainUserIds[mainUserId] {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
						return
					}
				}
			}
		}
	}

	// Combine existing and new users
	combinedUsers := append(existingUsersAndTeams, newUsersAndTeams...)

	// Validate and process the combined user list
	processedUsers, err := utils.ValidateAndProcessUsersAndTeams(combinedUsers, poolType, utils.OperationAdd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Convert pool struct to map and set processed users
	poolBytes, _ := json.Marshal(pool)
	var poolMap map[string]interface{}
	json.Unmarshal(poolBytes, &poolMap)
	poolMap["usersAndTeams"] = processedUsers

	if !utils.WritePoolDataWithResponse(c, poolPath, poolMap) {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Users added successfully"})
}

func GetPool(c *gin.Context) {
	poolId := utils.GetOptionalQueryParam(c, "poolId")
	userIds := utils.GetOptionalQueryParam(c, "userIds")
	mainUsers := utils.GetOptionalQueryParam(c, "mainUsers")

	if mainUsers != "" && userIds != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	if poolId != "" {
		poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
		if !ok {
			return
		}

		pool, ok := utils.ReadPoolWithResponse(c, poolPath)
		if !ok {
			return
		}

		// Convert pool struct to map for endpoints that expect a map
		poolBytes, _ := json.Marshal(pool)
		var poolMap map[string]interface{}
		json.Unmarshal(poolBytes, &poolMap)

		if userIds == "true" {
			userIdList, _ := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
			c.JSON(http.StatusOK, gin.H{"poolId": poolId, "userIds": userIdList})
			return
		}

		if mainUsers == "true" {
			_, mainUsersList := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
			c.JSON(http.StatusOK, gin.H{"mainUsers": mainUsersList})
			return
		}

		poolMap["poolId"] = poolId
		poolMap["ctfdData"] = utils.HasCtfdData(poolPath)

		// Get creation time from pool.json file (same logic as GetAllPools)
		poolJsonPath := filepath.Join(poolPath, "pool.json")
		if fileInfo, err := os.Stat(poolJsonPath); err == nil {
			poolMap["createdAt"] = fileInfo.ModTime()
		}

		c.JSON(http.StatusOK, poolMap)
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

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
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
