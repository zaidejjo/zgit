package shell

import "testing"

func TestRouteTable(t *testing.T) {
	if len(Tabs) != 5 {
		t.Fatalf("tabs = %d, want 5", len(Tabs))
	}
	seenKeys := map[string]bool{}
	seenIDs := map[ScreenID]bool{}
	for _, tab := range Tabs {
		if tab.Key == "" || tab.Label == "" {
			t.Errorf("tab %+v needs key and label", tab)
		}
		if seenKeys[tab.Key] {
			t.Errorf("duplicate key %q", tab.Key)
		}
		seenKeys[tab.Key] = true
		if seenIDs[tab.ID] {
			t.Errorf("duplicate id %d", tab.ID)
		}
		seenIDs[tab.ID] = true
		if !Known(tab.ID) {
			t.Errorf("Known(%d) = false", tab.ID)
		}
		if tab.Full() != tab.Key+" "+tab.Label {
			t.Errorf("Full() = %q", tab.Full())
		}
	}
	if ByKey("5") == nil || ByKey("5").ID != Dashboard {
		t.Errorf("ByKey(5) should be Dashboard")
	}
	if ByKey("9") != nil {
		t.Errorf("ByKey(9) should be nil")
	}
	if Known(Panels) {
		t.Errorf("Panels is not a fullscreen tab")
	}
}

func TestComputeLayout(t *testing.T) {
	l := Compute(120, 30, false)
	if l.ContentHeight != 28 {
		t.Errorf("content = %d, want 28", l.ContentHeight)
	}
	if l.LeftWidth+l.RightWidth != 120 {
		t.Errorf("widths %d+%d != 120", l.LeftWidth, l.RightWidth)
	}
	if l.LeftWidth != 48 || l.RightWidth != 72 {
		t.Errorf("split = %d/%d, want 48/72", l.LeftWidth, l.RightWidth)
	}
	if l.BP != Standard {
		t.Errorf("bp = %d, want Standard", l.BP)
	}

	l = Compute(120, 30, true)
	if l.SidebarWidth != 42 || l.LeftWidth+l.RightWidth != 78 {
		t.Errorf("sidebar split wrong: %+v", l)
	}
	if BreakpointFor(80) != Compact || BreakpointFor(200) != Wide {
		t.Errorf("breakpoints wrong")
	}
	// Tiny terminal never goes negative.
	l = Compute(40, 10, false)
	if l.LeftWidth < 4 || l.RightWidth < 4 || l.ContentHeight < 4 {
		t.Errorf("floors violated: %+v", l)
	}
}
