package views

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	out := RenderMarkdown("# Title\n\n- one\n- two\n\n`code` **bold**", 80)
	if !strings.Contains(out, "Title") || !strings.Contains(out, "one") {
		t.Errorf("missing content in %q", out)
	}
	// Memoized second call must return identical output.
	if again := RenderMarkdown("# Title\n\n- one\n- two\n\n`code` **bold**", 80); again != out {
		t.Errorf("memo mismatch")
	}
	// Empty input stays empty; unparseable input passes through.
	if got := RenderMarkdown("   ", 80); got != "" {
		t.Errorf("empty = %q", got)
	}
}
