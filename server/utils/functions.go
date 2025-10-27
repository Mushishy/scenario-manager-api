package utils

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func CompareConfigs(config1, config2 interface{}) bool {
	return fmt.Sprintf("%v", config1) == fmt.Sprintf("%v", config2)
}

func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// containsAll checks if all items in needles are present in haystack
func ContainsAll(haystack, needles []string) bool {
	for _, needle := range needles {
		found := false
		for _, hay := range haystack {
			if hay == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// ValidateDateTimeRange validates date time format and ensures stop time is after start time
func ValidateDateTimeRange(startTime, stopTime string) bool {
	layout := "02/01/2006 15:04"

	// Validate start time format
	if startTime != "" {
		parsedStartTime, err := time.Parse(layout, startTime)
		if err != nil {
			return false
		}

		// Additional validation to ensure the parsed time matches the input
		if parsedStartTime.Format(layout) != startTime {
			return false
		}
	}

	// Validate stop time format
	if stopTime != "" {
		parsedStopTime, err := time.Parse(layout, stopTime)
		if err != nil {
			return false
		}

		// Additional validation to ensure the parsed time matches the input
		if parsedStopTime.Format(layout) != stopTime {
			return false
		}
	}

	// Validate order if both times are provided
	if startTime != "" && stopTime != "" {
		start, _ := time.Parse(layout, startTime)
		stop, _ := time.Parse(layout, stopTime)

		if !stop.After(start) {
			return false
		}
	}

	return true
}
