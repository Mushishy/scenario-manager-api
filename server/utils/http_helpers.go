package utils

import (
	"crypto/tls"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetRequiredQueryParam gets a required query parameter and handles error response
func GetRequiredQueryParam(c *gin.Context, paramName string) (string, bool) {
	value := c.Query(paramName)
	if value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return "", false
	}
	return value, true
}

// GetOptionalQueryParam gets an optional query parameter
func GetOptionalQueryParam(c *gin.Context, paramName string) string {
	return c.Query(paramName)
}

// createHTTPClient creates HTTP client with TLS verification disabled
func createHTTPClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

// ConvertResponsesToResults converts LudusResponse slice to gin.H results format
func ConvertResponsesToResults(responses []LudusResponse) []gin.H {
	var results []gin.H
	for _, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
		} else {
			results = append(results, gin.H{"userId": resp.UserID, "response": resp.Response})
		}
	}
	return results
}
