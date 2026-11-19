# codekernel: Architecture and Engineering Master Plan
> Local-First Codebase AST Intelligence Engine, Semantic Call Graph & Context Packing Core for AI Coding Agents
> Target: Zero-CGO Single-Binary Go Engine (< 25MB RAM, < 15ms startup) with Embedded React 19 Graph Studio & LSP Bridge

---

## 1. Problem Statement and Motivation

### 1.1 The Privacy and Structural Limitations of Cloud-Based Code Intelligence
Modern AI coding assistants (such as Cursor, Windsurf, GitHub Copilot, and Sourcegraph Cody) rely heavily on repository-level indexing (`@codebase`) to provide agents with relevant project context. However, current solutions present severe enterprise barriers:
1. **The Cloud Exfiltration Mandate**: Commercial tools upload source code, AST metadata, and proprietary symbols to remote third-party cloud servers. Financial institutions, defense contractors, healthcare organizations, and high-security enterprises strictly prohibit source code egress, rendering cloud-dependent tools non-compliant.
2. **The Inadequacy of Syntactic Grep (`ripgrep`)**: Text-based search engines (`grep`, `ripgrep`, `ag`) find string occurrences but lack all structural understanding of code: they cannot identify which interface an object implements, which functions call a specific method, or how type definitions propagate across module boundaries.
3. **The Failure of Standard Vector RAG on Code**: Naive vector chunkers split documents at fixed token counts (e.g. every 500 tokens). In source code, this arbitrary slicing cuts functions in half, strips parameter signatures from their bodies, destroys indentation scoping, and introduces fatal syntax errors into LLM context prompts.
4. **Context Window Token Bloat**: When an AI agent needs to modify a service, passing entire 2,000-line files consumes massive token budgets and degrades model reasoning attention. Agents require surgically packed context subgraphs containing only the relevant function definitions and their immediate dependencies.

### 1.2 The codekernel Solution
`codekernel` is an ultra-fast, local-first codebase intelligence engine written in Go. Running entirely on local CPU with zero data transmission:
- **Tree-sitter Multi-Language AST Parsing**: Scans million-line codebases in seconds across Go, TypeScript, JavaScript, Python, Rust, and Java, extracting abstract syntax trees without invoking language runtimes.
- **Relational Symbol & Call-Graph Engine**: Maps all functions, methods, structs, interfaces, imports, and cross-file caller/callee relationships into an embedded pure Go SQLite database (`.codekernel.db`).
- **AST-Aware Semantic Chunking**: Enforces strict syntactic boundaries: code chunks are anchored to complete functions, classes, or interface declarations with preserved docstrings and type annotations.
- **Token-Budgeted Context Packer**: Given an agent inquiry (e.g., *"How is payment authorization verified in the Stripe webhook?"*), performs a multi-hop traversal over the call graph, extracting the minimal essential subgraph and packing it cleanly within configured token limits.
- **Universal Local IDE & Agent Integration**: Connects seamlessly to VS Code, Neovim, and autonomous CLI agents (Claude Code, Aider, custom scripts) via Language Server Protocol (LSP) and stdio JSON-RPC.
- **Embedded React 19 Graph Studio**: Provides developers with an interactive in-browser 2D/3D call graph visualizer, symbol hierarchy explorer, and live prompt context sandbox.

---

## 2. High-Level System Architecture

```
       +-------------------------------------------------------------+
       |               AI Coding Agent / Local IDE / CLI             |
       |             (Claude Code, Cursor, VS Code, Neovim)          |
       +-------------------------------------------------------------+
                                      |
                     LSP / JSON-RPC / CLI Inquiry
                                      v
       +-------------------------------------------------------------+
       |                      codekernel Engine                      |
       |                    (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | Tree-sitter Parser  |------->| Symbol & AST Extractor |  |
       |  | (Go/TS/Py/Rust/Java)|        | (Funcs, Structs, Calls)|  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |           Relational Call-Graph Engine (SQLite)       |  |
       |  |  - In-Memory Adjacency: Caller <-> Callee             |  |
       |  |  - Interface Implementation & Type Propagation       |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |             AST-Aware Semantic Chunker                |  |
       |  |  Zero Arbitrary Token Splits; Preserves Full Scopes   |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |          Token-Budgeted Context Subgraph Packer       |  |
       |  |  Packs Target Node + 1-Hop Callers into 4k Budget     |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                 Surgically Packed Code Context              |
       |  1. Injected Prompt Subgraph (< 4k Tokens) to Agent LLM     |
       |  2. 100% Offline Local Privacy (Zero Cloud Egress)          |
       |  3. Embedded React 19 Graph Studio (Local HTTP :8888)       |
       +-------------------------------------------------------------+
```

---

## 3. Mathematical Formulation: Call-Graph Traversal & Context Budgeting

### 3.1 Call-Graph Adjacency Matrix
Let $G = (V, E)$ be the directed codebase graph where vertices $V$ represent code symbols (functions, types) and edges $E$ represent invocations:

$$A_{ij} = \begin{cases} 
1, & \text{if symbol } v_i \text{ directly invokes or references } v_j \\
0, & \text{otherwise}
\end{cases}$$

### 3.2 Knapsack Context Packing Objective
Given target query entrypoint $v_0$, let $\text{Dist}(v_0, v_i)$ denote the shortest path distance in $G$, $T(v_i)$ the token size of symbol $v_i$, and $B$ the maximum context token budget (e.g., 4,096 tokens):

$$\max \sum_{v_i \in V} \frac{1}{\text{Dist}(v_0, v_i) + 1} \cdot x_i, \quad \text{subject to } \sum_{v_i \in V} T(v_i) \cdot x_i \le B, \quad x_i \in \{0, 1\}$$

The solver prioritizes the immediate functional definition, followed by input parameter types and direct callers, maximizing semantic relevance within strict prompt boundaries.

---

## 4. Repository Structure

```
codekernel/
├── cmd/
│   └── codekernel/
│       └── main.go                 # CLI entry point, subcommands: index, query, lsp, studio
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration for languages, ignore patterns, token caps
│   │   └── schema.go               # Language parser mappings and context packing budgets
│   ├── parser/
│   │   ├── treesitter.go           # High-speed Tree-sitter AST binding manager
│   │   ├── lang_go.go              # Go AST extractor: packages, structs, functions, methods
│   │   ├── lang_typescript.go      # TypeScript/JavaScript AST extractor: interfaces, classes
│   │   ├── lang_python.go          # Python AST extractor: classes, defs, decorators
│   │   └── lang_rust.go            # Rust AST extractor: traits, impls, structs, fn
│   ├── graph/
│   │   ├── db.go                   # Pure Go SQLite store (modernc.org/sqlite) for symbols & edges
│   │   ├── symbols.go              # Symbol indexer extracting qualified names and line spans
│   │   ├── calls.go                # Cross-file caller/callee edge resolution
│   │   └── traversal.go            # Multi-hop neighborhood search algorithm
│   ├── chunker/
│   │   ├── ast_chunk.go            # Preserves structural AST boundaries (zero line truncation)
│   │   └── signatures.go           # Compresses full function bodies into typed signature stubs
│   ├── packer/
│   │   ├── budget.go               # Greedy knapsack context packer satisfying token limits
│   │   └── formatter.go            # Formats packed subgraphs into markdown code context
│   ├── lsp/
│   │   ├── server.go               # Language Server Protocol stdio handler
│   │   └── handlers.go             # textDocument/definition and workspace/symbol endpoints
│   └── server/
│       ├── api.go                  # Management REST API for studio graph queries
│       ├── embed.go                # Embedded React 19 studio static assets via //go:embed
│       └── sse.go                  # Live indexing progress stream
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── CallGraph3D.tsx     # WebGL 3D force-directed call graph visualizer
│   │   │   ├── SymbolHierarchy.tsx # Tree view of project symbols, types, and files
│   │   │   ├── ContextPackerTest.tsx # Interactive sandbox to test prompt context generation
│   │   │   ├── TokenBudgetGauge.tsx# Token counter dial comparing raw file size vs. packed size
│   │   │   └── FileDetailPane.tsx  # Syntax-highlighted code viewer with clickable AST nodes
│   │   └── pages/
│   │       └── Dashboard.tsx       # Main Codebase Intelligence console
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── codekernel.yaml             # Standard project configuration sample
│   └── sample_project/             # Multi-file Go and TypeScript project for indexing benchmarks
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Implementation Roadmap

### Phase 1: Go Engine Foundation and Tree-sitter Parser
- Initialize Go module with minimal dependencies.
- Build Tree-sitter binding engine in `internal/parser/treesitter.go`.
- Implement AST extractors for Go, TypeScript, and Python extracting function declarations and import statements.

### Phase 2: Relational Symbol Indexer & SQLite Graph
- Implement pure Go SQLite store in `internal/graph/db.go` with schema for symbols, files, and call edges.
- Build symbol resolver matching function calls to target definitions across files.

### Phase 3: AST-Aware Chunker and Signature Compressor
- Build chunker ensuring code blocks are never severed across function or class boundaries.
- Implement signature compressor extracting typed headers and docstrings for 2-hop dependencies.

### Phase 4: Token-Budgeted Context Packer
- Implement greedy knapsack packing algorithm in `internal/packer/budget.go`.
- Output clean, structured markdown context optimized for LLM attention spans.

### Phase 5: Language Server Protocol (LSP) Bridge
- Implement standard LSP stdio transport supporting `workspace/symbol` and definition lookups.
- Enable direct mounting inside VS Code, Neovim, and CLI agent harnesses.

### Phase 6: Embedded React 19 Graph Studio
- Develop modern web UI in `web/` featuring 3D force-directed call graph visualizer, symbol tree, and interactive context packing tester.
- Embed frontend build artifacts into the Go binary via `//go:embed` and serve locally on `:8888`.

### Phase 7: Verification, Benchmarking, and Documentation
- Run indexing benchmark on a 500,000-line repository verifying full indexing completes in $< 10\text{ seconds}$.
- Verify memory consumption remains strictly under 25MB RAM.
- Author humanized README with setup guides, IDE integration tutorials, and private enterprise compliance playbooks.
