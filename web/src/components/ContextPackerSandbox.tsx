import React, { useState, useEffect } from 'react';
import { PackedResult, SymbolItem } from '../types';
import { Box, Sliders, Copy, Check, Sparkles, Layers, ShieldCheck, ArrowRight } from 'lucide-react';

interface ContextPackerSandboxProps {
  initialSymbol?: string;
  symbols: SymbolItem[];
}

export const ContextPackerSandbox: React.FC<ContextPackerSandboxProps> = ({
  initialSymbol,
  symbols,
}) => {
  const [symbolName, setSymbolName] = useState(initialSymbol || '');
  const [budget, setBudget] = useState(4096);
  const [result, setResult] = useState<PackedResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);
  const [viewMode, setViewMode] = useState<'markdown' | 'visual'>('visual');

  useEffect(() => {
    if (initialSymbol) {
      setSymbolName(initialSymbol);
      handlePack(initialSymbol, budget);
    } else if (symbols.length > 0 && !symbolName) {
      const first = symbols[0].qualified_name;
      setSymbolName(first);
      handlePack(first, budget);
    }
  }, [initialSymbol, symbols]);

  const handlePack = async (sym: string, bud: number) => {
    if (!sym) return;
    setLoading(true);
    try {
      const res = await fetch('/api/pack', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ symbol: sym, budget: bud }),
      });
      if (res.ok) {
        const data = await res.json();
        setResult(data);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    if (!result) return;
    navigator.clipboard.writeText(result.markdown);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const percentUsed = result ? Math.min(100, Math.round((result.used_tokens / result.budget) * 100)) : 0;

  return (
    <div className="flex w-full h-[calc(100vh-4rem)] bg-[#070a10]">
      {/* Configuration Sidebar */}
      <div className="w-80 border-r border-slate-800 p-6 flex flex-col justify-between bg-[#0d121f]">
        <div className="space-y-6">
          <div>
            <div className="flex items-center gap-2 text-indigo-400 mb-1">
              <Box className="w-4 h-4" />
              <span className="text-xs uppercase font-mono tracking-wider font-semibold">
                Context Packing Core
              </span>
            </div>
            <h2 className="text-lg font-bold text-white">Knapsack Optimizer</h2>
            <p className="text-xs text-slate-400 mt-1">
              Selects target definition + 1-hop callers and callees fitting strictly within token bounds.
            </p>
          </div>

          {/* Symbol Selector */}
          <div className="space-y-2">
            <label className="text-xs font-mono text-slate-400 block">Target Symbol Entrypoint</label>
            <input
              type="text"
              value={symbolName}
              onChange={(e) => setSymbolName(e.target.value)}
              placeholder="e.g. ProcessPayment, VerifyToken"
              className="w-full px-3 py-2 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500"
            />
            {symbols.length > 0 && (
              <select
                onChange={(e) => {
                  setSymbolName(e.target.value);
                  handlePack(e.target.value, budget);
                }}
                className="w-full px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-400 font-mono focus:outline-none"
              >
                <option value="">Quick select indexed symbol...</option>
                {symbols.slice(0, 30).map((s) => (
                  <option key={s.qualified_name} value={s.qualified_name}>
                    {s.qualified_name}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Budget Slider */}
          <div className="space-y-3">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-slate-400">Token Budget Cap</span>
              <span className="text-indigo-400 font-bold">{budget} tokens</span>
            </div>
            <input
              type="range"
              min="512"
              max="16384"
              step="512"
              value={budget}
              onChange={(e) => {
                const b = parseInt(e.target.value);
                setBudget(b);
                handlePack(symbolName, b);
              }}
              className="w-full accent-indigo-500"
            />
            <div className="flex justify-between text-[10px] font-mono text-slate-500">
              <span>512</span>
              <span>4,096</span>
              <span>8,192</span>
              <span>16,384</span>
            </div>
          </div>

          {/* Token Budget Gauge */}
          {result && (
            <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-4 space-y-3 font-mono text-xs">
              <div className="flex justify-between">
                <span className="text-slate-400">Budget Consumed:</span>
                <span className="text-white font-bold">{percentUsed}%</span>
              </div>
              <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                <div
                  className="bg-gradient-to-r from-emerald-500 via-indigo-500 to-amber-500 h-full transition-all duration-300"
                  style={{ width: `${percentUsed}%` }}
                />
              </div>
              <div className="flex justify-between text-slate-400 text-[11px]">
                <span>Used: {result.used_tokens} tokens</span>
                <span>Cap: {result.budget}</span>
              </div>
              <div className="pt-2 border-t border-slate-800 text-[11px] text-slate-400 space-y-1">
                <div>Compressed signatures: {result.packed.compressed_count}</div>
                <div>Omitted distant nodes: {result.packed.dropped_count}</div>
              </div>
            </div>
          )}
        </div>

        <button
          onClick={() => handlePack(symbolName, budget)}
          disabled={loading || !symbolName}
          className="w-full py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/25 transition-all"
        >
          <Sparkles className="w-4 h-4" />
          {loading ? 'Optimizing Context...' : 'Re-pack Context'}
        </button>
      </div>

      {/* Main Workspace Preview */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Workspace Toolbar */}
        <div className="h-14 border-b border-slate-800 px-6 flex items-center justify-between bg-[#0d121f]">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setViewMode('visual')}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium font-mono transition-all ${
                viewMode === 'visual'
                  ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30'
                  : 'text-slate-400 hover:bg-slate-800'
              }`}
            >
              Visual Subgraph
            </button>
            <button
              onClick={() => setViewMode('markdown')}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium font-mono transition-all ${
                viewMode === 'markdown'
                  ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30'
                  : 'text-slate-400 hover:bg-slate-800'
              }`}
            >
              Raw Markdown Prompt
            </button>
          </div>

          <button
            onClick={handleCopy}
            disabled={!result}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition-all"
          >
            {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
            {copied ? 'Copied Prompt!' : 'Copy Context'}
          </button>
        </div>

        {/* Workspace Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {result ? (
            viewMode === 'visual' ? (
              <div className="space-y-6 max-w-4xl mx-auto">
                {/* Target Definition Block */}
                <div className="bg-slate-900/90 border border-indigo-500/30 rounded-xl p-5 shadow-lg shadow-indigo-500/5">
                  <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-800">
                    <div className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-emerald-400" />
                      <span className="text-xs font-mono font-bold text-white uppercase">
                        Target Symbol Definition (Hop 0)
                      </span>
                    </div>
                    <span className="text-xs font-mono text-indigo-400">
                      {result.packed.target_chunk.symbol_name} (~{result.packed.target_chunk.token_count} tokens)
                    </span>
                  </div>
                  <pre className="text-xs font-mono text-slate-300 bg-slate-950 p-4 rounded-lg border border-slate-800 overflow-x-auto whitespace-pre-wrap">
                    {result.packed.target_chunk.content}
                  </pre>
                </div>

                {/* Direct Callers Block */}
                {result.packed.caller_chunks?.length > 0 && (
                  <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-5">
                    <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-800">
                      <span className="text-xs font-mono font-bold text-amber-400 uppercase">
                        Direct Callers ({result.packed.caller_chunks.length})
                      </span>
                      <span className="text-xs font-mono text-slate-500">Inbound Call Sites</span>
                    </div>
                    <div className="space-y-3">
                      {result.packed.caller_chunks.map((c, i) => (
                        <div key={i} className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
                          <div className="text-[11px] font-mono text-slate-400 mb-1">
                            {c.file_path} (lines {c.start_line}-{c.end_line})
                          </div>
                          <pre className="text-xs font-mono text-slate-300 overflow-x-auto whitespace-pre-wrap">
                            {c.content}
                          </pre>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Dependencies Block */}
                {result.packed.callee_chunks?.length > 0 && (
                  <div className="bg-slate-900/90 border border-slate-800 rounded-xl p-5">
                    <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-800">
                      <span className="text-xs font-mono font-bold text-cyan-400 uppercase">
                        Immediate Dependencies ({result.packed.callee_chunks.length})
                      </span>
                      <span className="text-xs font-mono text-slate-500">Outbound Calls</span>
                    </div>
                    <div className="space-y-3">
                      {result.packed.callee_chunks.map((c, i) => (
                        <div key={i} className="bg-slate-950 p-3 rounded-lg border border-slate-800/80">
                          <div className="text-[11px] font-mono text-slate-400 mb-1">
                            {c.file_path} (lines {c.start_line}-{c.end_line})
                          </div>
                          <pre className="text-xs font-mono text-slate-300 overflow-x-auto whitespace-pre-wrap">
                            {c.content}
                          </pre>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="max-w-4xl mx-auto">
                <pre className="text-xs font-mono text-slate-200 bg-slate-950 p-6 rounded-xl border border-slate-800 overflow-x-auto whitespace-pre-wrap selection:bg-indigo-600">
                  {result.markdown}
                </pre>
              </div>
            )
          ) : (
            <div className="h-full flex items-center justify-center text-slate-500 font-mono text-xs">
              No context packed. Select a symbol and token budget to generate surgical prompt context.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
