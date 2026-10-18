package styles

import (
	"fmt"
	"testing"
)

// TestThemeSwitchRepaintsStyles pins the contract the TUI relies on: after
// SetLight/SetDark + Refresh, every prebuilt style points at the new
// tokens. Before Refresh existed these styles froze their colors at init
// and a theme change left the UI half-recolored.
func TestThemeSwitchRepaintsStyles(t *testing.T) {
	SetDark()
	Refresh()
	darkFg := fmt.Sprintf("%v", StatusBarStyle.GetForeground())
	darkBg := fmt.Sprintf("%v", ListItemSelectedStyle.GetBackground())
	darkBase := Base

	SetLight()
	Refresh()
	lightFg := fmt.Sprintf("%v", StatusBarStyle.GetForeground())
	lightBg := fmt.Sprintf("%v", ListItemSelectedStyle.GetBackground())

	if darkFg == lightFg {
		t.Fatalf("status bar foreground did not follow the theme (%s)", darkFg)
	}
	if darkBg == lightBg {
		t.Fatalf("selected row background did not follow the theme (%s)", darkBg)
	}
	if Base == darkBase {
		t.Fatalf("Base token did not change")
	}
	if ActivePanelBorderColor != Accent {
		t.Fatalf("active panel border should follow the accent token")
	}
	if InactivePanelBorderColor != Overlay0 {
		t.Fatalf("inactive panel border should follow the muted token")
	}

	SetDark()
	Refresh()
}

// TestThemeTokensStayLegible guards the light palette against the usual
// mistakes: a background token reused as a foreground, or a dark literal
// left behind in a style.
func TestThemeTokensStayLegible(t *testing.T) {
	for _, theme := range []struct {
		name string
		set  func()
		dark bool
	}{
		{"dark", SetDark, true},
		{"light", SetLight, false},
	} {
		theme.set()
		Refresh()

		if Dark != theme.dark {
			t.Fatalf("%s: Dark flag = %v", theme.name, Dark)
		}
		if Success != Green || Warning != Yellow || Danger != Red || Info != Blue || Accent != Mauve {
			t.Fatalf("%s: semantic aliases drifted from their tokens", theme.name)
		}
		if DiffAddBg == DiffDelBg {
			t.Fatalf("%s: diff backgrounds collapsed", theme.name)
		}
		if fmt.Sprintf("%v", DiffAddStyle.GetBackground()) != fmt.Sprintf("%v", DiffAddBg) {
			t.Fatalf("%s: diff add style not rebuilt from tokens", theme.name)
		}
		if fmt.Sprintf("%v", DiffDelStyle.GetBackground()) != fmt.Sprintf("%v", DiffDelBg) {
			t.Fatalf("%s: diff delete style not rebuilt from tokens", theme.name)
		}
		if Selection == Text {
			t.Fatalf("%s: selection equals text color", theme.name)
		}
		// No style may keep a fixed 80-column width: it only ever fits an
		// 80-column terminal.
		if w := AppStyle.GetWidth(); w != 0 {
			t.Fatalf("%s: AppStyle has a fixed width of %d", theme.name, w)
		}
		if w := StatusBarStyle.GetWidth(); w != 0 {
			t.Fatalf("%s: StatusBarStyle has a fixed width of %d", theme.name, w)
		}
	}
	SetDark()
	Refresh()
}
