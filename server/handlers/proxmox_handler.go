package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProxmoxStatistics is the Gin handler for Proxmox statistics endpoint
func GetProxmoxStatistics(c *gin.Context) {
	proxmoxURL := config.ProxmoxURL
	username := config.ProxmoxUsername
	password := config.ProxmoxPassword

	if proxmoxURL == "" || username == "" || password == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Server Error",
		})
		return
	}

	// Create Proxmox client
	client := utils.NewProxmoxClient(proxmoxURL)

	// Authenticate
	auth, err := client.Authenticate(username, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bad Request",
		})
		return
	}

	// Get cluster resources
	resources, err := client.GetClusterResources(auth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Server Error",
		})
		return
	}

	// Get API key for additional statistics
	apiKey := c.Request.Header.Get("X-API-Key")

	// Parse statistics
	stats := utils.ParseStatistics(resources, apiKey)

	// Return statistics
	c.JSON(http.StatusOK, stats)
}
