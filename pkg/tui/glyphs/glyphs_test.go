package glyphs

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// env builds a getenv func from a map.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParse(t *testing.T) {
	cases := map[string]Mode{
		"":          Auto,
		"auto":      Auto,
		"nerd":      Nerd,
		"nerdfont":  Nerd,
		"nerd-font": Nerd,
		"Nerd_Font": Nerd,
		"unicode":   Unicode,
		"utf-8":     Unicode,
		"safe":      Unicode,
		"ascii":     ASCII,
		"plain":     ASCII,
		"  nerd  ":  Nerd,
		"nonsense":  "",
	}
	for in, want := range cases {
		got, ok := Parse(in)
		if want == "" {
			if ok {
				t.Errorf("Parse(%q) = %v, want not-ok", in, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("Parse(%q) = %v/%v, want %v", in, got, ok, want)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want Mode
	}{
		{"bare terminal is safe", map[string]string{}, Unicode},
		{"term only is safe", map[string]string{"TERM": "xterm-256color"}, Unicode},
		{"kitty is not assumed", map[string]string{"TERM": "xterm-kitty"}, Unicode},
		{"vscode is not assumed", map[string]string{"TERM_PROGRAM": "vscode"}, Unicode},
		{"iterm is not assumed", map[string]string{"TERM_PROGRAM": "iTerm.app"}, Unicode},
		{"ghostty bundles symbols", map[string]string{"TERM_PROGRAM": "ghostty"}, Nerd},
		{"wezterm bundles symbols", map[string]string{"TERM_PROGRAM": "WezTerm"}, Nerd},
		{"explicit env signal", map[string]string{"NERD_FONT": "1"}, Nerd},
		{"alias env signal", map[string]string{"NERDFONT": "true"}, Nerd},
		{"user override beats the guess", map[string]string{"NERD_FONT": "1", EnvVar: "unicode"}, Unicode},
		{"user override to ascii", map[string]string{"TERM_PROGRAM": "ghostty", EnvVar: "ascii"}, ASCII},
		{"garbage override falls through", map[string]string{"NERD_FONT": "1", EnvVar: "wat"}, Nerd},
	}
	for _, c := range cases {
		if got := Detect(env(c.env)); got != c.want {
			t.Errorf("%s: Detect = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve(ASCII); got != ASCII {
		t.Errorf("explicit mode changed: %v", got)
	}
	if got := Resolve(""); got != Unicode {
		t.Errorf("empty mode with no env should be Unicode, got %v", got)
	}
	if got := Resolve("martian"); got != Unicode {
		t.Errorf("unknown mode should fall back to Unicode, got %v", got)
	}
}

// TestEverySetIsComplete keeps the three vocabularies in step: a mode
// that forgets a structural glyph would render ragged chrome.
func TestEverySetIsComplete(t *testing.T) {
	// Always drawn, in every mode.
	required := func(s Set) map[string]string {
		return map[string]string{
			"Rule": s.Rule, "VRule": s.VRule, "Bar": s.Bar, "Dash": s.Dash,
			"Ellipsis": s.Ellipsis, "Commit": s.Commit, "Lane": s.Lane,
			"Join": s.Join, "Dot": s.Dot, "Right": s.Right, "Up": s.Up, "Down": s.Down,
			"Check": s.Check, "Cross": s.Cross, "Warn": s.Warn, "Info": s.Info,
			"Staged": s.Staged, "Unstaged": s.Unstaged, "Untracked": s.Untracked,
			"Deleted": s.Deleted, "Draft": s.Draft,
		}
	}

	for _, m := range []Mode{Nerd, Unicode, ASCII} {
		s, ok := Sets[m]
		if !ok {
			t.Fatalf("mode %v has no set", m)
		}
		if s.Mode != m {
			t.Errorf("%v: Mode field is %v", m, s.Mode)
		}
		for name, glyph := range required(s) {
			if glyph == "" {
				t.Errorf("mode %v: %s is empty", m, name)
			}
			if strings.ContainsAny(glyph, "\n\r\t") {
				t.Errorf("mode %v: %s contains a control character", m, name)
			}
		}
	}

	// Decorative marks may be hidden in a font that has no safe symbol
	// for them (a box is worse than nothing) — but the icon set must
	// carry them all, since that is the whole point of nerd mode.
	for name, glyph := range map[string]string{
		"Branch": Sets[Nerd].Branch, "Tag": Sets[Nerd].Tag, "Head": Sets[Nerd].Head,
		"Stash": Sets[Nerd].Stash, "Worktree": Sets[Nerd].Worktree, "Star": Sets[Nerd].Star,
	} {
		if glyph == "" {
			t.Errorf("nerd mode: %s is empty", name)
		}
	}
	// Anything present in a safe mode must be a symbol that font family
	// carries: no empties that views then print as a gap mid-row.
	for _, m := range []Mode{Unicode, ASCII} {
		for name, glyph := range map[string]string{
			"Head": Sets[m].Head, "Star": Sets[m].Star, "Tag": Sets[m].Tag,
		} {
			if glyph == "" {
				t.Errorf("mode %v: %s must have a plain-text fallback", m, name)
			}
		}
	}
}

// TestASCIISetIsASCII pins the promise of the fallback set.
func TestASCIISetIsASCII(t *testing.T) {
	const asciiMax = 127
	for name, glyph := range map[string]string{
		"Rule": Sets[ASCII].Rule, "Bar": Sets[ASCII].Bar, "Commit": Sets[ASCII].Commit,
		"Merge": Sets[ASCII].Merge, "Check": Sets[ASCII].Check, "Cross": Sets[ASCII].Cross,
		"Ellipsis": Sets[ASCII].Ellipsis, "Dash": Sets[ASCII].Dash,
		"Right": Sets[ASCII].Right, "Up": Sets[ASCII].Up, "Down": Sets[ASCII].Down,
		"Staged": Sets[ASCII].Staged, "Untracked": Sets[ASCII].Untracked,
	} {
		for _, r := range glyph {
			if r > asciiMax {
				t.Errorf("ascii set: %s = %q is not ASCII", name, glyph)
			}
		}
	}
}

// TestNerdSetHasSeparators keeps the Powerline pair honest: both ends or
// neither, or segment rendering brackets itself wrongly.
func TestNerdSetHasSeparators(t *testing.T) {
	n := Sets[Nerd]
	if (n.PillLeft == "") != (n.PillRight == "") {
		t.Fatalf("nerd set has one separator: left=%q right=%q", n.PillLeft, n.PillRight)
	}
	if n.PillRight == "" {
		t.Fatal("nerd set should carry Powerline separators")
	}
	if utf8.RuneCountInString(n.PillRight) != 1 {
		t.Errorf("separator must be one rune: %q", n.PillRight)
	}
	// The safe sets must not use Powerline separators — unpatched fonts
	// render them as boxes.
	for _, m := range []Mode{Unicode, ASCII} {
		if Sets[m].PillRight != "" || Sets[m].PillLeft != "" {
			t.Errorf("mode %v must not use Powerline separators", m)
		}
	}
}

// TestUseAndCurrent checks the process-wide selection.
func TestUseAndCurrent(t *testing.T) {
	defer Use(Unicode)
	Use(ASCII)
	if got := Current(); got.Mode != ASCII {
		t.Fatalf("Current = %v", got.Mode)
	}
	Use(Auto)
	if Current().Mode == "" {
		t.Fatal("Auto must resolve to a concrete mode")
	}
}
