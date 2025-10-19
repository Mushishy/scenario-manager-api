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
	MaxConcurrentRequests = getEnvAsInt("MAX_CONCURRENT_REQUESTS")
	DataLocation := getEnv("DATA_LOCATION")

	LudusAdminUrl = getEnv("LUDUS_ADMIN_URL")
	LudusUrl = getEnv("LUDUS_URL")
	ProxmoxURL = getEnv("PROXMOX_URL")

	ProxmoxUsername = getEnv("PROXMOX_USERNAME")
	ProxmoxPassword = getEnv("PROXMOX_PASSWORD")
	ProxmoxCertPath = getEnv("PROXMOX_CERT_PATH")

	DatabaseLocation = DataLocation + "/input/dulus.db"
	TemplateCtfdTopologyLocation = DataLocation + "ctfd_topology.yml"
	CtfdScenarioFolder = DataLocation + "scenarios/"
	TopologyConfigFolder = DataLocation + "topologies/"
	PoolFolder = DataLocation + "pools/"
	TimestampFormat = "2006-01-02T15:04:05Z07:00"
}
