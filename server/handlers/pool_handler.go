package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
)

func PostPool(c *gin.Context) {
	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/pool_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
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

	// Validate TopologyId
	if _, err := utils.ValidateFolderID(config.TopologyConfigFolder, input["topologyId"].(string)); err != nil {
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Validate CtfdDataId
	if _, err := utils.ValidateFolderID(config.CtfdDataFolder, input["ctfdDataId"].(string)); err != nil {
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Validate UsersAndTeams if provided
	if usersAndTeams, ok := input["usersAndTeams"].([]interface{}); ok && len(usersAndTeams) > 0 {
		if err := utils.ValidateUsersAndTeams(usersAndTeams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
	}

	// If type is SHARED or INDIVIDUAL, MainUser is required
	if input["type"] == "SHARED" || input["type"] == "INDIVIDUAL" {
		if mainUser, ok := input["mainUser"].(string); !ok || mainUser == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
	}

	// Generate a new unique ID for the pool
	poolId, err := utils.GenerateUniqueID(config.PoolFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Create pool directory
	poolPath := filepath.Join(config.PoolFolder, poolId)
	if err := os.MkdirAll(poolPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Save pool data to file
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{"message": "Uploaded successfully", "id": poolId})
}

func PatchPoolUsers(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/pool_users_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
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

	// Validate pool exists
	poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Validate UsersAndTeams if provided
	if usersAndTeams, ok := input["usersAndTeams"].([]interface{}); ok && len(usersAndTeams) > 0 {
		if err := utils.ValidateUsersAndTeams(usersAndTeams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		}
	}

	// Read existing pool data
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	var poolData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Update usersAndTeams
	poolData["usersAndTeams"] = input["usersAndTeams"]

	// Save updated data
	file, err = os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolTopology(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/pool_topology_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
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

	// Validate pool exists
	poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Validate TopologyId exists
	if _, err := utils.ValidateFolderID(config.TopologyConfigFolder, input["topologyId"].(string)); err != nil {
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Read existing pool data
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	var poolData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Update topologyId
	poolData["topologyId"] = input["topologyId"]

	// Save updated data
	file, err = os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolCtfdData(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/pool_ctfd_data_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
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

	// Validate pool exists
	poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Validate CtfdDataId exists
	if _, err := utils.ValidateFolderID(config.CtfdDataFolder, input["ctfdDataId"].(string)); err != nil {
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		}
		return
	}

	// Read existing pool data
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	var poolData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Update ctfdDataId
	poolData["ctfdDataId"] = input["ctfdDataId"]

	// Save updated data
	file, err = os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func PatchPoolNote(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Load the JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://schemas/pool_note_schema.json")

	// Parse the JSON input
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate the input against the schema
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

	// Validate pool exists
	poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Read existing pool data
	filePath := filepath.Join(poolPath, "pool.json")
	file, err := os.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	var poolData map[string]interface{}
	if err := json.NewDecoder(file).Decode(&poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	// Update note
	poolData["note"] = input["note"]

	// Save updated data
	file, err = os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(poolData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated successfully"})
}

func GetPool(c *gin.Context) {
	poolId := c.Query("poolId")
	users := c.Query("users")

	if poolId != "" {
		// Validate pool exists
		poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
		switch err {
		case os.ErrInvalid:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
			return
		case os.ErrNotExist:
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}

		// Read pool data
		filePath := filepath.Join(poolPath, "pool.json")
		file, err := os.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			}
			return
		}
		defer file.Close()

		var poolData map[string]interface{}
		if err := json.NewDecoder(file).Decode(&poolData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		if users == "true" {
			// Return only users for the given pool
			response := gin.H{
				"poolId": poolId,
			}
			if usersAndTeams, exists := poolData["usersAndTeams"]; exists {
				response["usersAndTeams"] = usersAndTeams
			}
			c.JSON(http.StatusOK, response)
		} else {
			// Return full pool data
			poolData["poolId"] = poolId
			c.JSON(http.StatusOK, poolData)
		}
	} else {
		// Return all pools without mainUser and usersAndTeams
		pools, err := os.ReadDir(config.PoolFolder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		var poolList []gin.H
		for _, pool := range pools {
			if pool.IsDir() {
				poolPath := filepath.Join(config.PoolFolder, pool.Name())
				filePath := filepath.Join(poolPath, "pool.json")

				file, err := os.Open(filePath)
				if err != nil {
					continue // Skip pools that don't have pool.json or can't be read
				}

				var poolData map[string]interface{}
				if err := json.NewDecoder(file).Decode(&poolData); err != nil {
					file.Close()
					continue // Skip files that can't be decoded
				}
				file.Close()

				// Remove mainUser and usersAndTeams for list view
				delete(poolData, "mainUser")
				delete(poolData, "usersAndTeams")
				poolData["poolId"] = pool.Name()

				poolList = append(poolList, poolData)
			}
		}

		c.JSON(http.StatusOK, poolList)
	}
}

func DeletePool(c *gin.Context) {
	poolId := c.Query("poolId")
	if poolId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	}

	// Validate pool exists
	poolPath, err := utils.ValidateFolderID(config.PoolFolder, poolId)
	switch err {
	case os.ErrInvalid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
		return
	case os.ErrNotExist:
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	// Delete the pool directory
	if err := os.RemoveAll(poolPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}

	c.Status(http.StatusNoContent)
}
