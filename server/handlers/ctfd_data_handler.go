package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

func GetCtfdLogins(c *gin.Context) {
	ctfdDataId := c.Query("ctfdDataId")
	if ctfdDataId != "" {
		// Validate the folder ID
		dataPath, err := utils.ValidateFolderID(config.CtfdDataFolder, ctfdDataId)
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
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	}
}

func GetCtfdData(c *gin.Context) {
	ctfdDataId := c.Query("ctfdDataId")

	if ctfdDataId != "" {
		// Validate the folder ID
		dataPath, err := utils.ValidateFolderID(config.CtfdDataFolder, ctfdDataId)
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
		// List all ctfdDataId with their content
		dataItems, err := utils.GetAllCTFdData(config.CtfdDataFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, dataItems)
	}
}

func PutCtfdData(c *gin.Context) {
	ctfdDataId := c.Query("ctfdDataId")
	var dataPath string
	var err error

	if ctfdDataId != "" {
		// Validate the folder ID if ctfdDataId is provided
		dataPath, err = utils.ValidateFolderID(config.CtfdDataFolder, ctfdDataId)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
	} else {
		// Generate a new unique ID if ctfdDataId is not provided
		ctfdDataId, err = utils.GenerateUniqueID(config.CtfdDataFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		dataPath = filepath.Join(config.CtfdDataFolder, ctfdDataId)

		// Create the directory for the new ID
		if err := os.MkdirAll(dataPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
	}

	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ctfd_data_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
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
	if err := utils.SaveCTFdData(dataPath, input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": ctfdDataId})
}

func DeleteCtfdData(c *gin.Context) {
	ctfdDataId := c.Query("ctfdDataId")

	// Validate the folder ID
	dataPath, err := utils.ValidateFolderID(config.CtfdDataFolder, ctfdDataId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Remove the folder
	if err := os.RemoveAll(dataPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}
