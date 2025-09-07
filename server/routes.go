package main

import (
	"database/sql"
	"dulus/server/handlers"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
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
	userID, ok := utils.ExtractUserIDFromAPIKey(c, APIKey)
	if !ok {
		return
	}

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
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS, PATCH"},
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
	r.GET("/ctfd/data/logins", validateAPIKey, handlers.GetCtfdLogins)

	// Topology route
	r.GET("/topology", validateAPIKey, handlers.GetTopology)
	r.PUT("/topology", validateAPIKey, handlers.PutTopology)
	r.DELETE("/topology", validateAPIKey, handlers.DeleteTopology)

	// Pool route
	r.POST("/pool", validateAPIKey, handlers.PostPool)
	r.PATCH("/pool/topology", validateAPIKey, handlers.PatchPoolTopology)
	r.PATCH("/pool/note", validateAPIKey, handlers.PatchPoolNote)
	r.PATCH("/pool/users", validateAPIKey, handlers.PatchPoolUsers)
	r.POST("/pool/users", validateAPIKey, handlers.CheckUserIds)
	r.GET("/pool", validateAPIKey, handlers.GetPool)
	r.DELETE("/pool", validateAPIKey, handlers.DeletePool)

	// User management endpoints
	r.POST("/users/import", validateAPIKey, handlers.ImportUsers)
	r.POST("/users/delete", validateAPIKey, handlers.DeleteUsers)
	r.GET("/users/check", validateAPIKey, handlers.CheckUsers)

	// Range config
	r.POST("/range/config", validateAPIKey, handlers.SetRangeConfig)
	r.GET("/range/config", validateAPIKey, handlers.GetRangeConfig)

	// Range deployment
	r.POST("/range/deploy", validateAPIKey, handlers.DeployRange)
	r.GET("/range/status", validateAPIKey, handlers.CheckRangeStatus)
	r.POST("/range/redeploy", validateAPIKey, handlers.RedeployRange)
	r.POST("/range/abort", validateAPIKey, handlers.AbortRange)
	r.POST("/range/remove", validateAPIKey, handlers.RemoveRange)

	// Range sharing
	r.GET("/range/access", validateAPIKey, handlers.GetRangeAccess)
	r.POST("/range/share", validateAPIKey, handlers.ShareRange)
	r.POST("/range/unshare", validateAPIKey, handlers.UnshareRange)
	r.GET("/range/shared", validateAPIKey, handlers.GetSharedRanges)
}
