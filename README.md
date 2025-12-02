# bru-ship

A lightweight, powerful CLI tool written in Go to convert [Bruno](https://www.usebruno.com/) API collections (`.bru` files) into [Postman Collection v2.1](https://www.postman.com/collection/) format.

This tool is designed to be **read-only** regarding your source files. It parses your Bruno collection and generates a new JSON file for Postman, ensuring your original data remains untouched.

## Features

- **Recursive Conversion**: Automatically traverses your project directories to find all `.bru` files.
- **Selective Export**: Filter which folders to include in the final collection.
- **Variable Replacement**: Replace Bruno variables (e.g., `{{baseUrl}}`) with specific values or Postman variables during conversion.
- **Sensitive Data Sanitization**: Remove specific headers or variables (like Admin Tokens) from the exported collection. Endpoints using removed variables in their URL or Body will be **automatically skipped**.
- **Dynamic Output Naming**: Automatically generates output filenames with timestamps if not specified.

## Installation

### Via Go Install

If you have Go installed, you can install the tool directly:

```bash
go install github.com/jonathanhecl/bru-ship@latest
```

### From Source

Ensure you have [Go](https://go.dev/) installed (1.18+ recommended).

```bash
git clone https://github.com/jonathanhecl/bru-ship.git
cd bru-ship
go build -o bru-ship
```

## Usage

Run the tool from your terminal. If no arguments are provided, it will display the help message.

```bash
./bru-ship [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-input` | Root directory of your Bruno collection. | `.` (Current Dir) |
| `-output` | Path for the generated Postman JSON file. If omitted, generates `[Folders]-[Timestamp].json`. | `collection.json` (or dynamic) |
| `-folders` | Comma-separated list of specific folders to include (e.g., `Auth,Users`). | (All folders) |
| `-replace` | Replace a variable in URLs/Bodies. Format: `key=value`. Can be repeated. | - |
| `-remove` | Remove a header or variable by key. Can be repeated. | - |
| `-env` | Name of the environment file to load variables from (e.g., `Production`). Looks in `environments/<name>.bru`. | - |
| `-verbose` | Enable verbose logging to see skipped endpoints and other details. | `false` |

### Examples

**1. Basic Conversion**
Convert the current directory's Bruno collection to `collection.json`.
```bash
./bru-ship
```

**2. Selective Export with Replacements**
Export only the `Core` and `Billing` folders, replace `{{baseUrl}}` with a staging URL, and remove the `AdminSecret` header.
```bash
./bru-ship -folders "Core,Billing" -replace "baseUrl=https://staging.api.com" -remove "AdminSecret" -env "Production"
```

**3. Custom Input and Output**
Convert a collection located in `../my-api` and save it as `export.json`.
```bash
./bru-ship -input "../my-api" -output "export.json"
```

## How it Works

1. **Scans** the input directory recursively.
2. **Parses** `.bru` files using a custom parser (handling blocks like `meta`, `headers`, `body`, `vars`).
3. **Filters** content based on your `-folders` flag.
4. **Sanitizes** and **Replaces** variables in URLs and Bodies according to your configuration.
5. **Generates** a Postman v2.1 compatible JSON file.

## License

MIT