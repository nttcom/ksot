package util

import (
	"encoding/json"
	"os"
)

func LoadJson(inputPath string, obj interface{}) error {
	jsonByte, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonByte, &obj); err != nil {
		return err
	}
	return nil
}

func WriteJson(inputPath string, obj interface{}) error {
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(inputPath, jsonBytes, 0600)
}
