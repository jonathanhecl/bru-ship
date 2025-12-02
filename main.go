package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// arrayFlags allows setting multiple flags with the same name
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var folders string
	var replaces arrayFlags
	var removes arrayFlags
	var input string
	var output string

	flag.StringVar(&folders, "folders", "", "Comma-separated list of folders to include (e.g., Core,Users)")
	flag.Var(&replaces, "replace", "Variable replacement in format key=value (can be repeated)")
	flag.Var(&removes, "remove", "Variable to remove (can be repeated)")
	flag.StringVar(&input, "input", ".", "Root directory of Bruno collection")
	flag.StringVar(&output, "output", "collection.json", "Output file path")

	var env string
	flag.StringVar(&env, "env", "", "Environment name to load variables from (e.g., Production)")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	folderList := []string{}
	if folders != "" {
		folderList = strings.Split(folders, ",")
	}

	replaceMap := make(map[string]string)

	// Load environment variables if specified
	if env != "" {
		envPath := filepath.Join(input, "environments", env+".bru")
		envVars, err := ParseEnvFile(envPath)
		if err != nil {
			fmt.Printf("Warning: Could not load environment file %s: %v\n", envPath, err)
		} else {
			fmt.Printf("Loaded environment: %s\n", env)
			for k, v := range envVars {
				replaceMap[k] = v
			}
		}
	}

	for _, r := range replaces {
		parts := strings.SplitN(r, "=", 2)
		if len(parts) == 2 {
			replaceMap[parts[0]] = parts[1]
		}
	}

	config := Config{
		Folders: folderList,
		Replace: replaceMap,
		Remove:  removes,
		Input:   input,
		Output:  output,
		Verbose: verbose,
	}

	// Generate output filename if default or empty
	if config.Output == "collection.json" || config.Output == "" {
		prefix := "FullCollection"
		if len(config.Folders) > 0 {
			prefix = strings.Join(config.Folders, "")
		}
		timestamp := time.Now().Format("2006-01-02-150405")
		config.Output = fmt.Sprintf("%s-%s.json", prefix, timestamp)
		output = config.Output // Update local variable too for consistency
	}

	fmt.Printf("Starting conversion with config: %+v\n", config)

	collection, err := WalkAndConvert(config)
	if err != nil {
		fmt.Printf("Error converting: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Create(output)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(collection); err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Conversion completed successfully!")
}
