# Architecture

`codekernel` is a local-first codebase intelligence engine and context packing core written in Go. It operates on developer hardware without sending source code or symbol metadata to external networks.

## Core subsystems

The engine is composed of five decoupled subsystems:

1. **Tree-sitter Parser Manager (`internal/parser`)**:
   Loads multi-language grammars (Go, TypeScript, JavaScript, Python, Rust, Java) via tree-sitter C bindings. Files are parsed into abstract syntax trees concurrently using goroutine worker pools. Declarations, calls, and imports are extracted without running language interpreters or compilers.

2. **Relational Graph Store (`internal/graph`)**:
   Maintains symbols and call edges in an embedded SQLite database (`modernc.org/sqlite`). This driver is written in pure Go, requiring zero CGO at runtime. The database uses Write-Ahead Logging (WAL) and creates full-text search indices via SQLite FTS5 for BM25 ranking.

3. **AST-Aware Semantic Chunker (`internal/chunker`)**:
   Partitions code along syntactic AST boundaries. Functions, classes, and types are preserved as whole units. For remote dependencies in the graph, a signature compressor strips function bodies while retaining docstrings, argument types, and return values.

4. **Greedy Knapsack Context Packer (`internal/packer`)**:
   Given an entry symbol and a token budget, traverses the call graph outward up to $N$ hops. It ranks candidates by harmonic distance weight $1 / (\text{dist} + 1)$ and in-degree centrality, packing definitions into markdown code context optimized for prompt injection.

5. **Interface Bridges (`internal/lsp`, `internal/mcp`, `internal/server`)**:
   Connects to editors and AI agents through three standard interfaces:
   - Language Server Protocol (LSP 3.17 subset) over stdio.
   - Model Context Protocol (MCP) over stdio for agents like Claude Code and Cursor.
   - HTTP server serving REST endpoints and the embedded React 19 Graph Studio.

## Data flow

```
[Source Files]
      │
      ▼
[Tree-sitter AST Extraction]
      │
      ▼
[SQLite Storage: files, symbols, edges, FTS5]
      │
      ▼
[Call-Graph Resolution (caller -> callee)]
      │
      ▼
[N-Hop Neighborhood Traversal + Centrality Scoring]
      │
      ▼
[Greedy Knapsack Packing within Token Budget]
      │
      ├───────────────────────┼──────────────────────┐
      ▼                       ▼                      ▼
 [CLI Output]            [LSP / MCP]            [React 19 Studio]
```

## Concurrency and persistence

- File indexing executes across CPU cores using a worker pool.
- Files are hashed with SHA256. Unmodified files are skipped on subsequent index runs.
- Database writes are batched in transactions to keep indexing times under 300 milliseconds on standard repositories.
