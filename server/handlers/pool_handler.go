package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/json"
	"fmt"
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

}

func PostPoolDev(c *gin.Context) {
	// Get API key from header
	APIKey := c.Request.Header.Get("X-API-Key")
	userID, ok := utils.ExtractUserIDFromAPIKey(c, APIKey)
	if !ok {
		return
	}

	// Get username from Ludus API
	userResponse, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/user", nil, APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Extract username from response
	var username string
	if userMap, ok := userResponse.(map[string]interface{}); ok {
		if usernameVal, exists := userMap["username"]; exists {
			if usernameStr, ok := usernameVal.(string); ok {
				username = usernameStr
			} else {
				username = userID // fallback to userID
			}
		} else {
			username = userID // fallback to userID
		}
	} else {
		username = userID // fallback to userID
	}

	// Parse request body to get note
	var requestBody struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
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

	// Create the fixed pool structure
	poolData := map[string]interface{}{
		"createdBy":  userID,
		"note":       requestBody.Note,
		"topologyId": "ctfdev",
		"type":       "INDIVIDUAL",
		"usersAndTeams": []map[string]interface{}{
			{
				"user":   username,
				"userId": userID,
			},
		},
	}

	// Save pool data
	if !utils.WritePoolDataWithResponse(c, poolPath, poolData) {
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

	// Combine existing and new users
	combinedUsers := append(existingUsersAndTeams, newUsersAndTeams...)

	// Validate and process the combined user list
	poolType := pool.Type
	processedUsers, err := utils.ValidateAndProcessUsersAndTeams(combinedUsers, poolType, utils.OperationAdd)
	if err != nil {
		fmt.Println(err)
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
