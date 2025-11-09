package utils

import (
	"dulus/server/config"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ProxmoxClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type ProxmoxAuthResponse struct {
	Data struct {
		CSRFPreventionToken string                 `json:"CSRFPreventionToken"`
		Ticket              string                 `json:"ticket"`
		Username            string                 `json:"username"`
		Cap                 map[string]interface{} `json:"cap"`
	} `json:"data"`
}

type ProxmoxClusterResourcesResponse struct {
	Data []ProxmoxResource `json:"data"`
}

type ProxmoxResource struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Node     string  `json:"node,omitempty"`
	VMID     int     `json:"vmid,omitempty"`
	Name     string  `json:"name,omitempty"`
	Pool     string  `json:"pool,omitempty"`
	Template int     `json:"template,omitempty"`
	Status   string  `json:"status,omitempty"`
	CPU      float64 `json:"cpu,omitempty"`
	MaxCPU   int     `json:"maxcpu,omitempty"`
	Mem      int64   `json:"mem,omitempty"`
	MaxMem   int64   `json:"maxmem,omitempty"`
	Disk     int64   `json:"disk,omitempty"`
	MaxDisk  int64   `json:"maxdisk,omitempty"`
	Uptime   int     `json:"uptime,omitempty"`
}

type ProxmoxStatistics struct {
	Users              int     `json:"users"`
	Templates          int     `json:"templates"`
	VMs                int     `json:"vms"`
	NumberOfTopologies int     `json:"numberOfTopologies"`
	NumberOfScenarios  int     `json:"numberOfScenarios"`
	NumberOfRoles      int     `json:"numberOfRoles"`
	NumberOfPools      int     `json:"numberOfPools"`
	CPUUsagePercentage float64 `json:"cpuUsagePercentage"`
	MaxCPU             int     `json:"maxCpu"`
	MemoryUsedGiB      float64 `json:"memoryUsedGiB"`
	MemoryTotalGiB     float64 `json:"memoryTotalGiB"`
	MemoryFreeGiB      float64 `json:"memoryFreeGiB"`
	DiskUsedGiB        float64 `json:"diskUsedGiB"`
	DiskTotalGiB       float64 `json:"diskTotalGiB"`
	UptimeFormatted    string  `json:"uptimeFormatted"`
	LudusVersion       string  `json:"ludusVersion"`
}

func NewProxmoxClient(baseURL string) *ProxmoxClient {
	return &ProxmoxClient{
		BaseURL:    baseURL,
		HTTPClient: createHTTPClient(),
	}
}

// AuthenticateProxmox authenticates with Proxmox and returns auth data
func (p *ProxmoxClient) AuthenticateProxmox(username, password string) (*ProxmoxAuthResponse, error) {
	authURL := fmt.Sprintf("%s/api2/json/access/ticket", p.BaseURL)

	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)

	req, err := http.NewRequest("POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth response: %w", err)
	}

	var authResp ProxmoxAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	return &authResp, nil
}

// GetClusterResources fetches cluster resources using authenticated session
func (p *ProxmoxClient) GetClusterResources(auth *ProxmoxAuthResponse) (*ProxmoxClusterResourcesResponse, error) {
	resourcesURL := fmt.Sprintf("%s/api2/json/cluster/resources", p.BaseURL)

	req, err := http.NewRequest("GET", resourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resources request: %w", err)
	}

	// Set authentication headers
	req.Header.Set("CSRFPreventionToken", auth.Data.CSRFPreventionToken)
	req.AddCookie(&http.Cookie{
		Name:  "PVEAuthCookie",
		Value: auth.Data.Ticket,
	})

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cluster resources request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read resources response: %w", err)
	}

	var resourcesResp ProxmoxClusterResourcesResponse
	if err := json.Unmarshal(body, &resourcesResp); err != nil {
		return nil, fmt.Errorf("failed to parse resources response: %w", err)
	}

	return &resourcesResp, nil
}

// ParseStatistics processes cluster resources and returns statistics
func ParseStatistics(resources *ProxmoxClusterResourcesResponse, apiKey string) *ProxmoxStatistics {
	stats := &ProxmoxStatistics{}

	var nodeResource *ProxmoxResource
	var totalDiskUsed int64
	var totalDiskMax int64

	for _, resource := range resources.Data {
		switch resource.Type {
		case "pool":
			// Exclude ADMIN and SHARED pools
			if resource.Pool != "ADMIN" && resource.Pool != "SHARED" {
				stats.Users++
			}
		case "qemu":
			if resource.Template == 1 {
				stats.Templates++
			} else {
				stats.VMs++
			}
		case "node":
			if resource.Node == config.ProxmoxNodeName {
				nodeResource = &resource
			}
		case "storage":
			if resource.Node == config.ProxmoxNodeName {
				totalDiskUsed += resource.Disk
				totalDiskMax += resource.MaxDisk
			}
		}
	}

	// Get additional statistics
	stats.NumberOfTopologies = countDirectories(config.TopologyConfigFolder)
	stats.NumberOfScenarios = countDirectories(config.CtfdScenarioFolder)
	stats.NumberOfPools = countDirectories(config.PoolFolder)
	stats.NumberOfRoles = getRolesCount(apiKey)

	// Get Ludus server version
	version, err := GetLudusServerVersion(apiKey)
	if err == nil {
		stats.LudusVersion = version
	} else {
		stats.LudusVersion = "Unknown"
	}

	// Process node statistics
	if nodeResource != nil {
		stats.CPUUsagePercentage = formatCPUUsagePercentage(nodeResource.CPU)
		stats.MaxCPU = nodeResource.MaxCPU
		stats.MemoryUsedGiB = bytesToGiB(nodeResource.Mem)
		stats.MemoryTotalGiB = bytesToGiB(nodeResource.MaxMem)
		stats.MemoryFreeGiB = bytesToGiB(nodeResource.MaxMem - nodeResource.Mem)
		stats.UptimeFormatted = formatUptime(nodeResource.Uptime)

		// Use storage statistics instead of node statistics for disk
		stats.DiskUsedGiB = bytesToGiB(totalDiskUsed)
		stats.DiskTotalGiB = bytesToGiB(totalDiskMax)
	}

	return stats
}

// formatCPUUsagePercentage formats CPU usage as a percentage with two decimal places
func formatCPUUsagePercentage(usage float64) float64 {
	usage = math.Round(usage*10000) / 100
	if usage > 100 {
		usage /= 10
	}
	return usage
}

// bytesToGiB converts bytes to GiB (Gibibytes)
func bytesToGiB(bytes int64) float64 {
	return math.Round(float64(bytes)/(1024*1024*1024)*100) / 100
}

// formatUptime converts seconds to human-readable format (weeks, days, hours)
func formatUptime(seconds int) string {
	duration := time.Duration(seconds) * time.Second

	weeks := int(duration.Hours()) / (24 * 7)
	days := (int(duration.Hours()) / 24) % 7
	hours := int(duration.Hours()) % 24

	var parts []string
	if weeks > 0 {
		parts = append(parts, fmt.Sprintf("%dw", weeks))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}

	if len(parts) == 0 {
		return "0h"
	}
	return strings.Join(parts, " ")
}

// countDirectories counts the number of directories in a given path
func countDirectories(dirPath string) int {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

// getRolesCount gets the number of roles from Ludus API
func getRolesCount(apiKey string) int {
	client := createHTTPClient()

	req, err := http.NewRequest("GET", config.LudusUrl+"/ansible", nil)
	if err != nil {
		return 0
	}

	req.Header.Set("X-Api-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}

	var roles []interface{}
	if err := json.Unmarshal(body, &roles); err != nil {
		return 0
	}

	return len(roles)
}
