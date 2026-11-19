package parser

// SymbolKind represents the type of a code symbol.
type SymbolKind string

const (
	KindFunction  SymbolKind = "function"
	KindMethod    SymbolKind = "method"
	KindStruct    SymbolKind = "struct"
	KindInterface SymbolKind = "interface"
	KindClass     SymbolKind = "class"
	KindTrait     SymbolKind = "trait"
	KindEnum      SymbolKind = "enum"
	KindTypeAlias SymbolKind = "type_alias"
	KindVariable  SymbolKind = "variable"
	KindModule    SymbolKind = "module"
)

// Symbol represents an extracted code declaration from the AST.
type Symbol struct {
	Name          string     `json:"name"`
	QualifiedName string     `json:"qualified_name"`
	Kind          SymbolKind `json:"kind"`
	Signature     string     `json:"signature"`
	Docstring     string     `json:"docstring"`
	FilePath      string     `json:"file_path"`
	StartLine     int        `json:"start_line"`
	EndLine       int        `json:"end_line"`
	StartByte     uint32     `json:"start_byte"`
	EndByte       uint32     `json:"end_byte"`
	Receiver      string     `json:"receiver,omitempty"`
	Exported      bool       `json:"exported"`
	Language      string     `json:"language"`
	Body          string     `json:"body"`
	TokenCount    int        `json:"token_count"`
}

// CallReference represents a function or method invocation.
type CallReference struct {
	CallerName     string `json:"caller_name"`
	CalleeName     string `json:"callee_name"`
	CalleeReceiver string `json:"callee_receiver,omitempty"`
	FilePath       string `json:"file_path"`
	Line           int    `json:"line"`
	ByteOffset     uint32 `json:"byte_offset"`
	IsMethodCall   bool   `json:"is_method_call"`
}

// ImportReference represents an imported package, module, or symbol.
type ImportReference struct {
	Path    string   `json:"path"`
	Alias   string   `json:"alias,omitempty"`
	Symbols []string `json:"symbols,omitempty"`
}

// ParsedFile contains all metadata, declarations, and call sites extracted from a source file.
type ParsedFile struct {
	FilePath string            `json:"file_path"`
	Language string            `json:"language"`
	Content  []byte            `json:"-"`
	Symbols  []Symbol          `json:"symbols"`
	Calls    []CallReference   `json:"calls"`
	Imports  []ImportReference `json:"imports"`
	LineCount int              `json:"line_count"`
	ByteSize int               `json:"byte_size"`
	Hash     string            `json:"hash"`
}
