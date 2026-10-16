package tui

import (
	"strings"
	"testing"

	gitpkg "github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
	"github.com/zaidejjo/zgit/pkg/tui/shell"
)

func TestRenderHeader(t *testing.T) {
	gitExec, err := gitpkg.NewNativeExec("")
	if err != nil {
		t.Skip("git not available")
	}
	m := NewModel(gitExec)
	m.width = 120
	m.status.Status = &models.Status{Branch: "dev"}
	m.ghUser = "octo"
	AppVersion = "9.9.9-test"

	out := m.renderHeader()
	for _, want := range []string{"zgit", "Status", "Log", "Branches", "PRs", "Issues", "@octo", "dev", "v9.9.9-test"} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q in %q", want, out)
		}
	}

	// Fullscreen PR tab highlights.
	m.fullScreenView = shell.PRs
	out = m.renderHeader()
	if !strings.Contains(out, "PRs") {
		t.Errorf("header missing PRs tab")
	}
}
