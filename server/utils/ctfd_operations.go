package utils

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"dulus/server/config"

	"github.com/gin-gonic/gin"
)

// CtfdTopologyRequest represents the request structure for CTFd topology creation
type CtfdTopologyRequest struct {
	TopologyName           string `json:"topologyName"`
	ScenarioID             string `json:"scenarioId"`
	PoolID                 string `json:"poolId"`
	AdminUsername          string `json:"adminUsername"`
	AdminPassword          string `json:"adminPassword"`
	CtfName                string `json:"ctfName"`
	CtfDescription         string `json:"ctfDescription"`
	ChallengeVisibility    string `json:"challengeVisibility"`
	ChallengeRatings       string `json:"challengeRatings"`
	AccountVisibility      string `json:"accountVisibility"`
	ScoreVisibility        string `json:"scoreVisibility"`
	RegistrationVisibility string `json:"registrationVisibility"`
	AllowNameChanges       string `json:"allowNameChanges"`
	AllowTeamCreation      string `json:"allowTeamCreation"`
	AllowTeamDisbanding    string `json:"allowTeamDisbanding"`
	ConfStartTime          string `json:"confStartTime"`
	ConfStopTime           string `json:"confStopTime"`
	TimeZone               string `json:"timeZone"`
	AllowViewingAfter      string `json:"allowViewingAfter"`
}

// CTFd-related types
type Flag struct {
	Variable string      `json:"variable"`
	Contents interface{} `json:"contents"`
}

type CtfdUser struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Team     string `json:"team,omitempty"`
	Flags    []Flag `json:"flags"`
}

type CtfdData struct {
	CtfdData []CtfdUser `json:"ctfd_data"`
}

// ReadCTFdJSON reads and parses CTFd JSON data
func ReadCTFdJSON(c *gin.Context, dataPath string) (CtfdData, bool) {
	filePath := filepath.Join(dataPath, "ctfd_data.json")
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return CtfdData{}, false
	}
	defer file.Close()

	var data CtfdData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return CtfdData{}, false
	}

	return data, true
}

// SaveCTFdData saves CTFd data to the specified path and handles HTTP responses
func SaveCTFdData(c *gin.Context, dataPath string, ctfdUsers []CtfdUser) bool {
	filePath := filepath.Join(dataPath, "ctfd_data.json")

	// Check if file already exists, if so, do nothing
	if _, err := os.Stat(filePath); err == nil {
		return true // File exists, silently return success
	}

	file, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return false
	}
	defer file.Close()

	// Create the data structure that matches the expected format
	data := map[string]interface{}{"ctfd_data": ctfdUsers}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return false
	}

	return true
}

// DeleteCtfdData deletes the ctfd_data.json file from a pool directory
func DeleteCtfdData(poolId string) error {
	ctfdDataPath := filepath.Join(config.PoolFolder, poolId, "ctfd_data.json")

	err := os.Remove(ctfdDataPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Check if pool has CTFd data
func HasCtfdData(poolPath string) bool {
	ctfdDataPath := filepath.Join(poolPath, "ctfd_data.json")
	_, err := os.Stat(ctfdDataPath)
	return err == nil
}

// GetScenarioModeFromZip extracts and validates a CTFd scenario zip file and returns the user mode
func GetScenarioModeFromZip(zipPath string) (string, error) {
	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("Bad Request")
	}
	defer r.Close()

	// Look for db/config.json in the zip
	var configFile *zip.File
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "db/config.json") {
			configFile = f
			break
		}
	}

	if configFile == nil {
		return "", fmt.Errorf("Bad Request")
	}

	// Read config.json
	rc, err := configFile.Open()
	if err != nil {
		return "", fmt.Errorf("Internal Server Error")
	}
	defer rc.Close()

	// Read the file content
	configBytes, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("Internal Server Error")
	}

	// Parse JSON as generic map
	var configData map[string]interface{}
	if err := json.Unmarshal(configBytes, &configData); err != nil {
		return "", fmt.Errorf("Bad Request")
	}

	// Get the results array
	results, ok := configData["results"].([]interface{})
	if !ok {
		return "", fmt.Errorf("Bad Request")
	}

	// Find user_mode setting
	for _, result := range results {
		entry, ok := result.(map[string]interface{})
		if !ok {
			continue
		}
		
		key, keyExists := entry["key"].(string)
		value, valueExists := entry["value"].(string)
		
		if keyExists && valueExists && key == "user_mode" {
			userMode := strings.ToLower(value)
			if userMode != "teams" && userMode != "users" {
				return "", fmt.Errorf("Bad Request")
			}
			return strings.ToUpper(userMode), nil
		}
	}

	return "", fmt.Errorf("Bad Request")
}

// GetSingleScenarioWithMode gets a single scenario and includes its mode
func GetSingleScenarioWithMode(c *gin.Context, scenarioID string) {
	scenarioPath, ok := ValidateFolderId(c, config.CtfdScenarioFolder, scenarioID)
	if !ok {
		return
	}

	fileInfo, err := ReadFirstFileInDir(scenarioPath)
	if HandleFileReadError(c, err) {
		return
	}

	// Try to get scenario mode if it's a zip file
	var scenarioMode *string
	if strings.HasSuffix(fileInfo.Name, ".zip") {
		zipPath := filepath.Join(scenarioPath, fileInfo.Name)
		mode, err := GetScenarioModeFromZip(zipPath)
		if err == nil {
			scenarioMode = &mode
		}
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(fileInfo.Content))

	response := gin.H{
		"scenarioId":   scenarioID,
		"scenarioName": fileInfo.Name,
		"scenarioFile": encoded,
		"createdAt":    fileInfo.CreationTime.Format(config.TimestampFormat),
	}

	if scenarioMode != nil {
		response["scenarioMode"] = *scenarioMode
	}

	c.JSON(http.StatusOK, response)
}

// GetAllScenariosWithMode gets all scenarios and includes their modes
func GetAllScenariosWithMode(c *gin.Context) {
	items, err := os.ReadDir(config.CtfdScenarioFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	var scenarioList []gin.H
	for _, item := range items {
		if item.IsDir() {
			scenarioPath := filepath.Join(config.CtfdScenarioFolder, item.Name())

			// Try to read first file for name and creation time
			fileInfo, err := ReadFirstFileInDir(scenarioPath)
			var fileName string
			var createdAt string
			var scenarioMode *string

			if err == nil {
				fileName = fileInfo.Name
				createdAt = fileInfo.CreationTime.Format(config.TimestampFormat)

				// Try to get scenario mode if it's a zip file
				if strings.HasSuffix(fileName, ".zip") {
					zipPath := filepath.Join(scenarioPath, fileName)
					mode, err := GetScenarioModeFromZip(zipPath)
					if err == nil {
						scenarioMode = &mode
					}
				}
			} else {
				// Fallback to directory modification time
				dirInfo, dirErr := os.Stat(scenarioPath)
				if dirErr == nil {
					createdAt = dirInfo.ModTime().Format(config.TimestampFormat)
				}
			}

			scenarioItem := gin.H{
				"scenarioId":   item.Name(),
				"scenarioName": fileName,
				"createdAt":    createdAt,
			}

			if scenarioMode != nil {
				scenarioItem["scenarioMode"] = *scenarioMode
			}

			scenarioList = append(scenarioList, scenarioItem)
		}
	}

	c.JSON(http.StatusOK, scenarioList)
}
