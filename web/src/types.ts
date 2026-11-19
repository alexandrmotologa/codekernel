export interface RepoStats {
  total_files: number;
  total_symbols: number;
  total_edges: number;
  total_lines: number;
  total_bytes: number;
}

export interface FileRecord {
  id: number;
  path: string;
  language: string;
  hash: string;
  byte_size: number;
  line_count: number;
  last_indexed_at: string;
}

export interface SymbolItem {
  name: string;
  qualified_name: string;
  kind: string;
  signature: string;
  docstring: string;
  file_path: string;
  start_line: number;
  end_line: number;
  start_byte: number;
  end_byte: number;
  receiver?: string;
  exported: boolean;
  language: string;
  body: string;
  token_count: number;
}

export interface GraphNode {
  id: number;
  name: string;
  qualified: string;
  kind: string;
  language: string;
  file_path: string;
  exported: boolean;
  tokens: number;
}

export interface GraphEdge {
  source: number;
  target: number;
  type: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface ChunkItem {
  file_path: string;
  symbol_name: string;
  kind: string;
  content: string;
  start_line: number;
  end_line: number;
  token_count: number;
}

export interface PackedResult {
  symbol: string;
  budget: number;
  used_tokens: number;
  markdown: string;
  packed: {
    entry_symbol: string;
    budget: number;
    used_tokens: number;
    target_chunk: ChunkItem;
    caller_chunks: ChunkItem[];
    callee_chunks: ChunkItem[];
    other_chunks: ChunkItem[];
    compressed_count: number;
    dropped_count: number;
  };
}

export interface IndexProgress {
  total_files: number;
  processed_files: number;
  skipped_files: number;
  indexed_files: number;
  total_symbols: number;
  current_file: string;
  elapsed: number;
  completed: boolean;
  error?: string;
}
