package main

import (
	"encoding/json"
	"testing"
)

func TestOutputJSON(t *testing.T) {
	data := map[string]interface{}{
		"name":    "test",
		"value":   42,
		"enabled": true,
	}

	// Test JSON encoding directly
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if decoded["name"] != "test" {
		t.Errorf("Expected name='test', got '%v'", decoded["name"])
	}
	if decoded["value"] != float64(42) {
		t.Errorf("Expected value=42, got '%v'", decoded["value"])
	}
	if decoded["enabled"] != true {
		t.Errorf("Expected enabled=true, got '%v'", decoded["enabled"])
	}
}

func TestOutputJSONNested(t *testing.T) {
	data := map[string]interface{}{
		"device": map[string]interface{}{
			"id":      1,
			"name":    "Test Device",
			"volume":  0.5,
			"muted":   false,
			"default": true,
		},
	}

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	device := decoded["device"].(map[string]interface{})
	if device["name"] != "Test Device" {
		t.Errorf("Expected device name='Test Device', got '%v'", device["name"])
	}
}
