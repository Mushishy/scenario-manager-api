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
	"strconv"
	"sync"
)

type LudusRequest struct {
	Method  string
	URL     string
	Payload interface{}
	UserID  string // For identification in results
}

type LudusResponse struct {
	UserID   string
	Response interface{}
	Error    error
}

// Create HTTP client with TLS verification disabled
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

// Helper function to create a map of results by UserID for easier lookup
func MapResponsesByUserID(responses []LudusResponse) map[string]LudusResponse {
	responseMap := make(map[string]LudusResponse)
	for _, resp := range responses {
		responseMap[resp.UserID] = resp
	}
	return responseMap
}

func MakeLudusRequest(method, url string, payload interface{}, apiKey string) (interface{}, error) {
	client := createHTTPClient() // Use client with TLS verification disabled

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
		maxConcurrency = 5 // Default fallback
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

	client := createHTTPClient() // Use client with TLS verification disabled
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
