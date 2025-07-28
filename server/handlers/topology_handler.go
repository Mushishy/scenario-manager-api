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

func GetTopology(c *gin.Context) {
	topologyId := c.Query("topologyId")

	if topologyId != "" {
		topologyPath, err := utils.ValidateFolderID(config.TopologyConfigFolder, topologyId)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}

		files, err := os.ReadDir(topologyPath)
		if err != nil || len(files) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		filePath := filepath.Join(topologyPath, files[0].Name())

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
			"topologyId":   topologyId,
			"topologyName": files[0].Name(),
			"topologyFile": encoded,
			"createdAt":    fileInfo.ModTime().Format(config.TimestampFormat),
		})
	} else {
		topologies, err := os.ReadDir(config.TopologyConfigFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		var topologyList []gin.H
		for _, topology := range topologies {
			if topology.IsDir() {
				topologyPath := filepath.Join(config.TopologyConfigFolder, topology.Name())
				files, err := os.ReadDir(topologyPath)
				if err != nil || len(files) == 0 {
					continue // Skip folders with no files or read errors
				}

				// Get file creation time
				filePath := filepath.Join(topologyPath, files[0].Name())
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					continue // Skip files we can't stat
				}

				topologyList = append(topologyList, gin.H{
					"topologyId":   topology.Name(),
					"topologyName": files[0].Name(),
					"createdAt":    fileInfo.ModTime().Format(config.TimestampFormat),
				})
			}
		}

		c.JSON(http.StatusOK, topologyList)
	}
}

func PutTopology(c *gin.Context) {
	topologyId := c.Query("topologyId")
	var topologyPath string
	var err error

	if topologyId != "" {
		topologyPath, err = utils.ValidateFolderID(config.TopologyConfigFolder, topologyId)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
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

	// Parse the form to get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Check if the file is a .yml file
	if filepath.Ext(file.Filename) != ".yml" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// If updating, clean the topology folder
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

	// Save the file to the topology folder
	filePath := filepath.Join(topologyPath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": topologyId})
}

func DeleteTopology(c *gin.Context) {
	topologyId := c.Query("topologyId")

	topologyPath, err := utils.ValidateFolderID(config.TopologyConfigFolder, topologyId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Remove the topology folder
	if err := os.RemoveAll(topologyPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}
