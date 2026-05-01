package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBruToPostman_SkipRemoved(t *testing.T) {
	bru := &BruFile{
		Name:   "Test Request",
		Method: "GET",
		Url:    "{{baseUrl}}/api/resource?token={{secretToken}}",
		Body:   "",
	}

	config := Config{
		Remove:  []string{"secretToken"},
		Verbose: true,
	}

	item := BruToPostman(bru, config, nil)
	if item != nil {
		t.Error("Expected item to be nil (skipped) because it contains a removed variable")
	}
}

func TestBruToPostman_NotSkipped(t *testing.T) {
	bru := &BruFile{
		Name:   "Test Request",
		Method: "GET",
		Url:    "{{baseUrl}}/api/resource",
		Body:   "",
	}

	config := Config{
		Remove: []string{"secretToken"},
	}

	item := BruToPostman(bru, config, nil)
	if item == nil {
		t.Error("Expected item to NOT be nil")
	}
}

func TestBruToPostman_SkipRemovedAuth(t *testing.T) {
	bru := &BruFile{
		Name:   "Test Request",
		Method: "GET",
		Url:    "{{baseUrl}}/api/resource",
		Body:   "",
		Auth: map[string]string{
			"token": "{{secretToken}}",
		},
	}

	config := Config{
		Remove:  []string{"secretToken"},
		Verbose: true,
	}

	item := BruToPostman(bru, config, nil)
	if item != nil {
		t.Error("Expected item to be nil (skipped) because it uses removed variable in Auth")
	}
}

func TestBruToPostmanStructure(t *testing.T) {
	// Simulate a populated BruFile
	bru := &BruFile{
		Name:   "Test Request",
		Method: "GET",
		Url:    "https://api.example.com/v1/resource?filter=active&sort=desc",
		Headers: []KeyValue{
			{Key: "Content-Type", Value: "application/json", Enabled: true},
			{Key: "Authorization", Value: "Bearer token", Enabled: true},
		},
		Body: `{"foo": "bar"}`,
		Auth: map[string]string{
			"mode":  "bearer",
			"token": "secret-token",
		},
		Docs: "# Documentation\n\nThis is a test.",
	}

	config := Config{}
	parentAuth := map[string]string{}

	// Convert
	item := BruToPostman(bru, config, parentAuth)

	// Marshal to JSON to check structure
	jsonData, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal item: %v", err)
	}
	jsonString := string(jsonData)

	// Assertions
	// 1. Check Headers are not null
	if !strings.Contains(jsonString, `"header": [`) {
		t.Errorf("Header should be an array, not null. Got: %s", jsonString)
	}

	// 2. Check URL Query params
	if !strings.Contains(jsonString, `"query": [`) {
		t.Errorf("URL should have query params parsed. Got: %s", jsonString)
	}
	if !strings.Contains(jsonString, `"key": "filter"`) {
		t.Errorf("Missing query param key 'filter'")
	}
	if !strings.Contains(jsonString, `"value": "active"`) {
		t.Errorf("Missing query param value 'active'")
	}

	// 3. Check Auth
	if !strings.Contains(jsonString, `"auth": {`) {
		t.Errorf("Missing auth object")
	}
	if !strings.Contains(jsonString, `"type": "bearer"`) {
		t.Errorf("Missing auth type bearer")
	}

	// 4. Check Description
	if !strings.Contains(jsonString, `"description": "# Documentation\n\nThis is a test."`) {
		t.Errorf("Missing or incorrect description")
	}

	// 5. Check ProtocolProfileBehavior (GET with body)
	if !strings.Contains(jsonString, `"protocolProfileBehavior": {`) {
		t.Errorf("Missing protocolProfileBehavior")
	}
	if !strings.Contains(jsonString, `"disableBodyPruning": true`) {
		t.Errorf("Missing disableBodyPruning")
	}
}

func TestWalkAndConvert_RemovedCollectionVariablesAreFiltered(t *testing.T) {
	tmpDir := t.TempDir()

	collectionBru := `vars:pre-request {
  ~myCallbackApiKey: from-collection
  baseUrl: https://api.example.com
}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "collection.bru"), []byte(collectionBru), 0644); err != nil {
		t.Fatalf("failed to write collection.bru: %v", err)
	}

	config := Config{
		Input:  tmpDir,
		Remove: []string{"~myCallbackApiKey"},
		Replace: map[string]string{
			"~myCallbackApiKey": "from-replace",
			"tenantId":          "123",
		},
	}

	collection, err := WalkAndConvert(config)
	if err != nil {
		t.Fatalf("WalkAndConvert returned error: %v", err)
	}

	varMap := make(map[string]string)
	for _, v := range collection.Variable {
		varMap[v.Key] = v.Value
	}

	if _, exists := varMap["~myCallbackApiKey"]; exists {
		t.Fatalf("expected removed variable ~myCallbackApiKey to be excluded from collection.variable")
	}

	if got := varMap["tenantId"]; got != "123" {
		t.Fatalf("expected tenantId=123, got %q", got)
	}

	if got := varMap["baseUrl"]; got != "https://api.example.com" {
		t.Fatalf("expected baseUrl from collection.bru to be kept, got %q", got)
	}
}

func TestWalkAndConvert_DisabledPrefixedVariablesAreFiltered(t *testing.T) {
	tmpDir := t.TempDir()

	collectionBru := `vars:pre-request {
  ~disabledFromCollection: from-collection
  baseUrl: https://api.example.com
}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "collection.bru"), []byte(collectionBru), 0644); err != nil {
		t.Fatalf("failed to write collection.bru: %v", err)
	}

	config := Config{
		Input: tmpDir,
		Replace: map[string]string{
			"~disabledFromReplace": "from-replace",
			"tenantId":             "123",
		},
	}

	collection, err := WalkAndConvert(config)
	if err != nil {
		t.Fatalf("WalkAndConvert returned error: %v", err)
	}

	varMap := make(map[string]string)
	for _, v := range collection.Variable {
		varMap[v.Key] = v.Value
	}

	if _, exists := varMap["~disabledFromCollection"]; exists {
		t.Fatalf("expected ~disabledFromCollection to be excluded from collection.variable")
	}

	if _, exists := varMap["~disabledFromReplace"]; exists {
		t.Fatalf("expected ~disabledFromReplace to be excluded from collection.variable")
	}

	if got := varMap["tenantId"]; got != "123" {
		t.Fatalf("expected tenantId=123, got %q", got)
	}

	if got := varMap["baseUrl"]; got != "https://api.example.com" {
		t.Fatalf("expected baseUrl from collection.bru to be kept, got %q", got)
	}
}
