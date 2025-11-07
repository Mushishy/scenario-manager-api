package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioId")

	if scenarioID != "" {
		utils.GetSingleItemWithFile(c, config.CtfdScenarioFolder, scenarioID, "scenario")
	} else {
		utils.GetAllItemsWithFileNames(c, config.CtfdScenarioFolder, "scenario")
	}
}

func PutScenario(c *gin.Context) {
	scenarioID := c.Query("scenarioId")

	id, ok := utils.SaveUploadedFile(c, config.CtfdScenarioFolder, scenarioID, ".zip")
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": id})
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
