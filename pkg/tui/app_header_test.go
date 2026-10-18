package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/shell"
	"github.com/zaidejjo/zgit/pkg/tui/styles"
)

func newHeaderModel(t *testing.T, width int, user string) *Model {
	t.Helper()
	gitExec, err := gitpkg.NewNativeExec("")
	if err != nil {
		t.Skip("git not available")
	}
	m := NewModel(gitExec)
	m.ready = true
	m.width = width
	m.status.Status = &models.Status{Branch: "dev"}
	m.ghUser = user
	AppVersion = "9.9.9-test"
	return m
}

// TestRenderHeaderWide checks the full-detail strip on a terminal with
// room for everything.
func TestRenderHeaderWide(t *testing.T) {
	m := newHeaderModel(t, 200, "octo")

	out := m.renderHeader()
	for _, want := range []string{"zgit", "Status", "Log", "Branches", "PRs", "Issues", "@octo", "dev", "v9.9.9-test"} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q in %q", want, out)
		}
	}
	if w := lipgloss.Width(out); w > 200 {
		t.Errorf("header is %d cells wide, limit 200", w)
	}

	// Fullscreen PR tab highlights.
	m.fullScreenView = shell.PRs
	out = m.renderHeader()
	if !strings.Contains(out, "PRs") {
		t.Errorf("header missing PRs tab")
	}
}

// TestRenderHeaderAdapts is the invariant that matters: whatever the
// terminal width, the header fits it. Detail is dropped instead of
// wrapping the layout.
func TestRenderHeaderAdapts(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100, 120, 149, 200} {
		m := newHeaderModel(t, width, "octocat")
		out := m.renderHeader()
		if w := lipgloss.Width(out); w > width {
			t.Errorf("width %d: header is %d cells (%q)", width, w, out)
		}
		if strings.Contains(out, "\n") {
			t.Errorf("width %d: header must be a single line", width)
		}
		// The logo and the panel tabs are the last things to go.
		if !strings.Contains(out, "zgit") {
			t.Errorf("width %d: header lost the logo: %q", width, out)
		}
	}
}

// TestRenderHeaderNarrowShrinks verifies the density ladder actually
// degrades rather than clipping mid-tab.
func TestRenderHeaderNarrowShrinks(t *testing.T) {
	m := newHeaderModel(t, 200, "octocat")
	wide := m.renderHeader()

	m.width = 60
	narrow := m.renderHeader()

	if lipgloss.Width(narrow) >= lipgloss.Width(wide) {
		t.Fatalf("narrow header (%d) should be shorter than wide (%d)", lipgloss.Width(narrow), lipgloss.Width(wide))
	}
	if w := lipgloss.Width(narrow); w > 60 {
		t.Fatalf("narrow header is %d cells, limit 60: %q", w, narrow)
	}
	styles.SetDark()
	styles.Refresh()
}

func TestConfirmTwice(t *testing.T) {
	m := &Model{}
	if m.confirmTwice("del:x") {
		t.Fatalf("first press should arm, not confirm")
	}
	if !m.confirmTwice("del:x") {
		t.Fatalf("second press should confirm")
	}
	if m.confirmTwice("del:x") {
		t.Fatalf("after confirm, next press re-arms")
	}
	if m.confirmTwice("del:y") {
		t.Fatalf("different action should arm separately")
	}
}
