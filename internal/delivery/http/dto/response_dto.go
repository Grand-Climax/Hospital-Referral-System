package dto

import (
	"encoding/json"
)

type BaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// SuccessPayload helps "spread" a struct into a map with success: true
// and an optional message, without nesting it in a "data" key.
func SuccessPayload(data interface{}, message string) map[string]interface{} {
	result := make(map[string]interface{})
	result["success"] = true
	if message != "" {
		result["message"] = message
	}

	if data == nil {
		return result
	}

	// Simple and performant way to spread structs into maps at runtime in Go
	// without using reflection directly (marshaling to map first).
	// This is only for single objects; for arrays we use a key.
	b, _ := json.Marshal(data)
	json.Unmarshal(b, &result)

	// Final override to ensure success is true and message is set correctly
	// even if the struct had conflicting fields.
	result["success"] = true
	if message != "" {
		result["message"] = message
	}

	return result
}
