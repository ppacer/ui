package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Functione encode JSON encodes and writes given object with given status.
func encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// Function decode decodes given HTTP request body into an expected type.
func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// getPathValueInt parses URL path argument and casts it into integer.
func getPathValueInt(r *http.Request, argName string) (int, error) {
	argValue := r.PathValue(argName)
	if argValue == "" {
		return -1, fmt.Errorf("parameter %s is unexpectedly empty", argName)
	}
	argInt, castErr := strconv.Atoi(argValue)
	if castErr != nil {
		return -1, fmt.Errorf("cannot cast parameter %s (%s) into integer",
			argName, argValue)
	}
	return argInt, nil
}

// getPathValueStr parses URL path argument and checks if it's not empty.
func getPathValueStr(r *http.Request, argName string) (string, error) {
	argValue := r.PathValue(argName)
	if argValue == "" {
		return "", fmt.Errorf("parameter %s is unexpectedly empty", argName)
	}
	return argValue, nil
}
