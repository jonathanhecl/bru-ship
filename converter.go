package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Folders []string
	Replace map[string]string
	Remove  []string
	Input   string
	Output  string
	Verbose bool
}

// WalkAndConvert walks the directory and converts .bru files to Postman collection
func WalkAndConvert(config Config) (*PostmanCollection, error) {
	collection := &PostmanCollection{
		Info: Info{
			Name:   "Bruno Collection", // Could be parameterized
			Schema: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item: []Item{},
	}

	// If folders are specified, only process those
	if len(config.Folders) > 0 {
		for _, folderName := range config.Folders {
			folderPath := filepath.Join(config.Input, folderName)
			item, err := processFolder(folderPath, config)
			if err != nil {
				return nil, err
			}
			if item != nil {
				collection.Item = append(collection.Item, *item)
			}
		}
	} else {
		// Process all folders in root
		entries, err := os.ReadDir(config.Input)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				item, err := processFolder(filepath.Join(config.Input, entry.Name()), config)
				if err != nil {
					return nil, err
				}
				if item != nil {
					collection.Item = append(collection.Item, *item)
				}
			}
		}
	}

	return collection, nil
}

func processFolder(path string, config Config) (*Item, error) {
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
				fmt.Printf("Skipping endpoint '%s' because it uses removed variable '%s' in URL or Body\n", bru.Name, r)
			}
			return nil
		}
		// Check Auth
		for _, v := range bru.Auth {
			if strings.Contains(v, placeholder) {
				if config.Verbose {
					fmt.Printf("Skipping endpoint '%s' because it uses removed variable '%s' in Auth\n", bru.Name, r)
				}
				return nil
			}
		}
	}

	for k, v := range config.Replace {
		placeholder := "{{" + k + "}}"
		url = strings.ReplaceAll(url, placeholder, v)
		body = strings.ReplaceAll(body, placeholder, v)
	}

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
			// Parse host/path if needed, but Raw is often enough for Postman
			// To be more correct, we might want to parse the URL
		},
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
		if strings.HasPrefix(strings.TrimSpace(body), "{") || strings.HasPrefix(strings.TrimSpace(body), "[") {
			// It's likely JSON, Postman might want options
			// For now, raw is fine.
		}
	}

	return &Item{
		Name:    bru.Name,
		Request: req,
	}
}
