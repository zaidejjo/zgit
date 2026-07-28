import { Commit } from "@/store/app";
import { GraphLayout, GraphNode, GraphEdge } from "./types";
import { getBranchColor } from "./branchColors";

export interface LayoutOptions {
  nodeRadius?: number;
  rowHeight?: number;
  columnWidth?: number;
  paddingX?: number;
  paddingY?: number;
}

const DEFAULTS = {
  nodeRadius: 8,
  rowHeight: 44,
  columnWidth: 48,
  paddingX: 32,
  paddingY: 16,
};

/**
 * Compute graph layout from an ordered list of commits.
 *
 * Algorithm: standard git column assignment used by tig, GitX, etc.
 * - Process commits newest→oldest (index 0 = top of graph).
 * - Maintain `activeColumns: string[]` — one entry per active lane.
 *   Each entry is a commit hash (the "tip" of that lane) or "" (vacant).
 * - For each commit:
 *   1. Find its hash in activeColumns → position = column index.
 *      If not found, compute an appropriate position.
 *   2. Record column index (immutable).
 *   3. Remove the hash from its column slot.
 *   4. Insert each parent at the same position (first parent inherits lane).
 *      Skip if parent already in activeColumns (avoids duplicate columns).
 *
 * Compaction: trailing empty slots are trimmed; middle empties are collapsed
 * to keep the column count minimal without shifting already-assigned positions.
 *
 * Input MUST be in topo-order (newest first) from `--topo-order`.
 */
export function computeLayout(
  commits: Commit[],
  options: LayoutOptions = {},
): GraphLayout {
  const {
    nodeRadius = DEFAULTS.nodeRadius,
    rowHeight = DEFAULTS.rowHeight,
    columnWidth = DEFAULTS.columnWidth,
    paddingX = DEFAULTS.paddingX,
    paddingY = DEFAULTS.paddingY,
  } = options;

  if (commits.length === 0) {
    return {
      nodes: [], edges: [],
      columns: 0, width: 0, height: 0,
      nodeRadius, rowHeight, columnWidth, paddingX, paddingY,
    };
  }

  const n = commits.length;
  const colOf = new Array<number>(n); // column index per commit
  const activeColumns: string[] = [];
  const hashToIndex = new Map<string, number>();

  for (let i = 0; i < n; i++) {
    hashToIndex.set(commits[i].hash, i);
  }

  // ── Pass 1: assign columns ──
  for (let i = 0; i < n; i++) {
    const c = commits[i];
    const cParents = c.parents || []; // guard: Go nil slice → null
    let pos = activeColumns.indexOf(c.hash);

    if (pos === -1) {
      // Commit not in any active lane.
      // Try parent-first positioning for branch continuity.
      if (cParents.length > 0) {
        const parentPos = findFirstActiveParent(cParents, activeColumns);
        pos = parentPos !== -1 ? parentPos : activeColumns.length;
      } else {
        pos = activeColumns.length; // root commit → new lane
      }
    }

    colOf[i] = pos;

    // Ensure activeColumns has at least pos+1 slots
    while (activeColumns.length <= pos) {
      activeColumns.push("");
    }

    // Remove this commit from its column slot
    activeColumns[pos] = "";

    // Insert parents at the column position
    // First parent inherits lane; subsequent parents extend to new lanes
    let insertAt = pos;
    for (const parentHash of cParents) {
      if (!activeColumns.includes(parentHash)) {
        activeColumns.splice(insertAt, 0, parentHash);
        insertAt++;
      }
    }

    // Compact trailing empties
    while (activeColumns.length > 0 && activeColumns[activeColumns.length - 1] === "") {
      activeColumns.pop();
    }

    // Compact middle empties — collapse without affecting assigned colOf
    // (only future lookups change, which is correct)
    const compacted = activeColumns.filter((h) => h !== "");
    if (compacted.length < activeColumns.length) {
      activeColumns.length = 0;
      activeColumns.push(...compacted);
    }
  }

  // ── Pass 2: compute rendering positions ──
  const totalColumns = Math.max(...colOf) + 1;
  const width = paddingX * 2 + totalColumns * columnWidth;
  const height = paddingY * 2 + n * rowHeight;

  const nodes: GraphNode[] = [];
  const edgeMap = new Map<string, GraphEdge[]>();

  for (let i = 0; i < n; i++) {
    const c = commits[i];
    const cParents = c.parents || [];
    const col = colOf[i];
    const x = paddingX + col * columnWidth + columnWidth / 2;
    const y = paddingY + i * rowHeight + rowHeight / 2;

    nodes.push({
      hash: c.hash,
      parents: cParents,
      column: col,
      row: i,
      x, y,
      color: getBranchColor(col),
      message: c.message,
      ref_names: c.ref_names,
    });

    for (const parentHash of cParents) {
      const parentIdx = hashToIndex.get(parentHash);
      if (parentIdx === undefined) {
        // Parent outside visible set → dangling edge
        const parentCol = findDanglingColumn(col, totalColumns);
        pushEdge(edgeMap, parentHash, {
          fromX: x, fromY: y,
          toX: paddingX + parentCol * columnWidth + columnWidth / 2,
          toY: y + rowHeight,
          fromColumn: col, toColumn: parentCol,
          color: getBranchColor(col),
          isMerge: cParents.length > 1,
          isDash: true,
        });
        continue;
      }

      const parentCol = colOf[parentIdx];
      pushEdge(edgeMap, parentHash, {
        fromX: x, fromY: y,
        toX: paddingX + parentCol * columnWidth + columnWidth / 2,
        toY: paddingY + parentIdx * rowHeight + rowHeight / 2,
        fromColumn: col, toColumn: parentCol,
        color: getBranchColor(col),
        isMerge: cParents.length > 1,
        isDash: false,
      });
    }
  }

  // Flatten edge map
  const edges: GraphEdge[] = [];
  for (const [, list] of edgeMap) edges.push(...list);

  return {
    nodes, edges,
    columns: totalColumns,
    width, height,
    nodeRadius, rowHeight, columnWidth, paddingX, paddingY,
  };
}

/** Find first parent that exists in activeColumns. Returns -1 if none. */
function findFirstActiveParent(parents: string[], activeColumns: string[]): number {
  for (const p of parents) {
    const idx = activeColumns.indexOf(p);
    if (idx !== -1) return idx;
  }
  return -1;
}

/** Place parent in adjacent column if space allows, otherwise child's column. */
function findDanglingColumn(childCol: number, totalColumns: number): number {
  if (childCol > 0) return childCol - 1;
  if (childCol + 1 < totalColumns) return childCol + 1;
  return childCol;
}

function pushEdge(map: Map<string, GraphEdge[]>, key: string, edge: GraphEdge) {
  const arr = map.get(key) || [];
  arr.push(edge);
  map.set(key, arr);
}


