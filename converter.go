package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Folders     []string
	Replace     map[string]string
	Remove      []string
	Ignore      []string
	Input       string
	Output      string
	Verbose     bool
	KeepFolders bool
}

// WalkAndConvert walks the directory and converts .bru files to Postman collection
func WalkAndConvert(config Config) (*PostmanCollection, error) {
	collectionName := "Bruno Collection"

	// Try to read bruno.json
	brunoConfigPath := filepath.Join(config.Input, "bruno.json")
	if fileContent, err := os.ReadFile(brunoConfigPath); err == nil {
		var brunoConfig BrunoConfig
		if err := json.Unmarshal(fileContent, &brunoConfig); err == nil && brunoConfig.Name != "" {
			collectionName = brunoConfig.Name
		}
	} else {
		// Fallback to directory name if bruno.json is missing
		if absPath, err := filepath.Abs(config.Input); err == nil {
			collectionName = filepath.Base(absPath)
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	collection := &PostmanCollection{
		Info: Info{
			Name:        collectionName,
			Description: fmt.Sprintf("Exported on %s", timestamp),
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item:     []Item{},
		Variable: []Variable{},
	}

	// Populate Collection Variables
	existingVars := make(map[string]bool)
	for k, v := range config.Replace {
		collection.Variable = append(collection.Variable, Variable{
			Key:   k,
			Value: v,
		})
		existingVars[k] = true
	}

	// Try to read collection.bru for global variables and auth
	var globalAuth map[string]string
	collectionBruPath := filepath.Join(config.Input, "collection.bru")
	if _, err := os.Stat(collectionBruPath); err == nil {
		if bru, err := ParseBruFile(collectionBruPath); err == nil {
			globalAuth = bru.Auth
			for _, v := range bru.Vars {
				if !existingVars[v.Key] {
					collection.Variable = append(collection.Variable, Variable{
						Key:   v.Key,
						Value: v.Value,
					})
					existingVars[v.Key] = true
				}
			}
		}
	}

	// Helper function to process items
	processItems := func(folderPath string, parentAuth map[string]string) ([]Item, error) {
		item, err := processFolder(folderPath, config, parentAuth)
		if err != nil {
			return nil, err
		}
		if item != nil {
			if config.KeepFolders {
				return []Item{*item}, nil
			} else {
				// Flatten: return the items inside the folder
				return item.Item, nil
			}
		}
		return []Item{}, nil
	}

	// If folders are specified, only process those
	if len(config.Folders) > 0 {
		for _, folderName := range config.Folders {
			folderPath := filepath.Join(config.Input, folderName)
			items, err := processItems(folderPath, globalAuth)
			if err != nil {
				fmt.Printf("Warning: Could not process folder '%s': %v\n", folderPath, err)
				continue
			}
			collection.Item = append(collection.Item, items...)
		}
	} else {
		// Process all folders in root
		entries, err := os.ReadDir(config.Input)
		if err != nil {
			fmt.Printf("Warning: Could not read input directory '%s': %v\n", config.Input, err)
			return collection, nil
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				items, err := processItems(filepath.Join(config.Input, entry.Name()), globalAuth)
				if err != nil {
					return nil, err
				}
				collection.Item = append(collection.Item, items...)
			}
		}
	}

	return collection, nil
}

func processFolder(path string, config Config, parentAuth map[string]string) (*Item, error) {
	if config.Verbose {
		fmt.Printf("Scanning folder: %s\n", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	item := &Item{
		Name: info.Name(),
		Item: []Item{},
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Determine current folder auth
	currentAuth := parentAuth
	folderBruPath := filepath.Join(path, "folder.bru")
	if _, err := os.Stat(folderBruPath); err == nil {
		if bru, err := ParseBruFile(folderBruPath); err == nil {
			// If folder has auth, check if it is inherit
			if len(bru.Auth) > 0 {
				isInherit := false
				if val, ok := bru.Auth["inherit"]; ok && val == "true" {
					isInherit = true
				} else if mode, ok := bru.Auth["mode"]; ok && mode == "inherit" {
					isInherit = true
				}

				if !isInherit {
					currentAuth = bru.Auth
				}
			}
		}
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			subItem, err := processFolder(fullPath, config, currentAuth)
			if err != nil {
				return nil, err
			}
			if subItem != nil {
				item.Item = append(item.Item, *subItem)
			}
		} else if strings.HasSuffix(entry.Name(), ".bru") {
			// Ignore folder.bru files as they only contain metadata
			if entry.Name() == "folder.bru" {
				continue
			}

			bru, err := ParseBruFile(fullPath)
			if err != nil {
				// Log error but maybe continue? For now return error
				return nil, err
			}

			// Check ignore patterns
			shouldIgnore := false
			for _, pattern := range config.Ignore {
				if strings.Contains(bru.Name, pattern) {
					shouldIgnore = true
					if config.Verbose {
						fmt.Printf("[SKIP] Skipped: %s (matches ignore pattern '%s')\n", bru.Name, pattern)
					}
					break
				}
			}
			if shouldIgnore {
				continue
			}

			postmanItem := BruToPostman(bru, config, currentAuth)
			if postmanItem != nil {
				item.Item = append(item.Item, *postmanItem)
				if config.Verbose {
					fmt.Printf("[OK] Exported: %s\n", bru.Name)
				}
			}
		}
	}

	if len(item.Item) == 0 {
		return nil, nil
	}

	return item, nil
}

func BruToPostman(bru *BruFile, config Config, parentAuth map[string]string) *Item {
	// Apply replacements to URL and Body
	url := bru.Url
	body := bru.Body

	// Check if the endpoint uses any removed variables
	for _, r := range config.Remove {
		placeholder := "{{" + r + "}}"
		if strings.Contains(url, placeholder) || strings.Contains(body, placeholder) {
			if config.Verbose {
				fmt.Printf("[SKIP] Skipped: %s (uses removed variable '%s' in URL or Body)\n", bru.Name, r)
			}
			return nil
		}
		// Check Auth
		for _, v := range bru.Auth {
			if strings.Contains(v, placeholder) {
				if config.Verbose {
					fmt.Printf("[SKIP] Skipped: %s (uses removed variable '%s' in Auth)\n", bru.Name, r)
				}
				return nil
			}
		}
	}

	// We NO LONGER replace variables in the URL string.
	// Instead, we rely on Postman Collection Variables.
	// However, we might want to clean up the URL if it has issues, but generally {{var}} is fine.

	// Build Headers
	var headers []Header
	for _, h := range bru.Headers {
		// Check removals
		remove := false
		for _, r := range config.Remove {
			if strings.Contains(h.Key, r) || strings.Contains(h.Value, "{{"+r+"}}") {
				remove = true
				break
			}
		}
		if !remove {
			headers = append(headers, Header{
				Key:   h.Key,
				Value: h.Value,
				Type:  "text",
			})
		}
	}

	// Build Request
	req := &Request{
		Method: bru.Method,
		Header: headers,
		Url: Url{
			Raw: url,
		},
		Description: bru.Docs,
	}

	// Handle Auth
	// Logic: If bru.Auth is present and not "inherit", use it.
	// If it is "inherit" or missing, use parentAuth.
	var effectiveAuth map[string]string

	// Check if explicit auth is set
	if len(bru.Auth) > 0 {
		// Check if it is explicitly set to inherit
		if val, ok := bru.Auth["inherit"]; ok && val == "true" {
			effectiveAuth = parentAuth
		} else if _, ok := bru.Auth["awsv4"]; ok {
			// TODO: Handle other auth types if needed, for now just use what's there
			effectiveAuth = bru.Auth
		} else if _, ok := bru.Auth["bearer"]; ok {
			effectiveAuth = bru.Auth
		} else if _, ok := bru.Auth["basic"]; ok {
			effectiveAuth = bru.Auth
		} else {
			// Default fallback or custom logic
			// If key "mode" exists (from our parser logic for 'auth { mode: ... }')
			if mode, ok := bru.Auth["mode"]; ok {
				if mode == "inherit" {
					effectiveAuth = parentAuth
				} else {
					effectiveAuth = bru.Auth
				}
			} else {
				// No mode, maybe just inherit?
				// If empty, use parent?
				// But len > 0.
				// Let's assume if it's not inherit, it's specific.
				effectiveAuth = bru.Auth
			}
		}
	} else {
		// No auth defined, inherit by default?
		// Bruno defaults to inherit usually.
		effectiveAuth = parentAuth
	}

	if len(effectiveAuth) > 0 {
		// Construct Postman Auth
		// We expect effectiveAuth to contain keys like "mode", "token", "username", "password"
		// Our parser flattens "auth { mode: bearer }" and "auth:bearer { token: ... }" into one map.

		mode := effectiveAuth["mode"]
		if mode == "" {
			// Try to infer mode
			if _, ok := effectiveAuth["token"]; ok {
				mode = "bearer"
			} else if _, ok := effectiveAuth["username"]; ok {
				mode = "basic"
			}
		}

		if mode == "bearer" {
			req.Auth = &PostmanAuth{
				Type: "bearer",
				Bearer: []AuthElement{
					{
						Key:   "token",
						Value: effectiveAuth["token"],
						Type:  "string",
					},
				},
			}
		} else if mode == "basic" {
			req.Auth = &PostmanAuth{
				Type: "basic",
				Basic: []AuthElement{
					{
						Key:   "username",
						Value: effectiveAuth["username"],
						Type:  "string",
					},
					{
						Key:   "password",
						Value: effectiveAuth["password"],
						Type:  "string",
					},
				},
			}
		}
	}

	// Parse URL components
	// Example: https://{{serverURL}}/status
	// Protocol: https
	// Host: {{serverURL}}
	// Path: status

	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", 2)
		req.Url.Protocol = parts[0]
		remaining := parts[1]

		// Split host and path
		pathParts := strings.Split(remaining, "/")
		if len(pathParts) > 0 {
			req.Url.Host = []string{pathParts[0]}
			if len(pathParts) > 1 {
				req.Url.Path = pathParts[1:]
			}
		}
	} else {
		// No protocol, maybe just {{baseUrl}}/path
		pathParts := strings.Split(url, "/")
		if len(pathParts) > 0 {
			req.Url.Host = []string{pathParts[0]}
			if len(pathParts) > 1 {
				req.Url.Path = pathParts[1:]
			}
		}
	}

	// Handle Body
	if body != "" {
		req.Body = &Body{
			Mode: "raw", // Default to raw
			Raw:  body,
		}
		// Add options for JSON if needed
		if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
			req.Body.Options = map[string]interface{}{
				"raw": map[string]string{
					"language": "json",
				},
			}
		}
	}

	item := &Item{
		Name:    bru.Name,
		Request: req,
	}

	// Handle Examples (Responses)
	for _, ex := range bru.Examples {
		// Convert BruExample to PostmanResponse
		pmResponse := PostmanResponse{
			Name: ex.Name,
			OriginalRequest: &Request{
				Method: ex.Request.Method,
				Url: Url{
					Raw: ex.Request.Url,
				},
				// We might need to parse URL for originalRequest too
			},
			Status:                 ex.Response.StatusText,
			Code:                   ex.Response.Status,
			PostmanPreviewLanguage: "json", // Default to json
			Body:                   ex.Response.Body,
		}

		// Parse OriginalRequest URL
		if strings.Contains(ex.Request.Url, "://") {
			parts := strings.SplitN(ex.Request.Url, "://", 2)
			pmResponse.OriginalRequest.Url.Protocol = parts[0]
			remaining := parts[1]
			pathParts := strings.Split(remaining, "/")
			if len(pathParts) > 0 {
				pmResponse.OriginalRequest.Url.Host = []string{pathParts[0]}
				if len(pathParts) > 1 {
					pmResponse.OriginalRequest.Url.Path = pathParts[1:]
				}
			}
		} else {
			pathParts := strings.Split(ex.Request.Url, "/")
			if len(pathParts) > 0 {
				pmResponse.OriginalRequest.Url.Host = []string{pathParts[0]}
				if len(pathParts) > 1 {
					pmResponse.OriginalRequest.Url.Path = pathParts[1:]
				}
			}
		}

		// Headers
		for _, h := range ex.Response.Headers {
			pmResponse.Header = append(pmResponse.Header, Header{
				Key:   h.Key,
				Value: h.Value,
			})
		}

		item.Response = append(item.Response, pmResponse)
	}

	return item
}
