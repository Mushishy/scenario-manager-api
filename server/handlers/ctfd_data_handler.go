package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// if one user has flags all users must have flags !

// GetCtfdLogins for a pool and return as CSV of login credentials to ctfd
func GetCtfdLogins(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	ctfdData, ok := utils.ReadCTFdJSON(c, poolPath)
	if !ok {
		return
	}

	var csvOutput strings.Builder
	for i, user := range ctfdData.CtfdData {
		if i > 0 {
			csvOutput.WriteString("\n")
		}

		username := user.User
		password := user.Password
		team := user.Team

		// Create CSV line "username, password, team"
		if team != "" {
			csvOutput.WriteString(fmt.Sprintf("%s, %s, %s", username, password, team))
		} else {
			csvOutput.WriteString(fmt.Sprintf("%s, %s", username, password))
		}
	}

	// Return as plain text
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, csvOutput.String())
}

func GetCtfdData(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	ctfdData, ok := utils.ReadCTFdJSON(c, poolPath)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ctfdData": ctfdData.CtfdData,
	})
}

// Retrieve flags from deployed pools into ctfd_data.json
func PutCtfdData(c *gin.Context) {
	apiKey := c.Request.Header.Get("X-API-Key")
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

	var users []string
	if pool.Type == "SHARED" {
		_, mainUsers := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = mainUsers
	} else {
		userIds, _ := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = userIds
	}

	if !utils.AllRangesDeployed(users, apiKey, c) {
		return
	}

	flagsMap, ok := utils.GetFlagsForUsers(c, users, apiKey)
	if !ok {
		return
	}

	var ctfdUsers []utils.CtfdUser
	for _, userTeam := range pool.UsersAndTeams {
		var lookupKey string
		if pool.Type == "SHARED" {
			lookupKey = userTeam.MainUserId
		} else {
			lookupKey = userTeam.UserId
		}

		flags, exists := flagsMap[lookupKey]
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No flags found for user: " + lookupKey})
			return
		}
		if flags == nil {
			flags = []utils.Flag{}
		}

		ctfdUser := utils.CtfdUser{
			User:     strings.ReplaceAll(userTeam.User, " ", ""),
			Password: utils.RandomLowercaseString(5),
			Team:     userTeam.Team,
			Flags:    flags,
		}
		ctfdUsers = append(ctfdUsers, ctfdUser)
	}

	if !utils.SaveCTFdData(c, poolPath, ctfdUsers) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Flags extracted and saved successfully",
		"poolId":    poolId,
		"ctfd_data": ctfdUsers,
	})
}
