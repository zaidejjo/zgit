// Distinguishable branch colors for graph lanes.
// Pairs well with dark Zinc theme.
const COLORS = [
  "#60a5fa", // blue-400
  "#f472b6", // pink-400
  "#34d399", // emerald-400
  "#fbbf24", // amber-400
  "#a78bfa", // violet-400
  "#fb923c", // orange-400
  "#22d3ee", // cyan-400
  "#f87171", // red-400
  "#a3e635", // lime-400
  "#e879f9", // fuchsia-400
  "#38bdf8", // sky-400
  "#facc15", // yellow-400
  "#818cf8", // indigo-400
  "#4ade80", // green-400
  "#c084fc", // purple-400
  "#2dd4bf", // teal-400
];

export function getBranchColor(column: number): string {
  return COLORS[column % COLORS.length];
}
