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
	"github.com/xeipuuv/gojsonschema"
)

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
		filePath := filepath.Join(dataPath, "ctfd_data.json")
		file, err := os.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
			return
		}
		defer file.Close()

		var data map[string]interface{}
		if err := json.NewDecoder(file).Decode(&data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ctfdDataId": ctfdDataId,
			"ctfdData":   data["ctfd_data"],
		})
	} else {
		// List all ctfdDataId with their content
		folders, err := os.ReadDir(config.CtfdDataFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		var dataList []gin.H
		for _, folder := range folders {
			if folder.IsDir() {
				// Read the ctfd_data.json file for each folder
				filePath := filepath.Join(config.CtfdDataFolder, folder.Name(), "ctfd_data.json")
				file, err := os.Open(filePath)
				if err != nil {
					continue // Skip folders that don't have ctfd_data.json or can't be read
				}

				var data map[string]interface{}
				if err := json.NewDecoder(file).Decode(&data); err != nil {
					file.Close()
					continue // Skip files that can't be decoded
				}
				file.Close()

				dataList = append(dataList, gin.H{
					"ctfdDataId": folder.Name(),
					"ctfdData":   data["ctfd_data"],
				})
			}
		}

		c.JSON(http.StatusOK, dataList)
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
	if err := utils.ValidateTeamField(ctfdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate flags field consistency - if one user has flags, all users must have flags
	if err := validateFlagsConsistency(ctfdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Save the JSON data to a file
	filePath := filepath.Join(dataPath, "ctfd_data.json")
	file, err := os.Create(filePath) // Overwrite or create the file
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully"})
}

// validateFlagsConsistency ensures that if one user has flags set, all users must have flags set
func validateFlagsConsistency(ctfdData []interface{}) error {
	if len(ctfdData) == 0 {
		return nil
	}

	var hasFlags *bool = nil

	for _, item := range ctfdData {
		user, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		flags, flagsExist := user["flags"]
		userHasFlags := flagsExist && flags != nil

		// If flags is an array, check if it's not empty
		if userHasFlags {
			if flagsArray, ok := flags.([]interface{}); ok {
				userHasFlags = len(flagsArray) > 0
			}
		}

		if hasFlags == nil {
			hasFlags = &userHasFlags
		} else if *hasFlags != userHasFlags {
			return fmt.Errorf("inconsistent flags: if one user has flags, all users must have flags")
		}
	}

	return nil
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
