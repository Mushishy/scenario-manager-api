package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func GetScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")

	if scenarioID != "" {
		scenarioPath, err := utils.ValidateFolderID(config.CtfdScenarioFolder, scenarioID)
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

		filePath := filepath.Join(scenarioPath, files[0].Name())

		// Get file creation time
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		file, err := os.Open(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		encoded := base64.StdEncoding.EncodeToString(fileBytes)

		c.JSON(http.StatusOK, gin.H{
			"scenarioID":   scenarioID,
			"scenarioName": files[0].Name(),
			"scenarioFile": encoded,
			"createdAt":    fileInfo.ModTime().Format(config.TimestampFormat),
		})
	} else {
		scenarios, err := os.ReadDir(config.CtfdScenarioFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		var scenarioList []gin.H
		for _, scenario := range scenarios {
			if scenario.IsDir() {
				scenarioPath := filepath.Join(config.CtfdScenarioFolder, scenario.Name())
				files, err := os.ReadDir(scenarioPath)
				if err != nil || len(files) == 0 {
					continue // Skip folders with no files or read errors
				}

				// Get file creation time
				filePath := filepath.Join(scenarioPath, files[0].Name())
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					continue // Skip files we can't stat
				}

				scenarioList = append(scenarioList, gin.H{
					"scenarioID":   scenario.Name(),
					"scenarioName": files[0].Name(),
					"createdAt":    fileInfo.ModTime().Format(config.TimestampFormat),
				})
			}
		}

		c.JSON(http.StatusOK, scenarioList)
	}
}

func PutScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")
	var scenarioPath string
	var err error

	if scenarioID != "" {
		scenarioPath, err = utils.ValidateFolderID(config.CtfdScenarioFolder, scenarioID)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
	} else {
		scenarioID, err = utils.GenerateUniqueID(config.CtfdScenarioFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		scenarioPath = filepath.Join(config.CtfdScenarioFolder, scenarioID)
		if err := os.MkdirAll(scenarioPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
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

	// If updating, clean the scenario folder
	if scenarioID != "" {
		if err := os.RemoveAll(scenarioPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		if err := os.MkdirAll(scenarioPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
	}

	// Save the file to the scenario folder
	filePath := filepath.Join(scenarioPath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": scenarioID})
}

func DeleteScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioID")

	scenarioPath, err := utils.ValidateFolderID(config.CtfdScenarioFolder, scenarioID)
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
