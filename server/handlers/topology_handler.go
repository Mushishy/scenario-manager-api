package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetTopology(c *gin.Context) {
	topologyId := utils.GetOptionalQueryParam(c, "topologyId")

	if topologyId != "" {
		utils.GetSingleItemWithFile(c, config.TopologyConfigFolder, topologyId, "topology")
	} else {
		utils.GetAllItemsWithFileNames(c, config.TopologyConfigFolder, "topology")
	}
}

func PutTopology(c *gin.Context) {
	topologyId := utils.GetOptionalQueryParam(c, "topologyId")

	if topologyId == "ctfdev" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	id, ok := utils.SaveUploadedFile(c, config.TopologyConfigFolder, topologyId, ".yml")
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": id})
}

func DeleteTopology(c *gin.Context) {
	topologyId, ok := utils.GetRequiredQueryParam(c, "topologyId")
	if !ok {
		return
	}

	if topologyId == "ctfdev" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	topologyPath, ok := utils.ValidateFolderId(c, config.TopologyConfigFolder, topologyId)
	if !ok {
		return
	}

	// Check if topology is used in any pool
	poolDirs, err := os.ReadDir(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	for _, poolDir := range poolDirs {
		if poolDir.IsDir() {
			poolPath := filepath.Join(config.PoolFolder, poolDir.Name())
			pool, err := utils.ReadPoolInternal(poolPath)
			if err != nil {
				continue // Skip pools we can't read
			}

			if pool.TopologyId == topologyId {
				c.JSON(http.StatusConflict, gin.H{"error": "Conflict"})
				return
			}
		}
	}

	if err := os.RemoveAll(topologyPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func PostCtfdTopology(c *gin.Context) {
	input, ok := utils.ValidateJSONSchema(c, "file://schemas/ctfd_topology_schema.json")
	if !ok {
		return
	}

	inputBytes, _ := json.Marshal(input)
	var inputCtfdOptions utils.CtfdTopologyRequest
	if err := json.Unmarshal(inputBytes, &inputCtfdOptions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate date time formats and order
	if valid := utils.ValidateDateTimeRange(inputCtfdOptions.ConfStartTime, inputCtfdOptions.ConfStopTime); !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Read the template file using proper config
	templateContent, err := os.ReadFile(config.TemplateCtfdTopologyLocation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Validate topology exists
	_, ok = utils.ValidateFolderId(c, config.CtfdScenarioFolder, inputCtfdOptions.ScenarioID)
	if !ok {
		return
	}

	_, ok = utils.ValidateFolderId(c, config.PoolFolder, inputCtfdOptions.PoolID)
	if !ok {
		return
	}

	// Replace variables in template
	content := string(templateContent)
	replacements := map[string]string{
		"$SECENARIO_ID":            `"` + inputCtfdOptions.ScenarioID + `"`,
		"$POOL_ID":                 `"` + inputCtfdOptions.PoolID + `"`,
		"$ADMIN_USERNAME":          `"` + inputCtfdOptions.AdminUsername + `"`,
		"$ADMIN_PASSWORD":          `"` + inputCtfdOptions.AdminPassword + `"`,
		"$CTF_NAME":                `"` + inputCtfdOptions.CtfName + `"`,
		"$CTF_DESCRIPTION":         `"` + inputCtfdOptions.CtfDescription + `"`,
		"$CHALLENGE_VISIBILITY":    `"` + inputCtfdOptions.ChallengeVisibility + `"`,
		"$CHALLENGE_RATINGS":       `"` + inputCtfdOptions.ChallengeRatings + `"`,
		"$ACCOUNT_VISIBILITY":      `"` + inputCtfdOptions.AccountVisibility + `"`,
		"$SCORE_VISIBILITY":        `"` + inputCtfdOptions.ScoreVisibility + `"`,
		"$REGISTRATION_VISIBILITY": `"` + inputCtfdOptions.RegistrationVisibility + `"`,
		"$ALLOW_NAME_CHANGES":      `"` + inputCtfdOptions.AllowNameChanges + `"`,
		"$ALLOW_TEAM_CREATION":     `"` + inputCtfdOptions.AllowTeamCreation + `"`,
		"$ALLOW_TEAM_DISBANDING":   `"` + inputCtfdOptions.AllowTeamDisbanding + `"`,
		"$CONF_START_TIME":         `"` + inputCtfdOptions.ConfStartTime + `"`,
		"$CONF_STOP_TIME":          `"` + inputCtfdOptions.ConfStopTime + `"`,
		"$TIME_ZONE":               `"` + inputCtfdOptions.TimeZone + `"`,
		"$ALLOW_VIEWING_AFTER":     `"` + inputCtfdOptions.AllowViewingAfter + `"`,
	}

	for placeholder, value := range replacements {
		content = strings.ReplaceAll(content, placeholder, value)
	}

	// Generate unique topology ID
	topologyId, err := utils.GenerateUniqueID(config.TopologyConfigFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Create topology directory
	topologyPath := filepath.Join(config.TopologyConfigFolder, topologyId)
	if err := os.MkdirAll(topologyPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Save the generated topology file
	filename := "ctfd_" + inputCtfdOptions.TopologyName + ".yml"
	filePath := filepath.Join(topologyPath, filename)
	if err := os.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
		// Clean up directory on failure
		os.RemoveAll(topologyPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "CTFd topology created successfully",
		"topologyId":   topologyId,
		"topologyName": filename,
	})
}
