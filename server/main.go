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
	//cgin.SetMode(gin.ReleaseMode)
	r.SetTrustedProxies(nil)

	// Register routes
	RegisterRoutes(r)

	// Ensure base directories exist
	utils.EnsureDirectoryExists(config.ScenarioFolder)
	utils.EnsureDirectoryExists(config.CtfdDataFolder)

	// Start the server
	r.Run(":8080")
}
