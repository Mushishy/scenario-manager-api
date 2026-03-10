package handlers

import (
	"dulus/server/config"
	"dulus/server/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func PutTestingStart(c *gin.Context) {
	utils.ExecuteTestingAction(c, "/testing/start/", nil)
}

func PutTestingStop(c *gin.Context) {
	utils.ExecuteTestingAction(c, "/testing/stop/", gin.H{"force": true})
}

func PutPowerOn(c *gin.Context) {
	utils.ExecuteTestingAction(c, "/range/poweron", gin.H{"machines": []string{"all"}})
}

func PutPowerOff(c *gin.Context) {
	utils.ExecuteTestingAction(c, "/range/poweroff", gin.H{"machines": []string{"all"}})
}

func GetTestingStatus(c *gin.Context) {
	poolId, ok := utils.GetRequiredQueryParam(c, "poolId")
	if !ok {
		return
	}

	poolPath, ok := utils.ValidateFolderId(c, config.PoolFolder, poolId)
	if !ok {
		return
	}

	pool, ok := utils.ReadPoolWithResponse(c, poolPath)
	if !ok {
		return
	}

	var users []string
	if pool.Type == "SHARED" {
		_, mainUsers := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = mainUsers
	} else {
		userIds, _ := utils.ExtractUserIdsAndMainUserIdsFromPool(pool)
		users = userIds
	}

	apiKey := c.Request.Header.Get("X-API-Key")

	requests := make([]utils.LudusRequest, len(users))
	for i, userID := range users {
		requests[i] = utils.LudusRequest{
			Method:  "GET",
			URL:     config.LudusUrl + "/range?userID=" + userID,
			Payload: nil,
			UserID:  userID,
		}
	}

	responses := utils.MakeConcurrentLudusRequests(requests, apiKey, config.MaxConcurrentRequests)

	// Check testing enabled status
	var testingEnabledValues []bool
	allPoweredOn := true
	atLeastOneRangeHasNoVMs := false

	for _, resp := range responses {
		if resp.Error != nil || resp.Response == nil {
			continue
		}

		// Check if response is an empty array (user doesn't exist)
		if respArray, ok := resp.Response.([]interface{}); ok && len(respArray) == 0 {
			continue
		}

		// Parse the response to extract testingEnabled
		if respMap, ok := resp.Response.(map[string]interface{}); ok {
			if numberOfVMs, ok := respMap["numberOfVMs"].(float64); ok {
				if int(numberOfVMs) == 0 {
					atLeastOneRangeHasNoVMs = true
				}
			}
			fmt.Println(respMap)
			if testingEnabled, exists := respMap["testingEnabled"]; exists {
				if boolValue, ok := testingEnabled.(bool); ok {
					testingEnabledValues = append(testingEnabledValues, boolValue)
				}
			}
			// if VMs exist, check their power state if at least one is off than you dont have to check the rest
			if VMs, exists := respMap["VMs"]; exists && allPoweredOn {
				if vmsArray, ok := VMs.([]interface{}); ok && len(vmsArray) > 0 {
					fmt.Println(vmsArray...)
					for _, vm := range vmsArray {
						if vmMap, ok := vm.(map[string]interface{}); ok {
							if poweredOn, exists := vmMap["poweredOn"]; exists {
								fmt.Println(vmMap["name"], poweredOn)
								if poweredOnBool, ok := poweredOn.(bool); ok && !poweredOnBool {
									allPoweredOn = false
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Check if all testingEnabled values are the same
	allSame := true
	var testingEnabledValue bool

	if len(testingEnabledValues) > 0 {
		testingEnabledValue = testingEnabledValues[0]
		for _, value := range testingEnabledValues {
			if value != testingEnabledValue {
				allSame = false
				break
			}
		}
	} else {
		allSame = false
	}

	// Determine power state
	var powerState bool
	if atLeastOneRangeHasNoVMs {
		powerState = false
	} else if allPoweredOn {
		powerState = true
	} else {
		powerState = false
	}

	c.JSON(http.StatusOK, gin.H{
		"allSame":        allSame,
		"testingEnabled": testingEnabledValue,
		"poweredOn":      powerState,
	})
}
