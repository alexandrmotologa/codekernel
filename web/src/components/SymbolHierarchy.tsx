import React, { useState } from 'react';
import { SymbolItem } from '../types';
import { Search, Folder, Code, Hash, Layers, ArrowUpRight, CheckCircle2 } from 'lucide-react';

interface SymbolHierarchyProps {
  symbols: SymbolItem[];
  onSelectSymbol: (symbolName: string) => void;
}

export const SymbolHierarchy: React.FC<SymbolHierarchyProps> = ({ symbols, onSelectSymbol }) => {
  const [search, setSearch] = useState('');
  const [selectedKind, setSelectedKind] = useState<string>('all');
  const [selectedLang, setSelectedLang] = useState<string>('all');
  const [activeSymbol, setActiveSymbol] = useState<SymbolItem | null>(null);

  // Filter symbols
  const filtered = symbols.filter((s) => {
    const matchesSearch =
      s.name.toLowerCase().includes(search.toLowerCase()) ||
      s.qualified_name.toLowerCase().includes(search.toLowerCase()) ||
      s.file_path.toLowerCase().includes(search.toLowerCase());
    const matchesKind = selectedKind === 'all' || s.kind === selectedKind;
    const matchesLang = selectedLang === 'all' || s.language === selectedLang;
    return matchesSearch && matchesKind && matchesLang;
  });

  // Group by file path
  const groupedByFile = filtered.reduce<Record<string, SymbolItem[]>>((acc, s) => {
    if (!acc[s.file_path]) {
      acc[s.file_path] = [];
    }
    acc[s.file_path].push(s);
    return acc;
  }, {});

  const allKinds = Array.from(new Set(symbols.map((s) => s.kind)));
  const allLangs = Array.from(new Set(symbols.map((s) => s.language)));

  return (
    <div className="flex w-full h-[calc(100vh-4rem)] bg-[#070a10]">
      {/* Left List Pane */}
      <div className="w-1/2 border-r border-slate-800 flex flex-col">
        {/* Filters Header */}
        <div className="p-4 border-b border-slate-800 space-y-3 bg-[#0d121f]">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filter by name, path, receiver..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500 font-mono"
            />
          </div>

          <div className="flex items-center gap-2">
            <select
              value={selectedKind}
              onChange={(e) => setSelectedKind(e.target.value)}
              className="px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-300 font-mono focus:outline-none"
            >
              <option value="all">All Kinds ({symbols.length})</option>
              {allKinds.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>

            <select
              value={selectedLang}
              onChange={(e) => setSelectedLang(e.target.value)}
              className="px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-300 font-mono focus:outline-none"
            >
              <option value="all">All Languages</option>
              {allLangs.map((l) => (
                <option key={l} value={l}>
                  {l}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Tree List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 font-mono text-xs">
          {Object.entries(groupedByFile).map(([filePath, fileSymbols]) => (
            <div key={filePath} className="bg-slate-900/60 rounded-xl border border-slate-800 p-3">
              <div className="flex items-center gap-2 text-slate-400 font-medium pb-2 mb-2 border-b border-slate-800/80">
                <Folder className="w-3.5 h-3.5 text-indigo-400" />
                <span className="truncate">{filePath}</span>
                <span className="text-[10px] text-slate-500 ml-auto">
                  {fileSymbols.length} symbols
                </span>
              </div>

              <div className="space-y-1">
                {fileSymbols.map((sym) => (
                  <button
                    key={sym.qualified_name + sym.start_line}
                    onClick={() => setActiveSymbol(sym)}
                    className={`w-full text-left px-2.5 py-1.5 rounded-lg flex items-center justify-between transition-colors ${
                      activeSymbol?.qualified_name === sym.qualified_name
                        ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30'
                        : 'text-slate-300 hover:bg-slate-800/60'
                    }`}
                  >
                    <div className="flex items-center gap-2 truncate">
                      <span className="text-[10px] uppercase px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
                        {sym.kind}
                      </span>
                      <span className="truncate">{sym.name}</span>
                    </div>
                    <span className="text-[10px] text-slate-500">L{sym.start_line}</span>
                  </button>
                ))}
              </div>
            </div>
          ))}

          {filtered.length === 0 && (
            <div className="text-center py-12 text-slate-500">No matching symbols found.</div>
          )}
        </div>
      </div>

      {/* Right Symbol Details */}
      <div className="w-1/2 p-6 flex flex-col justify-between overflow-y-auto">
        {activeSymbol ? (
          <div className="space-y-6">
            <div>
              <div className="flex items-center gap-2">
                <span className="text-xs uppercase font-mono px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-400 border border-indigo-500/30">
                  {activeSymbol.kind}
                </span>
                <span className="text-xs font-mono text-slate-400">{activeSymbol.language}</span>
                {activeSymbol.exported && (
                  <span className="text-xs font-mono text-emerald-400 flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> exported
                  </span>
                )}
              </div>
              <h2 className="text-xl font-bold font-mono text-white mt-2">
                {activeSymbol.qualified_name}
              </h2>
              <p className="text-xs font-mono text-slate-400 mt-1">{activeSymbol.file_path}</p>
            </div>

            {activeSymbol.docstring && (
              <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-4">
                <span className="text-[11px] font-mono text-slate-500 uppercase block mb-1">
                  Docstring / Comments
                </span>
                <p className="text-xs text-slate-300 font-mono whitespace-pre-wrap">
                  {activeSymbol.docstring}
                </p>
              </div>
            )}

            <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-4">
              <span className="text-[11px] font-mono text-slate-500 uppercase block mb-2">
                Signature & Line Span
              </span>
              <pre className="text-xs font-mono text-indigo-300 overflow-x-auto bg-slate-950 p-3 rounded-lg border border-slate-800">
                {activeSymbol.signature}
              </pre>
              <div className="mt-3 flex gap-4 text-xs font-mono text-slate-400">
                <span>Start: Line {activeSymbol.start_line}</span>
                <span>End: Line {activeSymbol.end_line}</span>
                <span>Tokens: ~{activeSymbol.token_count}</span>
              </div>
            </div>

            <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-4">
              <span className="text-[11px] font-mono text-slate-500 uppercase block mb-2">
                AST Node Body
              </span>
              <pre className="text-xs font-mono text-slate-300 overflow-x-auto bg-slate-950 p-3 rounded-lg border border-slate-800 max-h-64">
                {activeSymbol.body}
              </pre>
            </div>
          </div>
        ) : (
          <div className="h-full flex flex-col items-center justify-center text-slate-500 text-sm">
            <Layers className="w-8 h-8 mb-2 opacity-50" />
            Select a symbol from the hierarchy tree to inspect AST details
          </div>
        )}

        {activeSymbol && (
          <div className="pt-6 border-t border-slate-800">
            <button
              onClick={() => onSelectSymbol(activeSymbol.qualified_name)}
              className="w-full py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/25 transition-all"
            >
              Open in Context Packer Sandbox
              <ArrowUpRight className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
};
