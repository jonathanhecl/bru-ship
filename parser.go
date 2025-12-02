package main

import (
	"bufio"
	"os"
	"strings"
)

// ParseBruFile parses a .bru file and returns a BruFile struct
func ParseBruFile(path string) (*BruFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bru := &BruFile{
		Headers: []KeyValue{},
		Vars:    []KeyValue{},
		Auth:    make(map[string]string),
	}
	scanner := bufio.NewScanner(file)

	var currentBlock string
	var bodyBuffer strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" && !strings.HasPrefix(currentBlock, "body") {
			continue
		}

		// Detect block start
		if strings.HasSuffix(trimmedLine, " {") {
			blockName := strings.TrimSuffix(trimmedLine, " {")
			if blockName == "meta" || blockName == "headers" || blockName == "vars:pre-request" || blockName == "vars:post-response" || strings.HasPrefix(blockName, "body") || blockName == "docs" || strings.HasPrefix(blockName, "auth") {
				currentBlock = blockName
				continue
			}
			// HTTP Methods
			if blockName == "get" || blockName == "post" || blockName == "put" || blockName == "delete" || blockName == "patch" || blockName == "options" || blockName == "head" {
				bru.Method = strings.ToUpper(blockName)
				currentBlock = "request"
				continue
			}
		}

		// Detect block end
		if trimmedLine == "}" {
			if strings.HasPrefix(currentBlock, "body") {
				// For body, check if it's the closing brace of the block
				// This is a naive check, assuming the closing brace is on its own line
				if line == "}" {
					bru.Body = bodyBuffer.String()
					currentBlock = ""
					continue
				}
			} else {
				currentBlock = ""
				continue
			}
		}

		switch currentBlock {
		case "meta":
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "name" {
					bru.Name = val
				} else if key == "type" {
					bru.Type = val
				}
			}
		case "request":
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "url" {
					bru.Url = val
				}
			}
		case "headers":
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				bru.Headers = append(bru.Headers, KeyValue{
					Key:     strings.TrimSpace(parts[0]),
					Value:   strings.TrimSpace(parts[1]),
					Enabled: true,
				})
			}
		case "vars:pre-request", "vars:post-response":
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				bru.Vars = append(bru.Vars, KeyValue{
					Key:     strings.TrimSpace(parts[0]),
					Value:   strings.TrimSpace(parts[1]),
					Enabled: true,
				})
			}
		case "docs":
			// Docs are usually markdown, just append
			// TODO: Handle docs better if needed
		default:
			if strings.HasPrefix(currentBlock, "auth") {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					bru.Auth[key] = val
				}
			} else if strings.HasPrefix(currentBlock, "body") {
				bodyBuffer.WriteString(line + "\n")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return bru, nil
}

// ParseEnvFile parses a Bruno environment file and returns a map of variables
func ParseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	var currentBlock string

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		if strings.HasSuffix(trimmedLine, " {") {
			blockName := strings.TrimSuffix(trimmedLine, " {")
			if blockName == "vars" {
				currentBlock = "vars"
				continue
			}
		}

		if trimmedLine == "}" {
			currentBlock = ""
			continue
		}

		if currentBlock == "vars" {
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				vars[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return vars, nil
}
