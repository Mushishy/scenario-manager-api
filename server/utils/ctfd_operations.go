package utils

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"dulus/server/config"

	"github.com/gin-gonic/gin"
)

// CtfdTopologyRequest represents the request structure for CTFd topology creation
type CtfdTopologyRequest struct {
	TopologyName           string `json:"topologyName"`
	ScenarioID             string `json:"scenarioId"`
	PoolID                 string `json:"poolId"`
	UsernameConfig         string `json:"usernameConfig"`
	PasswordConfig         string `json:"passwordConfig"`
	AdminUsername          string `json:"adminUsername"`
	AdminPassword          string `json:"adminPassword"`
	CtfName                string `json:"ctfName"`
	CtfDescription         string `json:"ctfDescription"`
	ChallengeVisibility    string `json:"challengeVisibility"`
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
