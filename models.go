package main

// BruFile represents the parsed content of a .bru file
type BruFile struct {
	Name    string
	Type    string // http, graphql
	Url     string
	Method  string
	Headers []KeyValue
	Body    string
	Vars    []KeyValue
	Docs    string
}

type KeyValue struct {
	Key   string
	Value string
    Enabled bool
}

// PostmanCollection represents the root of the JSON
type PostmanCollection struct {
	Info Info   `json:"info"`
	Item []Item `json:"item"`
}

type Info struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"` // Use: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
}

// Item can be a Folder or a Request (recursive)
type Item struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Item        []Item        `json:"item,omitempty"`    // If it's a folder
	Request     *Request      `json:"request,omitempty"` // If it's an endpoint
	Response    []interface{} `json:"response,omitempty"` // Examples
}

type Request struct {
	Method string   `json:"method"`
	Header []Header `json:"header"`
	Body   *Body    `json:"body,omitempty"`
	Url    Url      `json:"url"`
}

type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type Body struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw,omitempty"`
}

// Url in Postman can be a string or an object, object is better for variables
type Url struct {
	Raw      string     `json:"raw"`
	Protocol string     `json:"protocol,omitempty"`
	Host     []string   `json:"host,omitempty"`
	Path     []string   `json:"path,omitempty"`
	Variable []Variable `json:"variable,omitempty"` // Path params
}

type Variable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
