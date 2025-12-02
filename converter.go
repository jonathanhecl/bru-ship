package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Folders     []string
	Replace     map[string]string
	Remove      []string
	Input       string
	Output      string
	Verbose     bool
	KeepFolders bool
}

// WalkAndConvert walks the directory and converts .bru files to Postman collection
func WalkAndConvert(config Config) (*PostmanCollection, error) {
	collection := &PostmanCollection{
		Info: Info{
			Name:   "Bruno Collection", // Could be parameterized
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item:     []Item{},
		Variable: []Variable{},
	}

	// Populate Collection Variables
	for k, v := range config.Replace {
		collection.Variable = append(collection.Variable, Variable{
			Key:   k,
			Value: v,
		})
	}

	// Helper function to process items
	processItems := func(folderPath string) ([]Item, error) {
		item, err := processFolder(folderPath, config)
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
			items, err := processItems(folderPath)
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
				items, err := processItems(filepath.Join(config.Input, entry.Name()))
				if err != nil {
					return nil, err
				}
				collection.Item = append(collection.Item, items...)
			}
		}
	}

	return collection, nil
}

func processFolder(path string, config Config) (*Item, error) {
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

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			subItem, err := processFolder(fullPath, config)
			if err != nil {
				return nil, err
			}
			if subItem != nil {
				item.Item = append(item.Item, *subItem)
			}
		} else if strings.HasSuffix(entry.Name(), ".bru") {
			bru, err := ParseBruFile(fullPath)
			if err != nil {
				// Log error but maybe continue? For now return error
				return nil, err
			}
			postmanItem := BruToPostman(bru, config)
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

func BruToPostman(bru *BruFile, config Config) *Item {
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
	}

	// Parse URL components
	// Simple parsing to extract host and path
	// If URL starts with http/https, we can try to parse it
	// If it starts with {{, it's a variable

	// Split by / to get path segments
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		// This is a very basic parser, Postman is quite flexible with "raw"
		// But populating Host and Path helps

		// If the first part is http: or https:, then parts[2] is host
		// If the first part is {{baseUrl}}, then that is host

		// For now, let's just rely on Raw, but ensure variables are preserved.
		// The user complained about invalid URL. Often this is because of spaces in replaced values.
		// Since we are NOT replacing values anymore, the {{var}} will be preserved.

		// We can try to populate Host/Path for better structure
		req.Url.Host = []string{}
		req.Url.Path = []string{}

		// TODO: A better URL parser would be good, but 'Raw' is usually sufficient if variables are used correctly.
	}

	// Handle Body
	if body != "" {
		req.Body = &Body{
			Mode: "raw", // Default to raw
			Raw:  body,
		}
		// Try to guess mode from type or headers?
		// Bruno usually has `body:json`
		// We can check if body looks like JSON
		// if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
		// 	// It's likely JSON, Postman might want options
		// 	// For now, raw is fine.
		// }
	}

	return &Item{
		Name:    bru.Name,
		Request: req,
	}
}
