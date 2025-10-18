package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	TemplateCtfdTopologyLocation string
	CtfdScenarioFolder           string
	TopologyConfigFolder         string
	PoolFolder                   string
	DatabaseLocation             string
	TimestampFormat              string
	LudusAdminUrl                string
	LudusUrl                     string
	MaxConcurrentRequests        int
	ProxmoxURL                   string
	ProxmoxUsername              string
	ProxmoxPassword              string
	ProxmoxCertPath              string
)

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Println("Env file is not used")
	}
	loadVariables()
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	log.Fatalf("Environment variable %s is not set", key)
	return ""
}

func getEnvAsInt(key string) int {
	valueStr := getEnv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatalf("Environment variable %s must be an integer, but got: %s", key, valueStr)
	}
	return value
}

func loadVariables() {
	TemplateCtfdTopologyLocation = getEnv("TEMPLATE_CTFD_TOPOLOGY_LOCATION")
	CtfdScenarioFolder = getEnv("CTFD_SCENARIO_FOLDER")
	TopologyConfigFolder = getEnv("TOPOLOGY_CONFIG_FOLDER")
	PoolFolder = getEnv("POOL_FOLDER")
	DatabaseLocation = getEnv("DATABASE_LOCATION")
	TimestampFormat = getEnv("TIMESTAMP_FORMAT")
	LudusAdminUrl = getEnv("LUDUS_ADMIN_URL")
	LudusUrl = getEnv("LUDUS_URL")
	MaxConcurrentRequests = getEnvAsInt("MAX_CONCURRENT_REQUESTS")
	ProxmoxURL = getEnv("PROXMOX_URL")
	ProxmoxUsername = getEnv("PROXMOX_USERNAME")
	ProxmoxPassword = getEnv("PROXMOX_PASSWORD")
	ProxmoxCertPath = getEnv("PROXMOX_CERT_PATH")
}
