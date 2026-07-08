package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// the function takes a file name and returns a list of URLs or an error
func LoadUrlsFromJSON(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var urls []string

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&urls); err != nil {
		return nil, fmt.Errorf("JSON parsing error: %w", err)
	}

	return urls, nil
}
