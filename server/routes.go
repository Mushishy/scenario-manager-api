package main

import (
	"database/sql"
	"dulus/server/handlers"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

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

	if CheckHash(APIKey, hashedAPIKey) {
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

func Index(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"result": "CTFd Extension for Ludus API"})
}

func RegisterRoutes(r *gin.Engine) {
	// Index route
	r.GET("/", validateAPIKey, Index)

	// Scenario route
	r.POST("/ctfd/scenario", validateAPIKey, handlers.PostScenario)
	r.GET("/ctfd/scenario", validateAPIKey, handlers.GetScenario)
	r.PUT("/ctfd/scenario", validateAPIKey, handlers.PutScenario)
	r.DELETE("/ctfd/scenario", validateAPIKey, handlers.DeleteScenario)

	// Data route
	r.POST("/ctfd/data", validateAPIKey, handlers.PostCtfdData)
	r.GET("/ctfd/data", validateAPIKey, handlers.GetCtfdData)
	r.PUT("/ctfd/data", validateAPIKey, handlers.PutCtfdData)
	r.DELETE("/ctfd/data", validateAPIKey, handlers.DeleteCtfdData)

	// Topology route
	r.POST("/topology", validateAPIKey, handlers.PostTopology)
	r.GET("/topology", validateAPIKey, handlers.GetTopology)
	r.PUT("/topology", validateAPIKey, handlers.PutTopology)
	r.DELETE("/topology", validateAPIKey, handlers.DeleteTopology)
}
