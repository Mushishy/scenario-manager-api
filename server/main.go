package main

import (
	"database/sql"
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", config.DatabaseLocation)
	if err != nil {
		log.Fatal(err)
	}
}

func initSSL() (string, string) {
	certPath := "/etc/pve/nodes/" + config.ProxmoxNode + "/pve-ssl.pem"
	keyPath := "/etc/pve/nodes/" + config.ProxmoxNode + "/pve-ssl.key"

	if utils.FileExists(certPath) && utils.FileExists(keyPath) {
		return certPath, keyPath
	}

	// Fallback to Ludus certificates
	certPath = "/opt/ludus/cert.pem"
	keyPath = "/opt/ludus/key.pem"

	if utils.FileExists(certPath) && utils.FileExists(keyPath) {
		return certPath, keyPath
	}

	// No valid certificates found
	return "", ""
}

func main() {
	initDB()
	defer db.Close()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	// Register routes
	RegisterRoutes(r)

	// Ensure base directories exist
	utils.EnsureDirectoryExists(config.CtfdScenarioFolder)
	utils.EnsureDirectoryExists(config.TopologyConfigFolder)
	utils.EnsureDirectoryExists(config.PoolFolder)

	// Initialize SSL certificates
	certPath, keyPath := initSSL()

	if certPath == "" || keyPath == "" {
		// Start the server without TLS
		fmt.Println("Starting http server")
		r.Run("127.0.0.1:5000")
	} else {
		// Start the server with TLS
		fmt.Println("Starting https server")
		gin.SetMode(gin.ReleaseMode)
		r.RunTLS("127.0.0.1:5000", certPath, keyPath)
	}
}
