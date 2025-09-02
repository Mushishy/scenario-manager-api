package utils

import (
	"bytes"
	"crypto/tls"
	"dulus/server/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// Core types
type LudusRequest struct {
	Method  string
	URL     string
	Payload interface{}
	UserID  string
}

type LudusResponse struct {
	UserID   string
	Response interface{}
	Error    error
}

// Pool-related types
type UserDetails struct {
	Username string
	Team     string
}

type UserTeam struct {
	UserId string `json:"userId"`
	User   string `json:"user"`
	Team   string `json:"team"`
}

type Pool struct {
	CreatedBy     string `json:"createdBy"`
	Note          string `json:"note"`
	TopologyId    string `json:"topologyId"`
	Type          string `json:"type"`
	MainUser      string `json:"mainUser,omitempty"`
	UsersAndTeams []struct {
		User   string `json:"user"`
		UserId string `json:"userId"`
		Team   string `json:"team,omitempty"`
	} `json:"usersAndTeams"`
}

// CTFd-related types
type Flag struct {
	Variable string      `json:"variable"`
	Contents interface{} `json:"contents"`
}

type CtfdUser struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Team     string `json:"team"`
	Flags    []Flag `json:"flags"`
}

type CtfdData struct {
	CtfdData []CtfdUser `json:"ctfd_data"`
}

// Range status types
type RangeStatus struct {
	RangeState string `json:"rangeState"`
}

type LogResult struct {
	Result string `json:"result"`
}

// createHTTPClient creates HTTP client with TLS verification disabled
func createHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

// MakeConcurrentLudusRequests processes multiple Ludus requests concurrently
func MakeConcurrentLudusRequests(requests []LudusRequest, apiKey string, maxConcurrency int) []LudusResponse {
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default fallback
	}

	// Create channels
	requestChan := make(chan LudusRequest, len(requests))
	responseChan := make(chan LudusResponse, len(requests))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency && i < len(requests); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range requestChan {
				response, err := MakeLudusRequest(req.Method, req.URL, req.Payload, apiKey)
				responseChan <- LudusResponse{
					UserID:   req.UserID,
					Response: response,
					Error:    err,
				}
			}
		}()
	}

	// Send requests to workers
	for _, req := range requests {
		requestChan <- req
	}
	close(requestChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(responseChan)
	}()

	// Collect results
	results := make([]LudusResponse, 0, len(requests))
	for response := range responseChan {
		results = append(results, response)
	}

	return results
}

// MakeLudusRequest makes a single request to Ludus API
func MakeLudusRequest(method, url string, payload interface{}, apiKey string) (interface{}, error) {
	client := createHTTPClient()

	var req *http.Request
	var err error

	if payload != nil {
		jsonData, _ := json.Marshal(payload)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	}

	req.Header.Set("X-Api-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}

	return result, nil
}

// MakeConcurrentFileUploads processes multiple file uploads concurrently
func MakeConcurrentFileUploads(userIds []string, configContent string, force bool, apiKey string, maxConcurrency int) []LudusResponse {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}

	// Create channels
	userChan := make(chan string, len(userIds))
	responseChan := make(chan LudusResponse, len(userIds))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency && i < len(userIds); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range userChan {
				response, err := UploadConfigFile(userID, configContent, force, apiKey)
				responseChan <- LudusResponse{
					UserID:   userID,
					Response: response,
					Error:    err,
				}
			}
		}()
	}

	// Send user IDs to workers
	for _, userID := range userIds {
		userChan <- userID
	}
	close(userChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(responseChan)
	}()

	// Collect results
	results := make([]LudusResponse, 0, len(userIds))
	for response := range responseChan {
		results = append(results, response)
	}

	return results
}

// UploadConfigFile uploads configuration file to Ludus
func UploadConfigFile(userID, configContent string, force bool, apiKey string) (interface{}, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file content
	fileWriter, err := writer.CreateFormFile("file", "topology.yml")
	if err != nil {
		return nil, err
	}
	fileWriter.Write([]byte(configContent))

	// Add force parameter
	writer.WriteField("force", strconv.FormatBool(force))
	writer.Close()

	// Create request
	url := fmt.Sprintf("%s/range/config?userID=%s", config.LudusUrl, url.QueryEscape(userID))
	req, err := http.NewRequest("PUT", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Api-Key", apiKey)

	client := createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}

	return result, nil
}

// AllRangesDeployed checks if all user ranges are deployed
func AllRangesDeployed(userIds []string, apiKey string, c *gin.Context) bool {
	requests := make([]LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = LudusRequest{
			Method: "GET",
			URL:    config.LudusUrl + "/range/?userID=" + userID,
			UserID: userID,
		}
	}

	responses := MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	for _, resp := range responses {
		if resp.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to check range status for user " + resp.UserID})
			return false
		}

		if resp.Response != nil {
			var rangeStatus RangeStatus
			if data, err := json.Marshal(resp.Response); err == nil {
				if err := json.Unmarshal(data, &rangeStatus); err == nil {
					if rangeStatus.RangeState != "SUCCESS" && rangeStatus.RangeState != "DEPLOYED" {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Not all ranges are deployed. User " + resp.UserID + " has state: " + rangeStatus.RangeState})
						return false
					}
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to determine range state for user " + resp.UserID})
					return false
				}
			}
		}
	}
	return true
}

// GetUserDetailsFromPool reads pool.json and returns a map of user details
func GetUserDetailsFromPool(poolPath string) (map[string]UserDetails, error) {
	poolJsonPath := filepath.Join(poolPath, "pool.json")
	poolData, err := os.ReadFile(poolJsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool data")
	}

	var pool Pool
	if err := json.Unmarshal(poolData, &pool); err != nil {
		return nil, fmt.Errorf("failed to parse pool data")
	}

	userDetailMap := make(map[string]UserDetails)
	for _, userTeam := range pool.UsersAndTeams {
		userDetailMap[userTeam.UserId] = UserDetails{
			Username: userTeam.User,
			Team:     userTeam.Team,
		}
	}
	return userDetailMap, nil
}

// ExtractFlagsFromLogs gets logs and extracts flags for all users
func ExtractFlagsFromLogs(userIds []string, userDetailMap map[string]UserDetails, apiKey string) []CtfdUser {
	requests := make([]LudusRequest, len(userIds))
	for i, userID := range userIds {
		requests[i] = LudusRequest{
			Method: "GET",
			URL:    config.LudusUrl + "/range/logs/?userID=" + userID,
			UserID: userID,
		}
	}

	responses := MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)
	flagPattern := regexp.MustCompile(`&%&(.*?)&%&`)
	var ctfdUsers []CtfdUser

	for _, resp := range responses {
		userDetails, exists := userDetailMap[resp.UserID]
		if !exists {
			continue // Skip users not found in pool data
		}

		ctfdUser := CtfdUser{
			User:     userDetails.Username,
			Password: RandomString(6),
			Team:     userDetails.Team,
			Flags:    ExtractUserFlags(resp, flagPattern),
		}

		ctfdUsers = append(ctfdUsers, ctfdUser)
	}
	return ctfdUsers
}

// ExtractUserFlags extracts flags from a single user's log response
func ExtractUserFlags(resp LudusResponse, flagPattern *regexp.Regexp) []Flag {
	var flags []Flag

	if resp.Error != nil || resp.Response == nil {
		return flags
	}

	var logResult LogResult
	if data, err := json.Marshal(resp.Response); err == nil {
		if err := json.Unmarshal(data, &logResult); err != nil {
			return flags
		}
	} else {
		return flags
	}

	match := flagPattern.FindStringSubmatch(logResult.Result)
	if len(match) <= 1 {
		return flags
	}

	content := strings.ReplaceAll(match[1], "\\", "")

	var jsonContent map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonContent); err != nil {
		return flags
	}

	for key, value := range jsonContent {
		flags = append(flags, Flag{
			Variable: key,
			Contents: value,
		})
	}
	return flags
}
