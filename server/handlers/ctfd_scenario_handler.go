package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func GetScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")

	if scenarioID != "" {
		scenarioPath, err := utils.ValidateFolderID(config.ScenarioFolder, scenarioID)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}

		files, err := os.ReadDir(scenarioPath)
		if err != nil || len(files) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"scenarioID":   scenarioID,
			"scenarioName": files[0].Name(),
			"scenarioFile": filepath.Join(scenarioPath, files[0].Name()),
		})
	} else {
		scenarios, err := os.ReadDir(config.ScenarioFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		var scenarioList []gin.H
		for _, scenario := range scenarios {
			if scenario.IsDir() {
				scenarioList = append(scenarioList, gin.H{
					"scenarioID":   scenario.Name(),
					"scenarioName": scenario.Name(),
				})
			}
		}

		c.JSON(http.StatusOK, scenarioList)
	}
}

func PutScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")

	// Validate the scenario folder
	scenarioPath, err := utils.ValidateFolderID(config.ScenarioFolder, scenarioID)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Parse the form to get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Check if the file is a .zip file
	if filepath.Ext(file.Filename) != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Clean the scenario folder
	if err := os.RemoveAll(scenarioPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	if err := os.MkdirAll(scenarioPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Save the file to the scenario folder
	filePath := filepath.Join(scenarioPath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scenario updated successfully"})
}

func DeleteScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")

	// Validate the scenario folder
	scenarioPath, err := utils.ValidateFolderID(config.ScenarioFolder, scenarioID)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Remove the scenario folder
	if err := os.RemoveAll(scenarioPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func PostScenario(c *gin.Context) {
	// Parse the form to get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Check if the file is a .zip file
	if filepath.Ext(file.Filename) != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Generate a unique 6-character ID
	uniqueID, err := utils.GenerateUniqueID(config.ScenarioFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Create the directory to save the file
	savePath := filepath.Join(config.ScenarioFolder, uniqueID)
	if err := os.MkdirAll(savePath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Save the file to the directory
	filePath := filepath.Join(savePath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully", "id": uniqueID})
}
