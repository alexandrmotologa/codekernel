import React from 'react';
import { Network, ListTree, Box, FileCode, RefreshCw, Cpu, Database, Zap } from 'lucide-react';
import { RepoStats, IndexProgress } from '../types';

interface NavbarProps {
  activeTab: 'graph' | 'symbols' | 'packer' | 'files';
  setActiveTab: (tab: 'graph' | 'symbols' | 'packer' | 'files') => void;
  stats: RepoStats | null;
  indexing: boolean;
  progress: IndexProgress | null;
  onReindex: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  setActiveTab,
  stats,
  indexing,
  progress,
  onReindex,
}) => {
  return (
    <header className="border-b border-slate-800 bg-[#0d121f]/90 backdrop-blur sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-4 h-16 flex items-center justify-between">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-gradient-to-tr from-indigo-600 to-cyan-400 flex items-center justify-center shadow-lg shadow-indigo-500/20">
            <Cpu className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-lg tracking-tight bg-gradient-to-r from-white via-slate-200 to-indigo-200 bg-clip-text text-transparent">
                codekernel
              </span>
              <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                Local AST Core
              </span>
            </div>
            <p className="text-[11px] text-slate-400 font-mono">Zero Cloud Egress • SQLite Graph</p>
          </div>
        </div>

        {/* Navigation Tabs */}
        <nav className="flex items-center gap-1 bg-slate-900/80 p-1 rounded-xl border border-slate-800">
          <button
            onClick={() => setActiveTab('graph')}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              activeTab === 'graph'
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <Network className="w-4 h-4" />
            Call Graph
          </button>
          <button
            onClick={() => setActiveTab('symbols')}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              activeTab === 'symbols'
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <ListTree className="w-4 h-4" />
            Symbol Tree
          </button>
          <button
            onClick={() => setActiveTab('packer')}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              activeTab === 'packer'
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <Box className="w-4 h-4" />
            Context Packer
          </button>
          <button
            onClick={() => setActiveTab('files')}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
              activeTab === 'files'
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <FileCode className="w-4 h-4" />
            Files
          </button>
        </nav>

        {/* Stats & Actions */}
        <div className="flex items-center gap-4">
          {stats && (
            <div className="hidden md:flex items-center gap-3 text-xs font-mono text-slate-400">
              <span className="flex items-center gap-1">
                <Database className="w-3.5 h-3.5 text-cyan-400" />
                {stats.total_symbols} symbols
              </span>
              <span className="flex items-center gap-1">
                <Zap className="w-3.5 h-3.5 text-amber-400" />
                {stats.total_edges} edges
              </span>
            </div>
          )}

          <button
            onClick={onReindex}
            disabled={indexing}
            className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-medium border transition-all ${
              indexing
                ? 'bg-slate-800 text-slate-500 border-slate-700 cursor-not-allowed'
                : 'bg-indigo-500/10 text-indigo-300 border-indigo-500/30 hover:bg-indigo-500/20'
            }`}
          >
            <RefreshCw className={`w-3.5 h-3.5 ${indexing ? 'animate-spin' : ''}`} />
            {indexing ? 'Indexing...' : 'Re-index'}
          </button>
        </div>
      </div>

      {/* Indexing Progress bar if active */}
      {indexing && progress && (
        <div className="w-full bg-slate-900 border-t border-slate-800 px-4 py-1.5 flex items-center justify-between text-xs font-mono text-slate-400">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-indigo-500 animate-ping" />
            <span>Scanning: {progress.current_file || 'reading files...'}</span>
          </div>
          <div>
            {progress.processed_files} / {progress.total_files} files (
            {progress.total_symbols} symbols extracted)
          </div>
        </div>
      )}
    </header>
  );
};
