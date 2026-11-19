import React, { useState, useEffect } from 'react';
import { FileRecord, SymbolItem } from '../types';
import { FileCode, Search, Tag, Eye } from 'lucide-react';

interface FileViewerProps {
  files: FileRecord[];
}

export const FileViewer: React.FC<FileViewerProps> = ({ files }) => {
  const [selectedPath, setSelectedPath] = useState<string>(files[0]?.path || '');
  const [fileData, setFileData] = useState<{
    path: string;
    content: string;
    symbols: SymbolItem[];
    language: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  useEffect(() => {
    if (selectedPath) {
      loadFile(selectedPath);
    } else if (files.length > 0) {
      setSelectedPath(files[0].path);
      loadFile(files[0].path);
    }
  }, [selectedPath, files]);

  const loadFile = async (path: string) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/file?path=${encodeURIComponent(path)}`);
      if (res.ok) {
        const data = await res.json();
        setFileData(data);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const filteredFiles = files.filter((f) =>
    f.path.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="flex w-full h-[calc(100vh-4rem)] bg-[#070a10]">
      {/* File List */}
      <div className="w-80 border-r border-slate-800 flex flex-col bg-[#0d121f]">
        <div className="p-4 border-b border-slate-800">
          <div className="relative">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filter files..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-slate-900 border border-slate-700 text-xs text-slate-200 placeholder-slate-500 font-mono focus:outline-none focus:border-indigo-500"
            />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-2 space-y-1 font-mono text-xs">
          {filteredFiles.map((f) => (
            <button
              key={f.id}
              onClick={() => setSelectedPath(f.path)}
              className={`w-full text-left px-3 py-2 rounded-lg flex items-center justify-between transition-colors ${
                selectedPath === f.path
                  ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/30'
                  : 'text-slate-400 hover:bg-slate-800/60'
              }`}
            >
              <div className="flex items-center gap-2 truncate">
                <FileCode className="w-4 h-4 flex-shrink-0 text-slate-500" />
                <span className="truncate">{f.path}</span>
              </div>
              <span className="text-[10px] text-slate-600">{f.line_count}L</span>
            </button>
          ))}
        </div>
      </div>

      {/* File Editor View */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {fileData ? (
          <>
            {/* Header info */}
            <div className="h-14 border-b border-slate-800 px-6 flex items-center justify-between bg-[#0d121f]">
              <div className="flex items-center gap-3">
                <span className="text-xs uppercase font-mono px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                  {fileData.language}
                </span>
                <span className="font-mono text-xs font-semibold text-white truncate">
                  {fileData.path}
                </span>
              </div>

              {fileData.symbols?.length > 0 && (
                <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
                  <Tag className="w-3.5 h-3.5 text-indigo-400" />
                  <span>{fileData.symbols.length} AST symbols</span>
                </div>
              )}
            </div>

            {/* Code Lines View */}
            <div className="flex-1 overflow-y-auto p-6 font-mono text-xs bg-slate-950">
              <div className="table w-full">
                {fileData.content.split('\n').map((line, idx) => {
                  const lineNum = idx + 1;
                  const matchingSym = fileData.symbols?.find((s) => s.start_line === lineNum);

                  return (
                    <div
                      key={idx}
                      className={`table-row hover:bg-slate-900/60 ${
                        matchingSym ? 'bg-indigo-950/20 border-l-2 border-indigo-500' : ''
                      }`}
                    >
                      <span className="table-cell pr-4 text-right text-slate-600 select-none w-12 py-0.5">
                        {lineNum}
                      </span>
                      <span className="table-cell text-slate-200 whitespace-pre py-0.5">
                        {line || ' '}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          </>
        ) : (
          <div className="h-full flex items-center justify-center text-slate-500 font-mono text-xs">
            {loading ? 'Loading file...' : 'Select a file to inspect'}
          </div>
        )}
      </div>
    </div>
  );
};
