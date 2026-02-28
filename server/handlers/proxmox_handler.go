package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetProxmoxStatistics is the Gin handler for Proxmox statistics endpoint
func GetProxmoxStatistics(c *gin.Context) {
	// Create Proxmox client
	client := utils.NewProxmoxClient(config.ProxmoxURL)

	apiKey := c.Request.Header.Get("X-API-Key")

	response, err := utils.MakeLudusRequest("GET", config.LudusUrl+"/user/credentials", nil, apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Extract credentials from the response (already parsed as map[string]interface{})
	credResp := response.(map[string]interface{})
	result := credResp["result"].(map[string]interface{})
	proxmoxUsername := result["proxmoxUsername"].(string) + "@pam"
	proxmoxPassword := result["proxmoxPassword"].(string)


	// AuthenticateProxmox
	auth, err := client.AuthenticateProxmox(proxmoxUsername, proxmoxPassword)
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

	// Parse statistics
	stats := utils.ParseStatistics(resources, apiKey)

	// Return statistics
	c.JSON(http.StatusOK, stats)
}
