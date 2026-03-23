package utils

import (
	"bytes"
	"dulus/server/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

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
	UsersAndTeams []struct {
		User       string `json:"user"`
		UserId     string `json:"userId"`
		Team       string `json:"team,omitempty"`
		MainUserId string `json:"mainUserId,omitempty"`
	} `json:"usersAndTeams"`
}

// Range status types
type RangeStatus struct {
	RangeState string `json:"rangeState"`
}

type LogResult struct {
	Result string `json:"result"`
}

// MakeConcurrentLudusRequests processes multiple Ludus requests concurrently with optional sleep between requests
func MakeConcurrentLudusRequests(requests []LudusRequest, apiKey string, maxConcurrency int) []LudusResponse {
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

// MakeSequentialRequestsWithSleep processes multiple Ludus requests sequentially with optional sleep between requests
func MakeSequentialRequestsWithSleep(requests []LudusRequest, apiKey string, sleepDuration time.Duration) []LudusResponse {
	results := make([]LudusResponse, 0, len(requests))

	for i, req := range requests {
		response, err := MakeLudusRequest(req.Method, req.URL, req.Payload, apiKey)
		results = append(results, LudusResponse{
			UserID:   req.UserID,
			Response: response,
			Error:    err,
		})

		// Sleep between requests if configured and not the last request
		if sleepDuration > 0 && i < len(requests)-1 {
			time.Sleep(sleepDuration)
		}
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

// WaitForBatchDestroyed waits until all users in batch are destroyed
func WaitForBatchDestroyed(userIds []string, apiKey string, checkInterval time.Duration) {
	for {
		allDestroyed := true

		// Check status of all users in batch
		requests := make([]LudusRequest, len(userIds))
		for i, userID := range userIds {
			requests[i] = LudusRequest{
				Method: "GET",
				URL:    config.LudusUrl + "/range/?userID=" + userID,
				UserID: userID,
			}
		}

		responses := MakeConcurrentLudusRequests(requests, apiKey, len(userIds))

		for _, resp := range responses {
			if resp.Error != nil {
				continue // Error might mean destroyed or not found
			}

			state := "unknown"
			if resp.Response != nil {
				// Handle both string and map responses
				switch response := resp.Response.(type) {
				case map[string]interface{}:
					if rangeState, exists := response["rangeState"]; exists {
						state = rangeState.(string)
					}
				case string:
					state = response
				}
			}

			// If any user is still destroying, we're not done
			if state == "DESTROYING" {
				allDestroyed = false
				break
			}
		}

		// If all destroyed, continue
		if allDestroyed {
			return
		}

		time.Sleep(checkInterval)
	}
}

// WaitForBatchDeployment waits until all users in batch are deployed or failed
func WaitForBatchDeployment(userIds []string, apiKey string, checkInterval time.Duration) {
	for {
		allDone := true
		// Check status of all users in batch
		requests := make([]LudusRequest, len(userIds))
		for i, userID := range userIds {
			requests[i] = LudusRequest{
				Method: "GET",
				URL:    config.LudusUrl + "/range/?userID=" + userID,
				UserID: userID,
			}
		}

		responses := MakeConcurrentLudusRequests(requests, apiKey, len(userIds))

		for _, resp := range responses {
			if resp.Error != nil {
				continue // Error = done (failed)
			}

			state := "unknown"
			if resp.Response != nil {
				if rangeState, exists := resp.Response.(map[string]interface{})["rangeState"]; exists {
					state = rangeState.(string)
				}
			}

			// If any user is still deploying, we're not done
			if state == "DEPLOYING" {
				allDone = false
			}
		}

		// If all done, continue to next batch
		if allDone {
			return
		}

		time.Sleep(checkInterval)
	}
}

// GetFlagsForUsers retrieves flags for multiple users concurrently
func GetFlagsForUsers(c *gin.Context, userIds []string, apiKey string) (map[string][]Flag, bool) {
	requests := make([]LudusRequest, len(userIds))
	for i, userId := range userIds {
		requests[i] = LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/range/logs/?userID=" + userId,
			Payload: nil,
			UserID:  userId,
		}
	}

	responses := MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Process responses and create a map of userId -> flags
	userFlagsMap := make(map[string][]Flag)
	flagPattern := regexp.MustCompile(`&%&&%&&%&(.*?)&%&&%&&%&`)
	for _, resp := range responses {
		if resp.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get flags for user: " + resp.UserID})
			return nil, false
		}

		// Extract flags for this user
		flags := ExtractUserFlags(resp, flagPattern)
		userFlagsMap[resp.UserID] = flags
	}

	return userFlagsMap, true
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

	matches := flagPattern.FindAllStringSubmatch(logResult.Result, -1)
	if len(matches) == 0 {
		return flags
	}

	for _, match := range matches {
		if len(match) <= 1 {
			continue
		}

		content := strings.ReplaceAll(match[1], "\\", "")

		var jsonContent map[string]interface{}
		if err := json.Unmarshal([]byte(content), &jsonContent); err != nil {
			continue
		}

		for key, value := range jsonContent {
			flags = append(flags, Flag{
				Variable: key,
				Contents: value,
			})
		}
	}
	return flags
}

// GetLudusServerVersion retrieves the Ludus server version and trims the commit hash
func GetLudusServerVersion(apiKey string) (string, error) {
	response, err := MakeLudusRequest("GET", config.LudusUrl+"/", nil, apiKey)
	if err != nil {
		return "", err
	}

	// Parse response to extract version
	if resultMap, ok := response.(map[string]interface{}); ok {
		if result, exists := resultMap["result"]; exists {
			if versionStr, ok := result.(string); ok {
				// Trim the part after + (e.g., "Ludus Server v1.0.0+abc123a" -> "Ludus Server v1.0.0")
				if plusIndex := strings.Index(versionStr, "+"); plusIndex != -1 {
					return versionStr[:plusIndex], nil
				}
				return versionStr, nil
			}
		}
	}

	return "", fmt.Errorf("invalid response format")
}
