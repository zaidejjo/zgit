package ai

import (
	"path/filepath"
	"strings"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// Lockfile/binary extensions to exclude from diff payload
var ignoredExtensions = map[string]bool{
	".lock": true,
	".sum":  true,

	// Binary artifacts
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".svg":   true,
	".ico":   true,
	".woff":  true,
	".woff2": true,
	".eot":   true,
	".ttf":   true,
	".mp4":   true,
	".webm":  true,
	".zip":   true,
	".tar":   true,
	".gz":    true,
	".bz2":   true,
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".class": true,
	".pyc":   true,
	".o":     true,
	".a":     true,
}

// Known lockfile base names
var ignoredBaseNames = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"go.sum":            true,
	"Cargo.lock":        true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
	"composer.lock":     true,
	"Podfile.lock":      true,
	"mix.lock":          true,
}

// DiffSummary contains token-optimized diff representation.
type DiffSummary struct {
	ChangedFiles int    `json:"changed_files"`
	Summary      string `json:"summary"`   // optimized diff text or --stat output
	IsFull       bool   `json:"is_full"`   // true when summary contains actual diff (not just stat)
	Truncated    bool   `json:"truncated"` // true when diff was truncated due to size
	Filtered     int    `json:"filtered"`  // number of files filtered out (lockfiles/binary)
}

// SummarizeDiff optimizes a diff for token-efficient LLM consumption.
//
// Heuristics:
//   - Remove lockfiles and binary files from the diff payload
//   - If after filtering >50 files remain, fall back to --stat format
//   - Truncate total diff to ~2000 characters (~500 tokens)
func SummarizeDiff(diff *models.Diff) *DiffSummary {
	if diff == nil || len(diff.Files) == 0 {
		return &DiffSummary{Summary: "No changes."}
	}

	var filtered int
	var includedFiles []models.FileChange

	for _, f := range diff.Files {
		if isIgnoredFile(f.NewPath) || isIgnoredFile(f.OldPath) {
			filtered++
			continue
		}
		if f.IsBinary {
			filtered++
			continue
		}
		includedFiles = append(includedFiles, f)
	}

	if len(includedFiles) == 0 {
		return &DiffSummary{
			Filtered:     filtered,
			ChangedFiles: len(diff.Files),
			Summary:      "All changes are in lockfiles or binary files — no meaningful diff to analyze.",
		}
	}

	// If too many files, use --stat format only
	if len(includedFiles) > 50 {
		return &DiffSummary{
			ChangedFiles: len(diff.Files),
			Filtered:     filtered,
			IsFull:       false,
			Summary:      buildStatSummary(includedFiles),
		}
	}

	// Build and truncate unified diff
	var sb strings.Builder
	sb.Grow(2500) // pre-allocate a bit over our limit

	totalAdds := 0
	totalDels := 0
	for _, f := range includedFiles {
		if f.UnifiedDiff != "" {
			if sb.Len()+len(f.UnifiedDiff) > 2000 {
				// Truncate: add remaining file count as note
				remaining := 0
				for j := range includedFiles {
					if includedFiles[j].UnifiedDiff != "" && includedFiles[j].UnifiedDiff != f.UnifiedDiff {
						remaining++
					}
				}
				// Write the current file's header but truncated
				path := f.NewPath
				if path == "" {
					path = f.OldPath
				}
				sb.WriteString(colorDiffHeader(path))
				sb.WriteString(truncateDiffLine(f.UnifiedDiff, 2000-sb.Len()))
				if remaining > 0 {
					sb.WriteString(colorDiffHeader(""))
					sb.WriteString("... and " + itoa(remaining) + " more files (truncated)\n")
				}
				return &DiffSummary{
					ChangedFiles: len(diff.Files),
					Filtered:     filtered,
					IsFull:       true,
					Truncated:    true,
					Summary:      sb.String(),
				}
			}
			sb.WriteString(f.UnifiedDiff)
			sb.WriteByte('\n')
		}
		totalAdds += f.Additions
		totalDels += f.Deletions
	}

	return &DiffSummary{
		ChangedFiles: len(diff.Files),
		Filtered:     filtered,
		IsFull:       true,
		Summary:      sb.String(),
	}
}

func isIgnoredFile(path string) bool {
	if path == "" {
		return false
	}

	base := filepath.Base(path)
	if ignoredBaseNames[base] {
		return true
	}

	ext := filepath.Ext(path)
	return ignoredExtensions[ext]
}

func buildStatSummary(files []models.FileChange) string {
	var sb strings.Builder
	sb.WriteString("Changes by file:\n")
	totalAdds := 0
	totalDels := 0
	for _, f := range files {
		path := f.NewPath
		if path == "" {
			path = f.OldPath
		}
		sb.WriteString("  ")
		sb.WriteString(path)
		sb.WriteString(" | ")
		sb.WriteString(itoa(f.Additions))
		sb.WriteString(" ++, ")
		sb.WriteString(itoa(f.Deletions))
		sb.WriteString(" --\n")
		totalAdds += f.Additions
		totalDels += f.Deletions
	}
	sb.WriteString("\n")
	sb.WriteString(itoa(len(files)))
	sb.WriteString(" files changed, ")
	sb.WriteString(itoa(totalAdds))
	sb.WriteString(" insertions(+), ")
	sb.WriteString(itoa(totalDels))
	sb.WriteString(" deletions(-)")
	return sb.String()
}

func truncateDiffLine(diff string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	// Keep diff header lines, truncate the actual content
	if len(diff) <= maxLen {
		return diff
	}
	// Reserve space for truncation marker
	usable := maxLen - 40
	if usable < 40 {
		usable = 40
	}
	return diff[:usable] + "\n# ... diff truncated for token efficiency\n"
}

// colorDiffHeader prefixes a path like a git diff header comment
func colorDiffHeader(path string) string {
	if path == "" {
		return ""
	}
	return "# File: " + path + "\n"
}

// itoa is a fast int-to-string for small numbers (avoids fmt import)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		digits = append(digits, '-')
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
