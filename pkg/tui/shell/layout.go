package shell

// Breakpoint selects the layout density for a terminal width.
type Breakpoint int

const (
	// Compact fits 80-column terminals: single focused panel.
	Compact Breakpoint = iota
	// Standard is today's two-column layout.
	Standard
	// Wide adds room for detail docks and wider panels.
	Wide
)

// BreakpointFor classifies a terminal width.
func BreakpointFor(width int) Breakpoint {
	switch {
	case width < 100:
		return Compact
	case width < 150:
		return Standard
	default:
		return Wide
	}
}

// Layout carries every computed dimension. Arithmetic mirrors the
// legacy calcLayout exactly; breakpoints only label the result for
// now (consumers reflow in later phases).
type Layout struct {
	Width         int
	Height        int
	BP            Breakpoint
	ContentHeight int
	SidebarWidth  int
	LeftWidth     int
	RightWidth    int
}

// Compute derives panel geometry. sidebarActive mirrors
// aiData.Sidebar.Active(); width/height are terminal dimensions.
func Compute(width, height int, sidebarActive bool) Layout {
	l := Layout{Width: width, Height: height, BP: BreakpointFor(width)}

	const statusBarH = 1
	const headerH = 1
	l.ContentHeight = height - statusBarH - headerH
	if l.ContentHeight < 4 {
		l.ContentHeight = 4
	}

	l.SidebarWidth = 0
	if sidebarActive {
		l.SidebarWidth = 42 // matches Width(42) in sidebar View
	}

	availW := width - l.SidebarWidth
	if availW < 54 {
		availW = 54 // minimum viable panel width
		l.SidebarWidth = width - availW
		if l.SidebarWidth < 0 {
			l.SidebarWidth = 0
		}
	}

	l.LeftWidth = int(float64(availW)*0.4 + 0.5)
	l.RightWidth = availW - l.LeftWidth

	// Minimum widths
	if l.LeftWidth < 24 {
		l.LeftWidth = 24
		l.RightWidth = width - l.LeftWidth
	}
	if l.RightWidth < 30 {
		l.RightWidth = 30
		l.LeftWidth = width - l.RightWidth
	}
	if l.LeftWidth < 24 {
		l.LeftWidth = 24
		l.RightWidth = width - l.LeftWidth
	}
	if l.LeftWidth < 4 {
		l.LeftWidth = 4
	}
	if l.RightWidth < 4 {
		l.RightWidth = 4
	}
	return l
}
