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
	scenarioID := c.Query("scenarioId")

	if scenarioID != "" {
		// Get single scenario with mode detection
		utils.GetSingleScenarioWithMode(c, scenarioID)
	} else {
		// Get all scenarios with mode detection
		utils.GetAllScenariosWithMode(c)
	}
}

func PutScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioId")

	// First save the file using existing utility function
	id, ok := utils.SaveUploadedFile(c, config.CtfdScenarioFolder, scenarioID, ".zip")
	if !ok {
		return
	}

	// Find the uploaded zip file path
	scenarioPath := filepath.Join(config.CtfdScenarioFolder, id)
	fileInfo, err := utils.ReadFirstFileInDir(scenarioPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	zipPath := filepath.Join(scenarioPath, fileInfo.Name)

	// Validate the CTFd scenario zip file and get scenario mode
	scenarioMode, err := utils.GetScenarioModeFromZip(zipPath)
	if err != nil {
		// If validation fails, clean up the uploaded file
		os.RemoveAll(scenarioPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Uploaded successfully",
		"id":           id,
		"scenarioMode": scenarioMode,
	})
}

func DeleteScenario(c *gin.Context) {
	scenarioID, ok := utils.GetRequiredQueryParam(c, "scenarioId")
	if !ok {
		return
	}

	scenarioPath, ok := utils.ValidateFolderId(c, config.CtfdScenarioFolder, scenarioID)
	if !ok {
		return
	}

	if err := os.RemoveAll(scenarioPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}
