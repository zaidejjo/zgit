export interface GraphNode {
  hash: string;
  parents: string[];
  column: number;
  row: number;
  x: number;
  y: number;
  color: string;
  message: string;
  ref_names?: string;
}

export interface GraphEdge {
  fromX: number;
  fromY: number;
  toX: number;
  toY: number;
  fromColumn: number;
  toColumn: number;
  color: string;
  isMerge: boolean;
  isDash: boolean;
}

export interface GraphLayout {
  nodes: GraphNode[];
  edges: GraphEdge[];
  columns: number;
  width: number;
  height: number;
  nodeRadius: number;
  rowHeight: number;
  columnWidth: number;
  paddingX: number;
  paddingY: number;
}
