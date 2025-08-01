package utils

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// HandleFileReadError handles common file reading errors with HTTP responses
func HandleFileReadError(c *gin.Context, err error) bool {
	if err == os.ErrNotExist {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return true
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return true
	}
	return false
}

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
