package main

import (
	"bufio"
	"fmt"
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
	var docsBuffer strings.Builder
	blockIndents := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Calculate indentation (spaces/tabs before content)
		indent := ""
		if len(trimmedLine) > 0 {
			indent = line[:strings.Index(line, trimmedLine)]
		}

		if trimmedLine == "" && !strings.HasPrefix(currentBlock, "body") && currentBlock != "docs" && currentBlock != "example" {
			continue
		}

		// Detect block start
		if strings.HasSuffix(trimmedLine, " {") && !strings.HasPrefix(currentBlock, "example") {
			blockName := strings.TrimSuffix(trimmedLine, " {")
			if blockName == "meta" || blockName == "headers" || blockName == "vars:pre-request" || blockName == "vars:post-response" || strings.HasPrefix(blockName, "body") || blockName == "docs" || strings.HasPrefix(blockName, "auth") || blockName == "example" {
				currentBlock = blockName
				blockIndents[currentBlock] = indent
				if blockName == "example" {
					// Start a new example
					bru.Examples = append(bru.Examples, BruExample{})
				}
				continue
			}
			// HTTP Methods
			if blockName == "get" || blockName == "post" || blockName == "put" || blockName == "delete" || blockName == "patch" || blockName == "options" || blockName == "head" {
				bru.Method = strings.ToUpper(blockName)
				currentBlock = "request"
				blockIndents[currentBlock] = indent
				continue
			}
		}

		// Detect block end
		if trimmedLine == "}" {
			if strings.HasPrefix(currentBlock, "body") {
				// For body, check if it's the closing brace of the block
				// strict check using indentation
				if line == blockIndents[currentBlock]+"}" {
					bru.Body = bodyBuffer.String()
					bodyBuffer.Reset()
					currentBlock = ""
					continue
				}
			} else if currentBlock == "docs" {
				if line == blockIndents[currentBlock]+"}" {
					bru.Docs = docsBuffer.String()
					docsBuffer.Reset()
					currentBlock = ""
					continue
				}
			} else if currentBlock == "example" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = ""
					continue
				}
			} else if strings.HasPrefix(currentBlock, "example-") {
				// Let switch handle nested example blocks
			} else {
				// For other blocks, check indentation if available
				if val, ok := blockIndents[currentBlock]; ok {
					if line == val+"}" {
						currentBlock = ""
						continue
					}
				} else {
					// Fallback for blocks without stored indent (shouldn't happen for new blocks)
					currentBlock = ""
					continue
				}
			}
		}

		var idx int = -1
		if len(bru.Examples) > 0 {
			idx = len(bru.Examples) - 1
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
			docsBuffer.WriteString(line + "\n")
		case "example":
			if idx == -1 {
				continue
			}

			if strings.HasSuffix(trimmedLine, "request: {") {
				currentBlock = "example-request"
				blockIndents[currentBlock] = indent
				continue
			}
			if strings.HasSuffix(trimmedLine, "response: {") {
				currentBlock = "example-response"
				blockIndents[currentBlock] = indent
				continue
			}

			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "name" {
					bru.Examples[idx].Name = val
				}
			}

		case "example-request":
			if idx == -1 {
				continue
			}
			if strings.HasSuffix(trimmedLine, "body:json: {") || strings.HasSuffix(trimmedLine, "body: {") {
				currentBlock = "example-request-body"
				blockIndents[currentBlock] = indent
				continue
			}
			// Parse method, url
			parts := strings.SplitN(trimmedLine, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key == "url" {
					bru.Examples[idx].Request.Url = val
				} else if key == "method" {
					bru.Examples[idx].Request.Method = val
				}
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example"
				}
			}

		case "example-request-body":
			if idx == -1 {
				continue
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example-request"
				} else {
					bru.Examples[idx].Request.Body += line + "\n"
				}
			} else {
				bru.Examples[idx].Request.Body += line + "\n"
			}

		case "example-response":
			if idx == -1 {
				continue
			}
			if strings.HasSuffix(trimmedLine, "body: {") {
				currentBlock = "example-response-body"
				blockIndents[currentBlock] = indent
				continue
			}
			if strings.HasSuffix(trimmedLine, "status: {") {
				currentBlock = "example-response-status"
				blockIndents[currentBlock] = indent
				continue
			}
			if strings.HasSuffix(trimmedLine, "headers: {") {
				currentBlock = "example-response-headers"
				blockIndents[currentBlock] = indent
				continue
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example"
				}
			}

		case "example-response-status":
			if idx == -1 {
				continue
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example-response"
				}
			} else {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					if key == "code" {
						fmt.Sscanf(val, "%d", &bru.Examples[idx].Response.Status)
					} else if key == "text" {
						bru.Examples[idx].Response.StatusText = val
					}
				}
			}

		case "example-response-headers":
			if idx == -1 {
				continue
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example-response"
				}
			} else {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					bru.Examples[idx].Response.Headers = append(bru.Examples[idx].Response.Headers, KeyValue{
						Key:   strings.TrimSpace(parts[0]),
						Value: strings.TrimSpace(parts[1]),
					})
				}
			}

		case "example-response-body":
			if idx == -1 {
				continue
			}
			if trimmedLine == "}" {
				if line == blockIndents[currentBlock]+"}" {
					currentBlock = "example-response"
				} else {
					// Handle content: ''' ... '''
					if strings.Contains(line, "'''") {
						// Just skip the triple quotes lines for now or handle them?
						// The content is usually inside triple quotes.
						// Let's just append everything for now and clean up later if needed.
						if !strings.Contains(line, "content:") && strings.TrimSpace(line) != "'''" {
							bru.Examples[idx].Response.Body += line + "\n"
						}
					} else {
						if strings.TrimSpace(line) != "type: json" {
							bru.Examples[idx].Response.Body += line + "\n"
						}
					}
				}
			} else {
				// Handle content: ''' ... '''
				if strings.Contains(line, "'''") {
					if !strings.Contains(line, "content:") && strings.TrimSpace(line) != "'''" {
						bru.Examples[idx].Response.Body += line + "\n"
					}
				} else {
					if strings.TrimSpace(line) != "type: json" {
						bru.Examples[idx].Response.Body += line + "\n"
					}
				}
			}

		default:
			if strings.HasPrefix(currentBlock, "auth") {
				parts := strings.SplitN(trimmedLine, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					bru.Auth[key] = val
				}
			} else if strings.HasPrefix(currentBlock, "body") {
				// fmt.Printf("DEBUG: Writing to bodyBuffer: %s\n", line)
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
				vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return vars, nil
}
