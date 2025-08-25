package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetCtfdLogins(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	dataPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	// Read the ctfd_data.json file
	data, err := utils.ReadCTFdJSON(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Extract ctfd_data array
	ctfdData, ok := data["ctfd_data"].([]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Build CSV string
	var csvLines []string
	for _, item := range ctfdData {
		user, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		username, _ := user["user"].(string)
		password, _ := user["password"].(string)
		team, teamExists := user["team"].(string)

		// Create CSV line: "username, password[, team]"
		var csvLine string
		if teamExists && team != "" {
			csvLine = fmt.Sprintf("%s, %s, %s", username, password, team)
		} else {
			csvLine = fmt.Sprintf("%s, %s", username, password)
		}
		csvLines = append(csvLines, csvLine)
	}

	// Join all lines with newlines
	csvOutput := strings.Join(csvLines, "\n")

	// Return as plain text
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, csvOutput)
}

func GetCtfdData(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	// Validate the folder id
	dataPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Read the ctfd_data.json file
	data, err := utils.ReadCTFdJSON(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ctfdData": data["ctfd_data"],
	})
}

func PutCtfdData(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	userIds, err := utils.GetUserIdsFromPool(poolId)
	if err != nil {
		if err.Error() == "pool not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		}
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Check if all ranges are deployed
	if !utils.AllRangesDeployed(userIds, apiKey, c) {
		return
	}

	// Read pool.json to get user details
	userDetailMap, err := utils.GetUserDetailsFromPool(poolPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract flags from logs for all users
	ctfdUsers := utils.ExtractFlagsFromLogs(userIds, userDetailMap, apiKey)

	// Prepare the data structure for saving
	ctfdData := utils.CtfdData{CtfdData: ctfdUsers}
	dataToSave := map[string]interface{}{"ctfd_data": ctfdUsers}

	// Save the new data to file
	if err := utils.SaveCTFdData(poolPath, dataToSave); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Flags extracted and saved successfully",
		"poolId":    poolId,
		"ctfd_data": ctfdData.CtfdData,
	})
}
