package lsp

import "encoding/json"

// JSONRPCMessage is the base envelope for LSP requests and responses.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *ResponseError   `json:"error,omitempty"`
}

// ResponseError defines JSON-RPC error details.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// InitializeParams received on initialize request.
type InitializeParams struct {
	ProcessID int    `json:"processId"`
	RootURI   string `json:"rootUri"`
	RootPath  string `json:"rootPath"`
}

// InitializeResult returned to client on initialize.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

// ServerInfo identifies the codekernel engine to the IDE.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities declares supported LSP features.
type ServerCapabilities struct {
	DefinitionProvider      bool `json:"definitionProvider"`
	ReferencesProvider      bool `json:"referencesProvider"`
	WorkspaceSymbolProvider bool `json:"workspaceSymbolProvider"`
}

// Position in a text document (0-indexed).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location inside a resource.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// SymbolInformation represents a symbol in workspace/symbol.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// WorkspaceSymbolParams for workspace/symbol requests.
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

// TextDocumentPositionParams identifies a text document location.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// TextDocumentIdentifier identifies a document URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// PackContextParams for custom AI agent context packing over LSP.
type PackContextParams struct {
	Symbol string `json:"symbol"`
	Budget int    `json:"budget"`
}

// PackContextResult returned for custom context packing.
type PackContextResult struct {
	Symbol     string `json:"symbol"`
	Budget     int    `json:"budget"`
	UsedTokens int    `json:"used_tokens"`
	Markdown   string `json:"markdown"`
	Data       any    `json:"data"`
}
