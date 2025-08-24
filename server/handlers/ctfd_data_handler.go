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
	poolId := c.Query("poolId")

	if poolId != "" {
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
	} else {
		// List all poolId with their content
		dataItems, err := utils.GetAllCTFdData(config.PoolFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, dataItems)
	}
}

// TODO ADD OPTION TO EXTRACT IT FROM LOGS
func PutCtfdData(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderWithResponse(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	input, ok := utils.ValidateJSONSchema(c, "file://schemas/ctfd_data_schema.json")
	if !ok {
		return
	}

	// Custom validation for 'team' field and 'flags' field
	ctfdData, ok := input["ctfd_data"].([]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate team field consistency
	if err := utils.ValidateUsersAndTeams(ctfdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate flags field consistency
	if err := utils.ValidateFlagsConsistency(ctfdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Save the JSON data to a file
	if err := utils.SaveCTFdData(poolPath, input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": poolId})
}
