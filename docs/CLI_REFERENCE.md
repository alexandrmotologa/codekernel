# CLI reference

`codekernel` provides a single command line executable for indexing, querying, and packing codebase intelligence.

## Global options

Run `codekernel help` or `codekernel --help` to view available commands.

## Commands

### `codekernel index [dir]`
Scans the target directory, extracts AST declarations and calls, and updates the local SQLite index.

```bash
# Index current directory
codekernel index

# Index specific project folder
codekernel index ./examples/sample_project
```

### `codekernel query <term>`
Searches indexed symbols using SQLite FTS5 full-text search with BM25 relevance ranking.

```bash
# Search for symbol or concept
codekernel query Authorize

# Search by docstring content
codekernel query "token verification"
```

### `codekernel pack <symbol> [options]`
Extracts an AST-aware context subgraph around a target symbol, packed strictly within the specified token budget.

```bash
# Pack context for a specific function with default 4096 budget
codekernel pack parser.Engine.Parse

# Set custom token budget and output format
codekernel pack parser.Engine.Parse --budget=2048 --format=json

# Pack context for all symbols modified in working directory
codekernel pack --changed

# Pack context for changes relative to git commit or branch
codekernel pack --diff=main --budget=8192
```

### `codekernel graph <symbol> [options]`
Generates architectural call graph diagrams for documentation and pull requests.

```bash
# Export Mermaid diagram (default)
codekernel graph parser.Engine.Parse --format=mermaid

# Export Graphviz DOT format
codekernel graph parser.Engine.Parse --format=dot --hops=2
```

### `codekernel mcp`
Launches the Model Context Protocol (MCP) server over standard input and output. Connects directly to Claude Code, Cursor, and other agent platforms.

```bash
codekernel mcp
```

### `codekernel lsp`
Runs the Language Server Protocol over stdio for IDE integration with VS Code, Neovim, and Sublime Text.

```bash
codekernel lsp
```

### `codekernel watch [dir]`
Runs continuous filesystem monitoring in the background. Incrementally re-indexes modified files on save.

```bash
codekernel watch
```

### `codekernel studio [options]`
Starts the embedded React 19 Graph Studio on a local HTTP port.

```bash
# Run web studio on default port 8888
codekernel studio

# Specify custom port
codekernel studio --port=9000
```

### `codekernel stats`
Displays indexing metrics, total line counts, symbol counts, and database storage size.

```bash
codekernel stats
```

### `codekernel benchmark`
Executes an automated performance test measuring startup latency, memory allocation, and indexing throughput.

```bash
codekernel benchmark
```
