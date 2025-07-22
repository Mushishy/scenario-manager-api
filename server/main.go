package main

import (
	"database/sql"
	"dulus/server/config"
	"dulus/server/utils"
	"log"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", config.DatabaseLocation)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	r := gin.Default()
	// gin.SetMode(gin.ReleaseMode)
	r.SetTrustedProxies(nil)

	// Register routes
	RegisterRoutes(r)

	// Ensure base directories exist
	utils.EnsureDirectoryExists(config.ScenarioFolder)
	utils.EnsureDirectoryExists(config.CtfdDataFolder)
	utils.EnsureDirectoryExists(config.TopologyConfigFolder)

	// Start the server
	r.Run("0.0.0.0:5000")
}
