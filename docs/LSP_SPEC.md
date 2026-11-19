# Language Server Protocol and MCP specification

`codekernel` provides standard protocol interfaces for editors and autonomous coding agents.

## Language Server Protocol (LSP 3.17)

The server communicates over standard input and output with standard `Content-Length: ...\r\n\r\n` JSON-RPC 2.0 framing.

### Standard endpoints

- `initialize`: Declares server capabilities, including `definitionProvider`, `referencesProvider`, and `workspaceSymbolProvider`.
- `workspace/symbol`: Queries symbols across all indexed files with fuzzy matching.
- `textDocument/definition`: Returns definition location for symbol at given coordinates.
- `textDocument/references`: Returns caller locations invoking the target symbol.

### Agent extension endpoints

- `codekernel/packContext`:
  Extracts packed context subgraph for agent prompt injection.
  - Parameters: `{ "symbol": string, "budget": number }`
  - Returns: `{ "symbol": string, "used_tokens": number, "markdown": string, "data": object }`

- `codekernel/callGraph`:
  Extracts neighborhood subgraph up to configured hops.
  - Parameters: `{ "symbol": string, "hops": number }`
  - Returns: `{ "nodes": array, "edges": array }`

## Model Context Protocol (MCP)

Runs over stdio using newline-delimited JSON-RPC messages.

### Available tools

1. `search_symbols`
   - Description: Search indexed codebase symbols, types, and functions using FTS5 and BM25 ranking.
   - Arguments: `query` (string, required), `limit` (integer, optional)

2. `pack_context`
   - Description: Extract AST-aware surgical prompt context subgraph for a target symbol within an exact token budget.
   - Arguments: `symbol` (string, required), `budget` (integer, optional)

3. `get_call_graph`
   - Description: Extract call graph neighborhood (incoming callers and outgoing dependencies) for a symbol.
   - Arguments: `symbol` (string, required), `hops` (integer, optional)

4. `get_file_outline`
   - Description: Retrieve AST outline of symbols, classes, and methods declared in a source file.
   - Arguments: `path` (string, required)
