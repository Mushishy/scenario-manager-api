package main

import (
	"database/sql"
	"dulus/server/handlers"
	"dulus/server/utils"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func validateAPIKey(c *gin.Context) {
	APIKey := c.Request.Header.Get("X-API-Key")
	var isAdmin bool
	var hashedAPIKey string

	if len(APIKey) == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No API Key provided"})
		c.Abort()
		return
	}

	// Check that we can pull the userID and apikey from what the user provided
	apiKeySplit := strings.Split(APIKey, ".")
	if len(apiKeySplit) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Malformed API Key provided"})
		c.Abort()
		return
	}
	userID := apiKeySplit[0]

	err := db.QueryRow("SELECT is_admin, hashed_api_key FROM user_objects WHERE user_id = ?", userID).Scan(&isAdmin, &hashedAPIKey)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bad Request"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal errorrrrrrrr"})
		}
		c.Abort()
		return
	}

	if utils.CheckHash(APIKey, hashedAPIKey) {
		if isAdmin {
			c.Set("isAdmin", true)
		} else {
			c.Set("isAdmin", false)
		}
		c.Set("userID", userID)
		return
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		c.Abort()
		return
	}
}

func RegisterRoutes(r *gin.Engine) {
	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Index route
	r.GET("/", validateAPIKey, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"result": "Ludus Extension API"})
	})

	// Scenario route
	r.GET("/ctfd/scenario", validateAPIKey, handlers.GetScenario)
	r.PUT("/ctfd/scenario", validateAPIKey, handlers.PutScenario)
	r.DELETE("/ctfd/scenario", validateAPIKey, handlers.DeleteScenario)

	// Data route
	r.GET("/ctfd/data", validateAPIKey, handlers.GetCtfdData)
	r.PUT("/ctfd/data", validateAPIKey, handlers.PutCtfdData)
	r.DELETE("/ctfd/data", validateAPIKey, handlers.DeleteCtfdData)

	// Topology route
	r.GET("/topology", validateAPIKey, handlers.GetTopology)
	r.PUT("/topology", validateAPIKey, handlers.PutTopology)
	r.DELETE("/topology", validateAPIKey, handlers.DeleteTopology)
}
