package main

import (
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

	item := BruToPostman(bru, config)
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

	item := BruToPostman(bru, config)
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

	item := BruToPostman(bru, config)
	if item != nil {
		t.Error("Expected item to be nil (skipped) because it uses removed variable in Auth")
	}
}
