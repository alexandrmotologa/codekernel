import React, { useState, useEffect } from 'react';
import { Navbar } from './components/Navbar';
import { CallGraphView } from './components/CallGraphView';
import { SymbolHierarchy } from './components/SymbolHierarchy';
import { ContextPackerSandbox } from './components/ContextPackerSandbox';
import { FileViewer } from './components/FileViewer';
import { RepoStats, FileRecord, SymbolItem, GraphData, IndexProgress, GraphNode } from './types';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'graph' | 'symbols' | 'packer' | 'files'>('graph');
  const [stats, setStats] = useState<RepoStats | null>(null);
  const [files, setFiles] = useState<FileRecord[]>([]);
  const [symbols, setSymbols] = useState<SymbolItem[]>([]);
  const [graphData, setGraphData] = useState<GraphData | null>(null);
  const [indexing, setIndexing] = useState(false);
  const [progress, setProgress] = useState<IndexProgress | null>(null);
  const [packerSymbol, setPackerSymbol] = useState<string>('');

  const loadAll = async () => {
    try {
      // 1. Stats
      const statsRes = await fetch('/api/stats');
      if (statsRes.ok) setStats(await statsRes.json());

      // 2. Files
      const filesRes = await fetch('/api/files');
      if (filesRes.ok) setFiles(await filesRes.json());

      // 3. Symbols
      const symRes = await fetch('/api/symbols?limit=300');
      if (symRes.ok) setSymbols(await symRes.json());

      // 4. Graph
      const graphRes = await fetch('/api/graph?limit=500');
      if (graphRes.ok) setGraphData(await graphRes.json());
    } catch (err) {
      console.error('Failed to load codekernel data:', err);
    }
  };

  useEffect(() => {
    loadAll();

    // Setup SSE listener for live indexing progress
    const evtSource = new EventSource('/api/events');
    evtSource.onmessage = (e) => {
      try {
        const prog: IndexProgress = JSON.parse(e.data);
        setProgress(prog);
        if (prog.completed) {
          setIndexing(false);
          loadAll();
        } else {
          setIndexing(true);
        }
      } catch (err) {
        console.error(err);
      }
    };

    return () => evtSource.close();
  }, []);

  const handleReindex = async () => {
    setIndexing(true);
    try {
      await fetch('/api/index', { method: 'POST' });
    } catch (err) {
      console.error('Failed to trigger re-index:', err);
      setIndexing(false);
    }
  };

  const handleSelectNode = (node: GraphNode) => {
    setPackerSymbol(node.qualified || node.name);
    setActiveTab('packer');
  };

  const handleSelectSymbolFromHierarchy = (symbolName: string) => {
    setPackerSymbol(symbolName);
    setActiveTab('packer');
  };

  return (
    <div className="min-h-screen bg-[#070a10] text-slate-100 flex flex-col font-sans">
      <Navbar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        stats={stats}
        indexing={indexing}
        progress={progress}
        onReindex={handleReindex}
      />

      <main className="flex-1 flex overflow-hidden">
        {activeTab === 'graph' && (
          <CallGraphView graphData={graphData} onSelectNode={handleSelectNode} />
        )}
        {activeTab === 'symbols' && (
          <SymbolHierarchy
            symbols={symbols}
            onSelectSymbol={handleSelectSymbolFromHierarchy}
          />
        )}
        {activeTab === 'packer' && (
          <ContextPackerSandbox
            initialSymbol={packerSymbol}
            symbols={symbols}
          />
        )}
        {activeTab === 'files' && <FileViewer files={files} />}
      </main>
    </div>
  );
};

export default App;
