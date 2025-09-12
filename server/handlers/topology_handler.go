package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/base64"
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
		topologyPath, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId)
		if !ok {
			return
		}

		fileInfo, err := utils.ReadFirstFileInDir(topologyPath)
		if utils.HandleFileReadError(c, err) {
			return
		}

		encoded := base64.StdEncoding.EncodeToString([]byte(fileInfo.Content))

		c.JSON(http.StatusOK, gin.H{
			"topologyId":   topologyId,
			"topologyName": fileInfo.Name,
			"topologyFile": encoded,
			"createdAt":    fileInfo.CreationTime.Format(config.TimestampFormat),
		})
		return
	}

	// Return all topologies
	topologies, err := utils.GetAllItems(config.TopologyConfigFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	var topologyList []gin.H
	for _, topology := range topologies {
		// Read the first file to get the topology name
		topologyPath := filepath.Join(config.TopologyConfigFolder, topology.ID)
		var topologyName string

		fileInfo, err := utils.ReadFirstFileInDir(topologyPath)
		if err == nil {
			topologyName = fileInfo.Name
		}

		topologyList = append(topologyList, gin.H{
			"topologyId":   topology.ID,
			"topologyName": topologyName, // This should now have the actual filename
			"createdAt":    topology.CreationTime.Format(config.TimestampFormat),
		})
	}

	c.JSON(http.StatusOK, topologyList)
}

func PutTopology(c *gin.Context) {
	topologyId := utils.GetOptionalQueryParam(c, "topologyId")
	var topologyPath string
	var err error

	if topologyId != "" {
		// Validate existing folder and set topologyPath
		validatedPath, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId)
		if !ok {
			return
		}
		topologyPath = validatedPath // Set the topologyPath for existing topology
	} else {
		topologyId, err = utils.GenerateUniqueID(config.TopologyConfigFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		topologyPath = filepath.Join(config.TopologyConfigFolder, topologyId)
		if err := os.MkdirAll(topologyPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
	}

	// Handle file upload
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Check file extension
	if filepath.Ext(file.Filename) != ".yml" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Clean and recreate folder if updating
	if topologyId != "" {
		if err := os.RemoveAll(topologyPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
		if err := os.MkdirAll(topologyPath, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}
	}

	// Save file
	filePath := filepath.Join(topologyPath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": topologyId})
}

func DeleteTopology(c *gin.Context) {
	topologyId, ok := utils.GetRequiredQueryParam(c, "topologyId")
	if !ok {
		return
	}

	topologyPath, ok := utils.ValidateFolderWithResponse(c, config.TopologyConfigFolder, topologyId)
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
			poolData, err := utils.ReadPoolData(poolPath)
			if err != nil {
				continue // Skip pools we can't read
			}

			if poolTopologyId, exists := poolData["topologyId"]; exists {
				if poolTopologyId == topologyId {
					c.JSON(http.StatusConflict, gin.H{"error": "Topology is in use by pool " + poolDir.Name()})
					return
				}
			}
		}
	}

	if err := os.RemoveAll(topologyPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// PostCtfdTopology creates a new CTFd topology based on template with variable substitution
func PostCtfdTopology(c *gin.Context) {
	// Validate JSON request body using schema (defaults are applied by JSON Schema)
	input, ok := utils.ValidateJSONSchema(c, "file://schemas/ctfd_topology_schema.json")
	if !ok {
		return
	}

	// Convert validated input to struct using JSON marshaling/unmarshaling
	inputBytes, _ := json.Marshal(input)
	var req utils.CtfdTopologyRequest
	if err := json.Unmarshal(inputBytes, &req); err != nil {
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
	_, ok = utils.ValidateFolderWithResponse(c, config.CtfdScenarioFolder, req.ScenarioID)
	if !ok {
		return
	}

	_, ok = utils.ValidateFolderWithResponse(c, config.PoolFolder, req.PoolID)
	if !ok {
		return
	}

	// Replace variables in template
	content := string(templateContent)
	replacements := map[string]string{
		"$SECENARIO_ID":            `"` + req.ScenarioID + `"`,
		"$POOL_ID":                 `"` + req.PoolID + `"`,
		"$USERNAME_CONFIG":         `"` + req.UsernameConfig + `"`,
		"$PASSWORD_CONFIG":         `"` + req.PasswordConfig + `"`,
		"$ADMIN_USERNAME":          `"` + req.AdminUsername + `"`,
		"$ADMIN_PASSWORD":          `"` + req.AdminPassword + `"`,
		"$CTF_NAME":                `"` + req.CtfName + `"`,
		"$CTF_DESCRIPTION":         `"` + req.CtfDescription + `"`,
		"$CHALLENGE_VISIBILITY":    `"` + req.ChallengeVisibility + `"`,
		"$ACCOUNT_VISIBILITY":      `"` + req.AccountVisibility + `"`,
		"$SCORE_VISIBILITY":        `"` + req.ScoreVisibility + `"`,
		"$REGISTRATION_VISIBILITY": `"` + req.RegistrationVisibility + `"`,
		"$ALLOW_NAME_CHANGES":      `"` + req.AllowNameChanges + `"`,
		"$ALLOW_TEAM_CREATION":     `"` + req.AllowTeamCreation + `"`,
		"$ALLOW_TEAM_DISBANDING":   `"` + req.AllowTeamDisbanding + `"`,
		"$CONF_START_TIME":         `"` + req.ConfStartTime + `"`,
		"$CONF_STOP_TIME":          `"` + req.ConfStopTime + `"`,
		"$TIME_ZONE":               `"` + req.TimeZone + `"`,
		"$ALLOW_VIEWING_AFTER":     `"` + req.AllowViewingAfter + `"`,
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
	filename := "ctfd_" + req.TopologyName + ".yml"
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
