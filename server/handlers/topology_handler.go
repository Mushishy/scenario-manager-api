package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"

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
