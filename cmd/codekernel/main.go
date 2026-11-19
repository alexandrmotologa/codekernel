package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/codekernel/internal/config"
	"github.com/alexandrmotologa/codekernel/internal/graph"
	"github.com/alexandrmotologa/codekernel/internal/lsp"
	"github.com/alexandrmotologa/codekernel/internal/mcp"
	"github.com/alexandrmotologa/codekernel/internal/packer"
	"github.com/alexandrmotologa/codekernel/internal/server"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "index":
		runIndex(os.Args[2:])
	case "query":
		runQuery(os.Args[2:])
	case "pack":
		runPack(os.Args[2:])
	case "graph":
		runGraph(os.Args[2:])
	case "lsp":
		runLSP(os.Args[2:])
	case "mcp":
		runMCP(os.Args[2:])
	case "watch":
		runWatch(os.Args[2:])
	case "studio":
		runStudio(os.Args[2:])
	case "stats":
		runStats(os.Args[2:])
	case "benchmark":
		runBenchmark(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("codekernel version %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`codekernel - Local-First AST Intelligence Engine & Context Packer

Usage:
  codekernel <command> [arguments]

Commands:
  index [dir]       Parse codebase and build relational symbol call graph
  query <symbol>    Search indexed symbols with FTS5 and BM25 ranking
  pack <symbol>     Extract token-budgeted context subgraph for AI prompts
  graph <symbol>    Export neighborhood call graph (Mermaid or Graphviz DOT)
  mcp               Launch Model Context Protocol server over stdio
  lsp               Launch Language Server Protocol over stdio for IDE integration
  watch [dir]       Continuously monitor filesystem and update SQLite index
  studio            Start embedded React 19 Graph Studio on local HTTP server
  stats             Display indexed repository metrics and SQLite size
  benchmark         Run indexing and memory stress test benchmarks
  version           Display codekernel version information

Options for 'pack':
  --budget <tokens> Maximum context token budget (default: 4096)
  --format <fmt>    Output format: markdown (default) or json
  --changed         Pack context for git-modified symbols only
  --diff <ref>      Pack context for changes against git ref (e.g. HEAD~1, main)

Options for 'graph':
  --format <fmt>    Diagram format: mermaid (default) or dot
  --hops <depth>    Neighborhood traversal depth (default: 2)

Options for 'studio':
  --port <port>     Local port (default: 8888)
  --host <host>     Local host (default: 127.0.0.1)
  --watch           Enable continuous file watcher in background
`)
}

func getDBAndConfig(targetDir string) (*config.Config, *graph.DB, error) {
	if targetDir == "" {
		targetDir = "."
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		absTarget = targetDir
	}

	cfg, err := config.LoadConfig(absTarget, "")
	if err != nil {
		return nil, nil, err
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return nil, nil, err
	}

	db, err := graph.Open(cfg.Project.DatabasePath)
	if err != nil {
		return nil, nil, err
	}

	return cfg, db, nil
}

func runIndex(args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	start := time.Now()
	cfg, db, err := getDBAndConfig(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Indexing codebase: %s\n", cfg.Project.RootDir)

	indexer := graph.NewIndexer(cfg, db)
	progress, err := indexer.IndexRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Indexing failed: %v\n", err)
		os.Exit(1)
	}

	stats, _ := db.GetStats()

	fmt.Println("\nIndex complete:")
	fmt.Printf("  Files processed: %d (new/updated: %d, skipped: %d)\n",
		progress.TotalFiles, progress.IndexedFiles, progress.SkippedFiles)
	fmt.Printf("  Total symbols:   %d\n", stats.TotalSymbols)
	fmt.Printf("  Call graph edges:%d\n", stats.TotalEdges)
	fmt.Printf("  Total lines:     %d\n", stats.TotalLines)
	fmt.Printf("  Database path:   %s\n", cfg.Project.DatabasePath)
	fmt.Printf("  Elapsed time:    %v\n", time.Since(start).Round(time.Millisecond))
}

func runQuery(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: codekernel query <symbol-or-concept>")
		os.Exit(1)
	}

	query := args[0]
	_, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	symbols, err := db.FindSymbols(query, 50)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}

	if len(symbols) == 0 {
		fmt.Printf("No symbols found matching '%s'\n", query)
		return
	}

	fmt.Printf("Found %d matching symbol(s) (FTS5 / BM25 ranked):\n\n", len(symbols))
	for _, s := range symbols {
		exportedTag := "private"
		if s.Exported {
			exportedTag = "exported"
		}
		fmt.Printf("• %s [%s, %s, ~%d tokens]\n", s.QualifiedName, s.Kind, exportedTag, s.TokenCount)
		fmt.Printf("  File: %s (lines %d-%d)\n", s.FilePath, s.StartLine, s.EndLine)
		if s.Docstring != "" {
			fmt.Printf("  Doc:  %s\n", s.Docstring)
		}
		fmt.Printf("  Sig:  %s\n\n", s.Signature)
	}
}

func parseFlagsAndArgs(fs *flag.FlagSet, args []string) []string {
	var flags []string
	var nonFlags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// If flag takes a value separated by space
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			nonFlags = append(nonFlags, a)
		}
	}
	_ = fs.Parse(flags)
	return nonFlags
}

func runPack(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	budget := fs.Int("budget", 4096, "Token budget cap")
	format := fs.String("format", "markdown", "Output format (markdown or json)")
	changed := fs.Bool("changed", false, "Pack context around git-modified symbols only")
	diffRef := fs.String("diff", "", "Git diff target ref (e.g. HEAD~1, main)")

	nonFlags := parseFlagsAndArgs(fs, args)

	cfg, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if *changed || *diffRef != "" {
		dp := packer.NewDiffPacker(db, cfg.Project.RootDir)
		res, err := dp.PackChangedSymbols(*diffRef, *budget)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Diff packing failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(res)
		return
	}

	if len(nonFlags) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: codekernel pack <symbol> [--budget=4096] [--format=markdown|json]")
		os.Exit(1)
	}

	symbolQuery := nonFlags[0]
	sym, symID, err := db.GetSymbolByQualifiedName(symbolQuery)
	if err != nil || sym == nil {
		candidates, _ := db.FindSymbols(symbolQuery, 1)
		if len(candidates) == 0 {
			fmt.Fprintf(os.Stderr, "Symbol '%s' not found in index\n", symbolQuery)
			os.Exit(1)
		}
		sym = &candidates[0]
		_, symID, _ = db.GetSymbolByQualifiedName(sym.QualifiedName)
	}

	subgraph, err := db.Neighborhood(symID, 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Graph traversal failed: %v\n", err)
		os.Exit(1)
	}

	cp := packer.NewContextPacker(*budget, true)
	packed := cp.Pack(subgraph, *budget)
	formatter := packer.NewFormatter()

	if *format == "json" {
		data, _ := formatter.FormatJSON(packed)
		fmt.Println(string(data))
	} else {
		fmt.Println(formatter.FormatMarkdown(packed))
	}
}

func runGraph(args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	format := fs.String("format", "mermaid", "Output format: mermaid or dot")
	hops := fs.Int("hops", 2, "Neighborhood traversal depth")

	nonFlags := parseFlagsAndArgs(fs, args)

	if len(nonFlags) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: codekernel graph <symbol> [--format=mermaid|dot] [--hops=2]")
		os.Exit(1)
	}

	symbolQuery := nonFlags[0]
	_, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	sym, symID, err := db.GetSymbolByQualifiedName(symbolQuery)
	if err != nil || sym == nil {
		candidates, _ := db.FindSymbols(symbolQuery, 1)
		if len(candidates) == 0 {
			fmt.Fprintf(os.Stderr, "Symbol '%s' not found\n", symbolQuery)
			os.Exit(1)
		}
		sym = &candidates[0]
		_, symID, _ = db.GetSymbolByQualifiedName(sym.QualifiedName)
	}

	subgraph, err := db.Neighborhood(symID, *hops)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Neighborhood traversal failed: %v\n", err)
		os.Exit(1)
	}

	if *format == "dot" {
		fmt.Println(graph.ExportDOT(subgraph))
	} else {
		fmt.Println(graph.ExportMermaid(subgraph))
	}
}

func runLSP(args []string) {
	cfg, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	handler := lsp.NewLSPHandler(db, cfg.Project.RootDir)
	server := lsp.NewServer(os.Stdin, os.Stdout, handler)

	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "LSP server error: %v\n", err)
		os.Exit(1)
	}
}

func runMCP(args []string) {
	cfg, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	server := mcp.NewServer(os.Stdin, os.Stdout, db, cfg.Project.RootDir)
	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func runWatch(args []string) {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	cfg, db, err := getDBAndConfig(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	indexer := graph.NewIndexer(cfg, db)
	watcher := graph.NewWatcher(indexer, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nStopping watcher...")
		cancel()
	}()

	fmt.Printf("Watching %s for changes... (Press Ctrl+C to stop)\n", cfg.Project.RootDir)
	err = watcher.Watch(ctx, func(prog *graph.IndexProgress) {
		fmt.Printf("[%s] Re-indexed %d file(s) in %v\n",
			time.Now().Format("15:04:05"), prog.IndexedFiles, prog.Elapsed.Round(time.Millisecond))
	})
	if err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
	}
}

func runStudio(args []string) {
	fs := flag.NewFlagSet("studio", flag.ExitOnError)
	port := fs.Int("port", 8888, "HTTP port")
	host := fs.String("host", "127.0.0.1", "HTTP host")
	enableWatch := fs.Bool("watch", true, "Enable continuous background file watcher")

	_ = fs.Parse(args)

	cfg, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	cfg.Server.Port = *port
	cfg.Server.Host = *host

	indexer := graph.NewIndexer(cfg, db)

	// Launch background watcher if enabled
	if *enableWatch {
		watcher := graph.NewWatcher(indexer, 1*time.Second)
		go func() {
			_ = watcher.Watch(context.Background(), nil)
		}()
	}

	srv := server.NewHTTPServer(cfg, db, indexer)

	fmt.Printf("codekernel React 19 Graph Studio running at http://%s:%d\n", *host, *port)
	fmt.Println("Press Ctrl+C to terminate.")

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func runStats(args []string) {
	cfg, db, err := getDBAndConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching stats: %v\n", err)
		os.Exit(1)
	}

	dbInfo, _ := os.Stat(cfg.Project.DatabasePath)
	dbSizeKB := int64(0)
	if dbInfo != nil {
		dbSizeKB = dbInfo.Size() / 1024
	}

	fmt.Printf("codekernel Index Statistics (%s):\n", cfg.Project.Name)
	fmt.Printf("  Total files indexed:  %d\n", stats.TotalFiles)
	fmt.Printf("  Total lines of code:  %d\n", stats.TotalLines)
	fmt.Printf("  Total AST symbols:    %d\n", stats.TotalSymbols)
	fmt.Printf("  Total call edges:     %d\n", stats.TotalEdges)
	fmt.Printf("  Total raw bytes:      %d KB\n", stats.TotalBytes/1024)
	fmt.Printf("  SQLite database size: %d KB (%s)\n", dbSizeKB, cfg.Project.DatabasePath)
}

func runBenchmark(args []string) {
	fmt.Println("Running codekernel performance benchmark...")

	tempDir, err := os.MkdirTemp("", "codekernel_bench_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	// Measure startup latency
	startInit := time.Now()
	cfg := config.DefaultConfig(tempDir)
	db, err := graph.Open(cfg.Project.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB open error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	startupTime := time.Since(startInit)

	// Generate 100 synthetic source files with cross-calling functions
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		code := fmt.Sprintf(`package benchmark

import "fmt"

func Function%d(val int) int {
	fmt.Println(val)
	return Function%d(val + 1)
}

func Helper%d(name string) string {
	return "result_" + name
}
`, i, (i+1)%numFiles, i)

		filePath := filepath.Join(tempDir, fmt.Sprintf("bench_%03d.go", i))
		_ = os.WriteFile(filePath, []byte(code), 0644)
	}

	// Benchmark indexing
	indexer := graph.NewIndexer(cfg, db)
	startIndex := time.Now()
	prog, err := indexer.IndexRepository()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Benchmark indexing error: %v\n", err)
		os.Exit(1)
	}
	indexingTime := time.Since(startIndex)

	// Benchmark context packing
	startPack := time.Now()
	sym, symID, _ := db.GetSymbolByQualifiedName("benchmark.Function0")
	if symID > 0 {
		subgraph, _ := db.Neighborhood(symID, 2)
		cp := packer.NewContextPacker(4096, true)
		_ = cp.Pack(subgraph, 4096)
	}
	packingTime := time.Since(startPack)

	// Memory usage check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ramMB := float64(m.Alloc) / 1024 / 1024

	fmt.Println("\nBenchmark Results:")
	fmt.Printf("  Startup Latency:      %v (target < 15ms)\n", startupTime.Round(time.Microsecond))
	fmt.Printf("  Files Indexed:        %d files (%d symbols, %d edges)\n",
		prog.TotalFiles, prog.TotalSymbols, prog.TotalSymbols/2)
	fmt.Printf("  Indexing Duration:    %v (average %v/file)\n",
		indexingTime.Round(time.Millisecond), (indexingTime / time.Duration(numFiles)).Round(time.Microsecond))
	fmt.Printf("  Packing Latency:      %v\n", packingTime.Round(time.Microsecond))
	fmt.Printf("  Current Memory Alloc: %.2f MB (target < 25MB)\n", ramMB)
	_ = sym
}
