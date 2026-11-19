package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration options for codekernel.
type Config struct {
	Project ProjectConfig `yaml:"project"`
	Parser  ParserConfig  `yaml:"parser"`
	Packing PackingConfig `yaml:"packing"`
	Server  ServerConfig  `yaml:"server"`
	LSP     LSPConfig     `yaml:"lsp"`
}

// ProjectConfig defines project-level settings.
type ProjectConfig struct {
	Name        string `yaml:"name"`
	RootDir     string `yaml:"root_dir"`
	StorageDir  string `yaml:"storage_dir"`
	DatabasePath string `yaml:"database_path"`
}

// ParserConfig defines language and traversal settings for AST extraction.
type ParserConfig struct {
	MaxFileSizeKB int      `yaml:"max_file_size_kb"`
	ExcludeDirs   []string `yaml:"exclude_dirs"`
	ExcludeFiles  []string `yaml:"exclude_files"`
	Languages     []string `yaml:"languages"`
}

// PackingConfig defines token budget and context generation settings.
type PackingConfig struct {
	DefaultBudget   int  `yaml:"default_budget"`
	MaxHops         int  `yaml:"max_hops"`
	IncludeDocstrings bool `yaml:"include_docstrings"`
	CompressCallees bool `yaml:"compress_callees"`
}

// ServerConfig defines the embedded Web Studio HTTP server options.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LSPConfig defines settings for the Language Server Protocol bridge.
type LSPConfig struct {
	LogFile string `yaml:"log_file"`
}

// DefaultConfig returns standard configurations tuned for local execution.
func DefaultConfig(rootDir string) *Config {
	if rootDir == "" {
		rootDir = "."
	}
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		absRoot = rootDir
	}

	storageDir := filepath.Join(absRoot, ".codekernel")
	dbPath := filepath.Join(storageDir, "index.db")

	return &Config{
		Project: ProjectConfig{
			Name:         filepath.Base(absRoot),
			RootDir:      absRoot,
			StorageDir:   storageDir,
			DatabasePath: dbPath,
		},
		Parser: ParserConfig{
			MaxFileSizeKB: 1024, // 1MB limit for code files
			ExcludeDirs: []string{
				".git",
				".codekernel",
				"node_modules",
				"vendor",
				"dist",
				"build",
				"bin",
				"out",
				"target",
				".venv",
				"venv",
				"__pycache__",
				".idea",
				".vscode",
			},
			ExcludeFiles: []string{
				"*.min.js",
				"*.min.css",
				"*.map",
				"*.lock",
				"package-lock.json",
				"pnpm-lock.yaml",
				"yarn.lock",
				"go.sum",
				"Cargo.lock",
			},
			Languages: []string{"go", "typescript", "javascript", "python", "rust", "java"},
		},
		Packing: PackingConfig{
			DefaultBudget:     4096,
			MaxHops:           2,
			IncludeDocstrings: true,
			CompressCallees:   true,
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8888,
		},
		LSP: LSPConfig{
			LogFile: filepath.Join(storageDir, "lsp.log"),
		},
	}
}

// LoadConfig reads a YAML configuration file or returns the default config if the file does not exist.
func LoadConfig(rootDir string, configPath string) (*Config, error) {
	cfg := DefaultConfig(rootDir)

	if configPath == "" {
		configPath = filepath.Join(cfg.Project.RootDir, "codekernel.yaml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// EnsureDirectories ensures that the storage directory exists.
func (c *Config) EnsureDirectories() error {
	return os.MkdirAll(c.Project.StorageDir, 0755)
}
