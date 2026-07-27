import { useMemo, useCallback, useState, useRef, useEffect } from "react";
import { GraphLayout, GraphNode, GraphEdge } from "@/lib/git-graph/types";
import { computeLayout } from "@/lib/git-graph/layout";
import { Commit, useAppStore } from "@/store/app";
import { cn, formatTimeAgo, truncate } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { GitBranch, GitFork, GitCommitHorizontal } from "lucide-react";

interface InteractiveGitGraphProps {
  commits: Commit[];
  loading?: boolean;
  className?: string;
}

export default function InteractiveGitGraph({
  commits,
  loading,
  className,
}: InteractiveGitGraphProps) {
  const graph = useMemo(() => computeLayout(commits), [commits]);
  const [hoveredHash, setHoveredHash] = useState<string | null>(null);
  const [selectedHash, setSelectedHash] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);

  const handleNodeClick = useCallback((hash: string) => {
    setSelectedHash((prev) => (prev === hash ? null : hash));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-40 text-muted-foreground">
        Loading graph...
      </div>
    );
  }

  if (!commits.length) {
    return (
      <div className="flex flex-col items-center justify-center h-40 text-muted-foreground gap-2">
        <GitCommitHorizontal className="w-8 h-8" />
        <span>No commits yet</span>
      </div>
    );
  }

  const svgWidth = Math.max(graph.width, 400);
  const infoWidth = 320;

  return (
    <ScrollArea className={cn("h-full", className)} ref={scrollRef}>
      <div className="relative min-w-fit">
        <svg
          width={svgWidth}
          height={graph.height}
          className="select-none"
          style={{ minWidth: svgWidth }}
        >
          {/* Background grid lines for columns */}
          {Array.from({ length: graph.columns }).map((_, col) => (
            <line
              key={`grid-${col}`}
              x1={graph.paddingX + col * graph.columnWidth + graph.columnWidth / 2}
              y1={0}
              x2={graph.paddingX + col * graph.columnWidth + graph.columnWidth / 2}
              y2={graph.height}
              stroke="currentColor"
              strokeOpacity={0.04}
              strokeWidth={1}
            />
          ))}

          {/* Edges */}
          {graph.edges.map((edge, i) => (
            <GraphEdgePath key={`edge-${i}`} edge={edge} />
          ))}

          {/* Nodes */}
          {graph.nodes.map((node) => (
            <g key={node.hash}>
              {/* Invisible wider click target */}
              <rect
                x={node.x - 16}
                y={node.y - 16}
                width={32}
                height={32}
                fill="transparent"
                onClick={() => handleNodeClick(node.hash)}
                onMouseEnter={() => setHoveredHash(node.hash)}
                onMouseLeave={() => setHoveredHash(null)}
                className="cursor-pointer"
              />
              {/* Node circle */}
              <circle
                cx={node.x}
                cy={node.y}
                r={graph.nodeRadius}
                fill={
                  selectedHash === node.hash
                    ? node.color
                    : hoveredHash === node.hash
                      ? node.color + "33"
                      : "#1e1e2e"
                }
                stroke={node.color}
                strokeWidth={selectedHash === node.hash ? 3 : hoveredHash === node.hash ? 2 : 1.5}
                className="transition-all duration-100"
              />
              {/* Inner dot for merge commits */}
              {node.parents.length > 1 && (
                <circle
                  cx={node.x}
                  cy={node.y}
                  r={3}
                  fill={node.color}
                />
              )}
              {/* Ref names (branch/tag labels) */}
              {node.ref_names && (
                <g>
                  <rect
                    x={node.x + graph.nodeRadius + 6}
                    y={node.y - 8}
                    width={node.ref_names.length * 7.5 + 12}
                    height={16}
                    rx={4}
                    fill={node.color}
                    fillOpacity={0.15}
                  />
                  <text
                    x={node.x + graph.nodeRadius + 12}
                    y={node.y + 4.5}
                    fill={node.color}
                    fontSize={11}
                    fontWeight={600}
                  >
                    {node.ref_names}
                  </text>
                </g>
              )}
            </g>
          ))}
        </svg>

        {/* Commit info panel — rendered as HTML overlay beside the SVG */}
        <div
          className="absolute pointer-events-none"
          style={{
            left: graph.paddingX + graph.columns * graph.columnWidth + 24,
            top: 0,
          }}
        >
          {graph.nodes.map((node) => (
            <div
              key={`info-${node.hash}`}
              className={cn(
                "flex items-center gap-3 px-3 transition-colors duration-100",
                selectedHash === node.hash && "bg-accent/30 rounded-md",
              )}
              style={{ height: graph.rowHeight }}
            >
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {truncate(node.message, 60)}
                </p>
                <p className="text-xs text-muted-foreground truncate">
                  {node.hash.slice(0, 7)} — {node.hash}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </ScrollArea>
  );
}

function GraphEdgePath({ edge }: { edge: GraphEdge }) {
  const { fromX, fromY, toX, toY, fromColumn, toColumn, color, isDash, isMerge } = edge;

  // Straight vertical edge: same column
  if (fromColumn === toColumn) {
    return (
      <line
        x1={fromX}
        y1={fromY}
        x2={toX}
        y2={toY}
        stroke={color}
        strokeWidth={isMerge ? 2.5 : 1.5}
        strokeOpacity={0.6}
        strokeDasharray={isDash ? "4 3" : undefined}
      />
    );
  }

  // Diagonal edge: bezier curve for smooth diagonal lines
  const midY = (fromY + toY) / 2;
  const dx = Math.abs(fromX - toX);
  const curvature = Math.min(dx * 0.4, 20);

  return (
    <path
      d={`M ${fromX} ${fromY} C ${fromX} ${fromY + curvature}, ${toX} ${toY - curvature}, ${toX} ${toY}`}
      fill="none"
      stroke={color}
      strokeWidth={isMerge ? 2.5 : 1.5}
      strokeOpacity={0.6}
      strokeDasharray={isDash ? "4 3" : undefined}
    />
  );
}
