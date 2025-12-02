package main

import (
	"os"
	"testing"
)

func TestParseBruFile(t *testing.T) {
	content := `meta {
  name: Login
  type: http
  seq: 1
}

post {
  url: {{baseUrl}}/auth/login
  body: json
  auth: none
}

headers {
  Content-Type: application/json
  Accept: application/json
}

body:json {
  {
    "username": "admin",
    "password": "password"
  }
}

vars:post-response {
  token: res.body.token
}
`
	tmpFile, err := os.CreateTemp("", "test.bru")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	bru, err := ParseBruFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bru.Name != "Login" {
		t.Errorf("Expected name Login, got %s", bru.Name)
	}
	if bru.Method != "POST" {
		t.Errorf("Expected method POST, got %s", bru.Method)
	}
	if bru.Url != "{{baseUrl}}/auth/login" {
		t.Errorf("Expected url {{baseUrl}}/auth/login, got %s", bru.Url)
	}
	if len(bru.Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(bru.Headers))
	}
	if len(bru.Vars) != 1 {
		t.Errorf("Expected 1 var, got %d", len(bru.Vars))
	}
	// Check body content (ignoring whitespace for simplicity or just checking containment)
	// The parser adds newlines, so we expect the body to be there.
	if bru.Body == "" {
		t.Error("Expected body to be parsed")
	}
}
