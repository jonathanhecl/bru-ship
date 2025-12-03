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

var (
	version string = "1.0.1"
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
	fmt.Printf("bru-ship v%s\n", version)
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

	var ignore string
	flag.StringVar(&ignore, "ignore", "", "Comma-separated list of endpoint names to ignore")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	var keepFolders bool
	flag.BoolVar(&keepFolders, "keep-folders", false, "Keep folder structure (default is to flatten)")

	var title string
	flag.StringVar(&title, "title", "", "Title for the generated Postman Collection")

	// Filter out standalone "\" arguments which might be passed by PowerShell when copy-pasting multi-line commands
	var args []string
	for _, arg := range os.Args {
		if arg != "\\" {
			args = append(args, arg)
		}
	}
	os.Args = args

	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	folderList := []string{}
	if folders != "" {
		folderList = strings.Split(folders, ",")
		for _, folder := range folderList {
			folderPath := filepath.Join(input, folder)
			if _, err := os.Stat(folderPath); os.IsNotExist(err) {
				fmt.Printf("Error: Folder does not exist: %s\n", folderPath)
				os.Exit(1)
			}
		}
	}

	ignoreList := []string{}
	if ignore != "" {
		ignoreList = strings.Split(ignore, ",")
	}

	replaceMap := make(map[string]string)

	// Load environment variables if specified
	if env != "" {
		envPath := filepath.Join(input, "environments", env+".bru")
		envVars, err := ParseEnvFile(envPath)
		if err != nil {
			fmt.Printf("Error: Could not load environment file %s: %v\n", envPath, err)
			os.Exit(1)
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
		Folders:     folderList,
		Replace:     replaceMap,
		Remove:      removes,
		Ignore:      ignoreList,
		Input:       input,
		Output:      output,
		Verbose:     verbose,
		KeepFolders: keepFolders,
		Title:       title,
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

	// Remove existing output file if it exists to ensure a clean overwrite
	if _, err := os.Stat(output); err == nil {
		if err := os.Remove(output); err != nil {
			fmt.Printf("Error removing existing output file: %v\n", err)
			os.Exit(1)
		}
		if verbose {
			fmt.Printf("Removed existing output file: %s\n", output)
		}
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

	absOutput, _ := filepath.Abs(output)
	fmt.Printf("Conversion completed successfully! Output file: %s\n", absOutput)
}
