package git

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/models"
)

// -- porcelain=v2 format:
// 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
// 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <X><scored><path>
// u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <path>
// ? <path>
// ! <path>
const (
	porcelainV2Normal    = '1'
	porcelainV2Renamed   = '2'
	porcelainV2Unmerged  = 'u'
	porcelainV2Untracked = '?'
	porcelainV2Ignored   = '!'
)

// ParsePorcelainV2 parses `git status --porcelain=v2` output into a models.Status.
func ParsePorcelainV2(data []byte, branchInfo *branchInfo) *models.Status {
	s := &models.Status{
		Files: make([]models.FileStatus, 0),
	}

	if branchInfo != nil {
		s.Branch = branchInfo.branch
		s.Upstream = branchInfo.upstream
		s.Ahead = branchInfo.ahead
		s.Behind = branchInfo.behind
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		switch line[0] {
		case porcelainV2Normal, porcelainV2Renamed:
			fs := parsePorcelainV2Entry(line)
			if fs != nil {
				s.Files = append(s.Files, *fs)
			}
		case porcelainV2Unmerged:
			fs := parsePorcelainV2Entry(line)
			if fs != nil {
				s.Files = append(s.Files, *fs)
			}
		case porcelainV2Untracked:
			path := strings.TrimPrefix(line, "? ")
			s.Files = append(s.Files, models.FileStatus{
				Path:     path,
				Staged:   models.StatusUntracked,
				Unstaged: models.StatusUnmodified,
			})
		case porcelainV2Ignored:
			// skip ignored files for now
		}
	}

	s.StagedCount = len(s.StagedFiles())
	s.UnstagedCount = len(s.UnstagedFiles())
	s.UntrackedCount = len(s.UntrackedFiles())
	s.IsClean = len(s.Files) == 0

	return s
}

type branchInfo struct {
	branch   string
	upstream string
	ahead    int
	behind   int
}

// ParseBranchInfo parses the first 4 lines of `git status --porcelain=v2 --branch`.
func ParseBranchInfo(data []byte) *branchInfo {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	bi := &branchInfo{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "# ") {
			break
		}
		rest := strings.TrimPrefix(line, "# ")
		switch {
		case strings.HasPrefix(rest, "branch.head "):
			bi.branch = strings.TrimPrefix(rest, "branch.head ")
		case strings.HasPrefix(rest, "branch.upstream "):
			bi.upstream = strings.TrimPrefix(rest, "branch.upstream ")
		case strings.HasPrefix(rest, "branch.ab "):
			ab := strings.TrimPrefix(rest, "branch.ab ")
			parts := strings.Split(ab, " ")
			if len(parts) == 2 {
				bi.ahead, _ = strconv.Atoi(strings.TrimPrefix(parts[0], "+"))
				bi.behind, _ = strconv.Atoi(strings.TrimPrefix(parts[1], "-"))
			}
		}
	}

	if bi.branch == "" {
		return nil
	}
	return bi
}

func parsePorcelainV2Entry(line string) *models.FileStatus {
	// Split on space, but filename can contain spaces.
	// The first N fields are fixed-width (no spaces), everything after is the path.
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return nil
	}

	// fields[0] = record type (1 or 2)
	// fields[1] = XY status
	xy := fields[1]
	if len(xy) != 2 {
		return nil
	}

	fs := &models.FileStatus{
		Staged:   parseStatusByte(xy[0]),
		Unstaged: parseStatusByte(xy[1]),
	}

	// For type 2 (renamed/copied), there's an extra score field before the path.
	// Type 1: 8 fixed fields [0..7], path starts at field 8
	// Type 2: 9 fixed fields [0..8], path starts at field 9
	pathStart := 8
	if line[0] == porcelainV2Renamed {
		pathStart = 9
	}

	if len(fields) < pathStart+1 {
		return fs
	}

	// Reconstruct the original path from the line to preserve spaces and tabs.
	// The path portion starts after the pathStart-th space-separated field.
	// We count spaces in the original line to find where the path begins.
	spaceCount := 0
	pathBegin := 0
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			spaceCount++
			if spaceCount == pathStart {
				pathBegin = i + 1
				break
			}
		}
	}

	if pathBegin == 0 {
		return fs
	}
	pathPart := line[pathBegin:]

	// For renames: path part contains tab-separated oldPath\tnewPath
	if line[0] == porcelainV2Renamed {
		pathParts := strings.SplitN(pathPart, "\t", 2)
		if len(pathParts) == 2 {
			// oldPath newPath format from git
			// Actually porcelain v2 format: newPath\toldPath for staged renames
			fs.Path = pathParts[1]
			fs.OldPath = pathParts[0]
		} else {
			fs.Path = pathParts[0]
		}
	} else {
		fs.Path = pathPart
	}

	return fs
}

func parseStatusByte(b byte) models.StatusType {
	switch b {
	case ' ':
		return models.StatusUnmodified
	case 'M':
		return models.StatusModified
	case 'A':
		return models.StatusAdded
	case 'D':
		return models.StatusDeleted
	case 'R':
		return models.StatusRenamed
	case 'C':
		return models.StatusCopied
	case 'U':
		return models.StatusUpdatedButUnmerged
	case '?':
		return models.StatusUntracked
	case '!':
		return models.StatusIgnored
	default:
		return models.StatusUnmodified
	}
}

// --- git log format ---
// We use tab (%x09) delimiter for easy parsing.
// %H = hash, %P = parent hashes, %an = author name, %ae = author email, %at = author timestamp, %s = subject, %D = ref names
// Note: log format uses %x09 (not %09 — those are for for-each-ref format only).
const logFormat = "%H%x09%P%x09%an%x09%ae%x09%at%x09%s%x09%D"

// ParseLog parses `git log --format=...` output into commits.
func ParseLog(data []byte) ([]*models.Commit, error) {
	var commits []*models.Commit
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		commit, err := parseCommitLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse commit line: %w", err)
		}
		commits = append(commits, commit)
	}
	return commits, scanner.Err()
}

func parseCommitLine(line string) (*models.Commit, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 6 {
		return nil, fmt.Errorf("malformed commit line: %q", line)
	}

	ts, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse timestamp %q: %w", parts[4], err)
	}

	// Parse parent hashes (space-separated, or empty for root commits)
	parents := make([]string, 0)
	if parentsStr := parts[1]; parentsStr != "" {
		parents = strings.Split(parentsStr, " ")
	}

	c := &models.Commit{
		Hash:      parts[0],
		Parents:   parents,
		Author:    parts[2],
		Email:     parts[3],
		Timestamp: time.Unix(ts, 0),
		Message:   parts[5],
	}

	if len(parts) > 6 && parts[6] != "" {
		c.RefNames = parts[6]
	}

	return c, nil
}

// --- git branch format ---
// Delimiter: %09 (tab). Avoid %xNN hex codes — not supported in some git versions.
const branchFormat = "%(refname:short)%09%(HEAD)%09%(upstream:short)%09%(upstream:track,nobracket)%09%(objectname:short)%09%(subject)"

// ParseBranches parses `git branch --format=...` output.
func ParseBranches(data []byte) ([]*models.Branch, error) {
	var branches []*models.Branch
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		b, err := parseBranchLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse branch line: %w", err)
		}
		branches = append(branches, b)
	}
	return branches, scanner.Err()
}

func parseBranchLine(line string) (*models.Branch, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed branch line: %q", line)
	}

	name := parts[0]
	isHead := parts[1] == "*"

	// All branches from `git branch` are local refs.
	// Remote branches come from `git branch -r`.
	b := &models.Branch{
		Name:    name,
		IsHead:  isHead,
		Type:    models.LocalBranch,
		FullRef: "refs/heads/" + name,
	}

	if len(parts) > 2 && parts[2] != "" {
		b.Upstream = parts[2]
	}

	// Parse upstream track info: "ahead N, behind M" or "ahead N" or "behind M"
	if len(parts) > 3 && parts[3] != "" {
		track := parts[3]
		if strings.Contains(track, "ahead") {
			b.Ahead = parseTrackNumber(track, "ahead")
		}
		if strings.Contains(track, "behind") {
			b.Behind = parseTrackNumber(track, "behind")
		}
	}

	if len(parts) > 4 {
		b.LatestHash = parts[4]
	}
	if len(parts) > 5 {
		b.LatestMsg = parts[5]
	}

	return b, nil
}

// parseTrackNumber extracts the numeric value after the given keyword in a track string.
// Example: parseTrackNumber("ahead 3, behind 1", "ahead") → 3
func parseTrackNumber(track, keyword string) int {
	idx := strings.Index(track, keyword)
	if idx < 0 {
		return 0
	}
	rest := track[idx+len(keyword):]
	rest = strings.TrimSpace(rest)
	// Handle both "ahead 3" and "ahead 3, behind 1"
	parts := strings.Fields(rest)
	if len(parts) > 0 {
		n, err := strconv.Atoi(strings.TrimRight(parts[0], ","))
		if err == nil {
			return n
		}
	}
	return 0
}

// --- git diff numstat ---
// Format: `additions\tdeletions\tpath\0oldpath\0` for renames
// We parse standard `git diff --numstat` output.

// ParseDiffNumstat parses `git diff --numstat` output.
func ParseDiffNumstat(data []byte) ([]models.FileChange, error) {
	var files []models.FileChange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fc, err := parseNumstatLine(line)
		if err != nil {
			return nil, fmt.Errorf("parse numstat line: %w", err)
		}
		files = append(files, *fc)
	}
	return files, scanner.Err()
}

func parseNumstatLine(line string) (*models.FileChange, error) {
	// Handle tab-separated: adds\tdeletes\tpath
	// For renames/copies: adds\tdeletes\toldpath\tnewpath
	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return nil, fmt.Errorf("malformed numstat line: %q", line)
	}

	fc := &models.FileChange{}

	adds, err := strconv.Atoi(parts[0])
	if err != nil {
		adds = 0
	}
	dels, err := strconv.Atoi(parts[1])
	if err != nil {
		dels = 0
	}
	fc.Additions = adds
	fc.Deletions = dels

	if parts[0] == "-" && parts[1] == "-" {
		fc.IsBinary = true
	}

	if len(parts) == 4 {
		// rename
		fc.OldPath = parts[2]
		fc.NewPath = parts[3]
		fc.Type = models.FileRenamed
	} else {
		fc.NewPath = parts[2]
		fc.Type = models.FileModified
	}

	return fc, nil
}

// ParseRemotes parses `git remote -v` output.
func ParseRemotes(data []byte) ([]*models.Remote, error) {
	remotes := make([]*models.Remote, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// format: origin	git@github.com:user/repo.git (fetch)
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			r := &models.Remote{
				Name: parts[0],
				URL:  parts[1],
				Type: strings.Trim(parts[2], "()"),
			}
			remotes = append(remotes, r)
		}
	}
	return remotes, scanner.Err()
}

// ParseStashList parses `git stash list --format='%H%x09%gs'` output.
// Format: commitHash\tstashMessage (tab-delimited)
func ParseStashList(data []byte) ([]*models.Stash, error) {
	var stashes []*models.Stash
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		s := &models.Stash{
			Index:   len(stashes),
			Hash:    parts[0],
			Message: strings.TrimSpace(parts[1]),
		}
		stashes = append(stashes, s)
	}
	return stashes, scanner.Err()
}

// ParseRevList parses `git rev-list` output (one hash per line).
func ParseRevList(data []byte) ([]string, error) {
	var hashes []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		hashes = append(hashes, line)
	}
	return hashes, scanner.Err()
}
