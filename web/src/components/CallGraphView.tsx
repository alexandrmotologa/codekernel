import React, { useRef, useEffect, useState } from 'react';
import { GraphData, GraphNode } from '../types';
import { Search, ZoomIn, ZoomOut, Maximize2, ShieldCheck, ArrowRight } from 'lucide-react';

interface CallGraphViewProps {
  graphData: GraphData | null;
  onSelectNode: (node: GraphNode) => void;
}

interface SimNode extends GraphNode {
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
}

const LANGUAGE_COLORS: Record<string, string> = {
  go: '#00ADD8',
  typescript: '#3178C6',
  tsx: '#61DAFB',
  javascript: '#F7DF1E',
  python: '#3776AB',
  rust: '#DEA584',
  java: '#ED8B00',
  default: '#818CF8',
};

export const CallGraphView: React.FC<CallGraphViewProps> = ({ graphData, onSelectNode }) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedNode, setSelectedNode] = useState<SimNode | null>(null);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });

  const simNodesRef = useRef<SimNode[]>([]);

  useEffect(() => {
    if (!graphData?.nodes || graphData.nodes.length === 0) return;

    const width = 1200;
    const height = 800;

    simNodesRef.current = graphData.nodes.map((n, i) => {
      const angle = (i / graphData.nodes.length) * 2 * Math.PI;
      const radius = 100 + ((i * 37) % 250);
      return {
        ...n,
        x: width / 2 + Math.cos(angle) * radius,
        y: height / 2 + Math.sin(angle) * radius,
        vx: 0,
        vy: 0,
        radius: Math.max(7, Math.min(16, Math.sqrt(n.tokens || 10) * 1.2)),
      };
    });
  }, [graphData]);

  // Simulation and render loop
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let animId: number;

    const tick = () => {
      const nodes = simNodesRef.current;
      if (nodes.length > 0) {
        const centerX = canvas.width / 2;
        const centerY = canvas.height / 2;

        for (let i = 0; i < nodes.length; i++) {
          // Gentle center gravity
          nodes[i].vx += (centerX - nodes[i].x) * 0.001;
          nodes[i].vy += (centerY - nodes[i].y) * 0.001;

          // Repulsion between nearby nodes
          for (let j = i + 1; j < nodes.length; j++) {
            const dx = nodes[j].x - nodes[i].x;
            const dy = nodes[j].y - nodes[i].y;
            const dist = Math.sqrt(dx * dx + dy * dy) || 1;
            if (dist < 80) {
              const force = Math.min(1.0, (80 - dist) / dist * 0.03);
              nodes[i].vx -= dx * force;
              nodes[i].vy -= dy * force;
              nodes[j].vx += dx * force;
              nodes[j].vy += dy * force;
            }
          }

          // Damping and speed limit
          nodes[i].vx = Math.max(-4, Math.min(4, nodes[i].vx * 0.88));
          nodes[i].vy = Math.max(-4, Math.min(4, nodes[i].vy * 0.88));
          nodes[i].x += nodes[i].vx;
          nodes[i].y += nodes[i].vy;

          // Screen bounds
          nodes[i].x = Math.max(40, Math.min(canvas.width - 40, nodes[i].x));
          nodes[i].y = Math.max(40, Math.min(canvas.height - 40, nodes[i].y));
        }
      }

      // Render
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.save();
      ctx.translate(pan.x, pan.y);
      ctx.scale(zoom, zoom);

      // Draw Edges
      if (graphData?.edges) {
        ctx.lineWidth = 1;
        graphData.edges.forEach((edge) => {
          const src = nodes.find((n) => n.id === edge.source);
          const tgt = nodes.find((n) => n.id === edge.target);
          if (src && tgt) {
            ctx.beginPath();
            ctx.moveTo(src.x, src.y);
            ctx.lineTo(tgt.x, tgt.y);
            ctx.strokeStyle = 'rgba(99, 102, 241, 0.25)';
            ctx.stroke();

            // Arrow head
            const angle = Math.atan2(tgt.y - src.y, tgt.x - src.x);
            const arrowLen = 6;
            ctx.beginPath();
            ctx.moveTo(tgt.x - Math.cos(angle) * (tgt.radius + 2), tgt.y - Math.sin(angle) * (tgt.radius + 2));
            ctx.lineTo(
              tgt.x - Math.cos(angle - Math.PI / 6) * (tgt.radius + arrowLen + 2),
              tgt.y - Math.sin(angle - Math.PI / 6) * (tgt.radius + arrowLen + 2)
            );
            ctx.lineTo(
              tgt.x - Math.cos(angle + Math.PI / 6) * (tgt.radius + arrowLen + 2),
              tgt.y - Math.sin(angle + Math.PI / 6) * (tgt.radius + arrowLen + 2)
            );
            ctx.fillStyle = 'rgba(99, 102, 241, 0.4)';
            ctx.fill();
          }
        });
      }

      // Draw Nodes
      nodes.forEach((node) => {
        const isMatch = searchQuery && node.name.toLowerCase().includes(searchQuery.toLowerCase());
        const isSelected = selectedNode?.id === node.id;
        const color = LANGUAGE_COLORS[node.language] || LANGUAGE_COLORS.default;

        ctx.beginPath();
        ctx.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
        ctx.fillStyle = isSelected ? '#ffffff' : color;
        ctx.shadowColor = color;
        ctx.shadowBlur = isSelected ? 15 : isMatch ? 12 : 6;
        ctx.fill();
        ctx.shadowBlur = 0;

        // Label
        if (zoom > 0.8 || isMatch || isSelected) {
          ctx.font = '10px JetBrains Mono, monospace';
          ctx.fillStyle = isSelected ? '#ffffff' : '#94a3b8';
          ctx.textAlign = 'center';
          ctx.fillText(node.name, node.x, node.y + node.radius + 12);
        }
      });

      ctx.restore();
      animId = requestAnimationFrame(tick);
    };

    animId = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animId);
  }, [graphData, pan, zoom, searchQuery, selectedNode]);

  // Click & Drag handlers
  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    setIsDragging(true);
    setDragStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });

    // Detect clicked node
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const mouseX = (e.clientX - rect.left - pan.x) / zoom;
    const mouseY = (e.clientY - rect.top - pan.y) / zoom;

    const clicked = simNodesRef.current.find((n) => {
      const dx = n.x - mouseX;
      const dy = n.y - mouseY;
      return Math.sqrt(dx * dx + dy * dy) <= n.radius + 4;
    });

    if (clicked) {
      setSelectedNode(clicked);
      onSelectNode(clicked);
    }
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (isDragging) {
      setPan({
        x: e.clientX - dragStart.x,
        y: e.clientY - dragStart.y,
      });
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  return (
    <div className="relative w-full h-[calc(100vh-4rem)] bg-[#070a10] overflow-hidden flex">
      {/* Search & Zoom Overlay */}
      <div className="absolute top-4 left-4 z-10 flex items-center gap-2">
        <div className="relative">
          <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search symbols..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9 pr-3 py-1.5 rounded-lg bg-slate-900/90 border border-slate-700 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-indigo-500 w-56 font-mono"
          />
        </div>

        <div className="flex items-center gap-1 bg-slate-900/90 p-1 rounded-lg border border-slate-700">
          <button
            onClick={() => setZoom((z) => Math.min(z + 0.2, 3))}
            className="p-1 hover:bg-slate-800 rounded text-slate-300"
            title="Zoom In"
          >
            <ZoomIn className="w-4 h-4" />
          </button>
          <button
            onClick={() => setZoom((z) => Math.max(z - 0.2, 0.4))}
            className="p-1 hover:bg-slate-800 rounded text-slate-300"
            title="Zoom Out"
          >
            <ZoomOut className="w-4 h-4" />
          </button>
          <button
            onClick={() => {
              setZoom(1);
              setPan({ x: 0, y: 0 });
            }}
            className="p-1 hover:bg-slate-800 rounded text-slate-300"
            title="Reset View"
          >
            <Maximize2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Language Legend */}
      <div className="absolute bottom-4 left-4 z-10 flex items-center gap-3 bg-slate-900/90 border border-slate-800 px-3 py-1.5 rounded-lg text-xs font-mono text-slate-400">
        {Object.entries(LANGUAGE_COLORS).map(([lang, color]) => (
          <span key={lang} className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: color }} />
            {lang}
          </span>
        ))}
      </div>

      {/* Main Canvas */}
      <canvas
        ref={canvasRef}
        width={1400}
        height={900}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        className="w-full h-full cursor-grab active:cursor-grabbing"
      />

      {/* Node Inspector Drawer */}
      {selectedNode && (
        <div className="w-80 border-l border-slate-800 bg-[#0d121f]/95 p-4 z-20 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between pb-3 border-b border-slate-800">
              <span className="text-[10px] uppercase font-mono tracking-wider px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                {selectedNode.kind}
              </span>
              <span className="text-xs font-mono text-slate-400">{selectedNode.language}</span>
            </div>

            <div className="mt-3">
              <h3 className="font-mono text-sm font-semibold text-white break-all">
                {selectedNode.name}
              </h3>
              <p className="text-xs font-mono text-slate-400 break-all mt-1">
                {selectedNode.qualified}
              </p>
            </div>

            <div className="mt-4 space-y-2 text-xs font-mono text-slate-300">
              <div className="flex justify-between py-1 border-b border-slate-800/60">
                <span className="text-slate-500">File:</span>
                <span className="text-right truncate max-w-[160px]">{selectedNode.file_path}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/60">
                <span className="text-slate-500">Tokens:</span>
                <span>~{selectedNode.tokens}</span>
              </div>
              <div className="flex justify-between py-1 border-b border-slate-800/60">
                <span className="text-slate-500">Exported:</span>
                <span>{selectedNode.exported ? 'Yes' : 'No'}</span>
              </div>
            </div>
          </div>

          <div className="pt-4 border-t border-slate-800">
            <button
              onClick={() => onSelectNode(selectedNode)}
              className="w-full py-2 px-3 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-medium flex items-center justify-center gap-2 shadow-lg shadow-indigo-600/20 transition-all"
            >
              Pack Context Subgraph
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
