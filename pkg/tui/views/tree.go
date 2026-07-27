package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

// Tree column colors — Catppuccin 6-color wheel.
var treeColors = []lipgloss.Color{
	styles.Teal,
	styles.Mauve,
	styles.Peach,
	styles.Green,
	styles.Yellow,
	styles.Blue,
}

// treeCol tracks one active column in the graph.
type treeCol struct {
	hash  string // commit hash at column tip
	color lipgloss.Color
}

// RenderTreeGraph builds a column-based topology graph for a commit list.
// Returns one graph-prefix string per commit, ready to prepend to log lines.
//
// Algorithm:
//   - Walk commits top-to-bottom (newest first)
//   - Assign each commit to a column; columns track the "frontier" hashes
//   - Reuse column when commit matches existing tip; fork new column otherwise
//   - After each commit, advance column tip to first parent
//   - Merge commits get '◆'; regular commits get '●'
//   - Branch labels from RefNames appear on their commit row
func RenderTreeGraph(commits []*models.Commit) []string {
	if len(commits) == 0 {
		return nil
	}

	var cols []treeCol
	// track per-hash which column index it's in for fast lookup
	colOfHash := make(map[string]int)

	rows := make([]string, len(commits))
	var lbs []branchLabel

	for i, c := range commits {
		// Step 1: find which column this commit occupies
		colIdx, found := colOfHash[c.Hash]
		if !found {
			// Commit not at any column tip — new branch (e.g. second parent of merge, or newly visible)
			// Insert new column at position 0 so it appears leftmost
			color := treeColors[len(cols)%len(treeColors)]
			cols = append([]treeCol{{hash: c.Hash, color: color}}, cols...)
			// Rebuild colOfHash for shifted indices
			colOfHash = rebuildColMap(cols)
			colIdx = 0
		}

		// Step 2: determine which columns are connected (merge parents)
		isMerge := len(c.Parents) > 1
		parentCols := make(map[int]bool) // column indices that are parents of this commit
		for _, p := range c.Parents {
			if idx, ok := colOfHash[p]; ok && idx != colIdx {
				parentCols[idx] = true
			}
		}

		// Step 3: draw the tree prefix for this row
		var row strings.Builder
		for j := 0; j < len(cols); j++ {
			connector := " "
			if j == colIdx {
				// Commit sits in this column
				if isMerge {
					connector = styles.StatusStagedStyle.Render("◆")
				} else {
					connector = "●"
				}
			} else if parentCols[j] {
				// This column's tip is a parent of this commit → merge line joins here
				connector = styles.SubtitleStyle.Render("┤")
			} else {
				// Check whether this column continues past this commit
				next := colNextHash(cols[j].hash, commits, i)
				if next != "" {
					connector = styles.SubtitleStyle.Render("│")
				}
			}
			row.WriteString(connector)
			row.WriteString(" ")
		}

		rows[i] = row.String()

		// Step 4: advance column tip to first parent
		if len(c.Parents) > 0 {
			cols[colIdx].hash = c.Parents[0]
			colOfHash[c.Parents[0]] = colIdx
		}
		// Remove current commit from hash map (it's no longer a tip)
		delete(colOfHash, c.Hash)
		// Remove other consumed parent hashes (they merge in, no longer tips)
		for _, p := range c.Parents {
			if p == c.Parents[0] {
				continue
			}
			delete(colOfHash, p)
		}

		// Deduplicate columns: if multiple columns have the same hash, keep only the
		// leftmost (lowest index). This happens after a merge when two columns converge
		// to the same parent.
		seen := make(map[string]int)
		var deduped []treeCol
		for idx, col := range cols {
			if _, exists := seen[col.hash]; !exists {
				seen[col.hash] = idx
				deduped = append(deduped, col)
			}
		}
		if len(deduped) != len(cols) {
			cols = deduped
			colOfHash = rebuildColMap(cols)
		}

		// Collect branch labels
		if c.RefNames != "" {
			lbs = append(lbs, branchLabel{
				text:  c.RefNames,
				color: treeColors[colIdx%len(treeColors)],
			})
		}
	}

	// Merge labels into rows as annotations
	lblIdx := 0
	for i, c := range commits {
		if c.RefNames != "" && lblIdx < len(lbs) {
			rows[i] += " " + branchLabelStyle(lbs[lblIdx])
			lblIdx++
		}
	}

	return rows
}

// colNextHash finds the next descendant of hash that appears after position pos in commits.
// Returns "" if the column ends.
func colNextHash(hash string, commits []*models.Commit, pos int) string {
	for i := pos + 1; i < len(commits); i++ {
		for _, p := range commits[i].Parents {
			if p == hash {
				return commits[i].Hash
			}
		}
	}
	return ""
}

// rebuildColMap reconstructs the hash→column lookup after column insertion.
func rebuildColMap(cols []treeCol) map[string]int {
	m := make(map[string]int, len(cols))
	for i, col := range cols {
		m[col.hash] = i
	}
	return m
}

type branchLabel struct {
	text  string
	color lipgloss.Color
}

func branchLabelStyle(l branchLabel) string {
	return lipgloss.NewStyle().
		Foreground(l.color).
		Bold(true).
		Render(l.text)
}
