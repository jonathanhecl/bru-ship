package main

// BruFile represents the parsed content of a .bru file
type BruFile struct {
	Name     string
	Type     string // http, graphql
	Url      string
	Method   string
	Headers  []KeyValue
	Body     string
	Vars     []KeyValue
	Docs     string
	Auth     map[string]string
	Examples []BruExample
}

type BruExample struct {
	Name     string
	Request  BruRequest
	Response BruResponse
}

type BruRequest struct {
	Method  string
	Url     string
	Headers []KeyValue
	Body    string
}

type BruResponse struct {
	Status     int
	StatusText string
	Headers    []KeyValue
	Body       string
}

type KeyValue struct {
	Key     string
	Value   string
	Enabled bool
}

// PostmanCollection represents the root of the JSON
type PostmanCollection struct {
	Info     Info       `json:"info"`
	Item     []Item     `json:"item"`
	Variable []Variable `json:"variable,omitempty"`
}

type Info struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"` // Use: "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
}

// Item can be a Folder or a Request (recursive)
type Item struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Item        []Item            `json:"item,omitempty"`     // If it's a folder
	Request     *Request          `json:"request,omitempty"`  // If it's an endpoint
	Response    []PostmanResponse `json:"response,omitempty"` // Examples
	Variable    []Variable        `json:"variable,omitempty"` // Folder variables
}

type PostmanResponse struct {
	Name                   string        `json:"name"`
	OriginalRequest        *Request      `json:"originalRequest"`
	Status                 string        `json:"status"`
	Code                   int           `json:"code"`
	PostmanPreviewLanguage string        `json:"_postman_previewlanguage"`
	Header                 []Header      `json:"header"`
	Cookie                 []interface{} `json:"cookie"`
	Body                   string        `json:"body"`
}

type Request struct {
	Method      string       `json:"method"`
	Header      []Header     `json:"header"`
	Body        *Body        `json:"body,omitempty"`
	Url         Url          `json:"url"`
	Description string       `json:"description,omitempty"`
	Auth        *PostmanAuth `json:"auth,omitempty"`
}

type PostmanAuth struct {
	Type   string        `json:"type"`
	Bearer []AuthElement `json:"bearer,omitempty"`
	Basic  []AuthElement `json:"basic,omitempty"`
	// Add other types as needed
}

type AuthElement struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type Header struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type Body struct {
	Mode    string                 `json:"mode"`
	Raw     string                 `json:"raw,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
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

type BrunoConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
