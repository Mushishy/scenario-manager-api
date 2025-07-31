package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

func ImportUsers(c *gin.Context) {
	// Load schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ludus_users_schema.json")

	var input struct {
		UserIds []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate schema
	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(input.UserIds))
	for i, userID := range input.UserIds {
		payload := gin.H{
			"name":    userID,
			"userID":  userID,
			"isAdmin": false,
		}
		requests[i] = utils.LudusRequest{
			Method:  "POST",
			URL:     config.LudusAdminUrl + "/user",
			Payload: payload,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Convert to results format
	var results []gin.H
	for _, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
		} else {
			results = append(results, gin.H{"userId": resp.UserID, "response": resp.Response})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func DeleteUsers(c *gin.Context) {
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ludus_users_schema.json")

	var input struct {
		UserIds []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(input.UserIds))
	for i, userID := range input.UserIds {
		requests[i] = utils.LudusRequest{
			Method:  "DELETE",
			URL:     config.LudusAdminUrl + "/user/" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Convert to results format
	var results []gin.H
	for _, resp := range responses {
		if resp.Error != nil {
			results = append(results, gin.H{"userId": resp.UserID, "error": resp.Error.Error()})
		} else {
			results = append(results, gin.H{"userId": resp.UserID, "response": resp.Response})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func CheckUsers(c *gin.Context) {
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/ludus_users_schema.json")

	var input struct {
		UserIds []string `json:"userIds"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	if !result.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	// Prepare concurrent requests
	requests := make([]utils.LudusRequest, len(input.UserIds))
	for i, userID := range input.UserIds {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/user?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	// Execute concurrent requests
	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Convert to results format
	var results []gin.H
	for _, resp := range responses {
		exists := resp.Error == nil && resp.Response != nil
		results = append(results, gin.H{
			"userId": resp.UserID,
			"exists": exists,
		})
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}
