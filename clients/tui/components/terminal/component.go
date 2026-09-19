package terminal

import (
	"errors"
	"maps"
	"strconv"
	"strings"

	"github.com/anytty/anytty/clients/tui/render"
)

// DefaultInset is the chrome inset (border thickness) this component draws
// when no explicit inset is declared. The kernel has no border concept:
// components own their chrome and declare the inset the runtime must respect
// for PTY winsize and cursor mapping (PROTOCOL §5, ARCHITECTURE §2.5).
const DefaultInset = 1

// Chrome prop keys this component interprets (program-declared
// content.props). Every value is an explicit, theme-free style string
// ("fg:#RRGGBB;bg:#RRGGBB;bold;dim;...") the host translates verbatim. A
// missing key falls back to the component built-in default; unknown keys are
// ignored.
const (
	// PropBorder is the border color of an unfocused, live terminal.
	PropBorder = "chrome.border"
	// PropTitle is the color of the border title.
	PropTitle = "chrome.title"
	// PropBorderFocus is the border color while the terminal is focused.
	PropBorderFocus = "chrome.border_focus"
	// PropBorderDead is the border color after the terminal exited.
	PropBorderDead = "chrome.border_dead"
	// PropBadge is the color of the title badges ([exited N], [↑N]).
	PropBadge = "chrome.badge"
	// PropInset declares the chrome inset explicitly; "0" requests a
	// borderless terminal (card-style panes draw the frame themselves). Any
	// other non-negative integer pins the inset; invalid values fall back to
	// the default.
	PropInset = "chrome.inset"
)

// Props are the declarative inputs the host pushes from the view: title,
// focus, lifecycle badge, scroll badge, chrome inset and the program-declared
// chrome styles.
type Props struct {
	Title        string
	Focused      bool
	Exited       bool
	ExitCode     int
	Scrolled     bool
	ScrollOffset int
	Inset        int
	// InsetSet distinguishes "no explicit inset" (Inset 0 means the default
	// border) from an explicit chrome.inset=0 borderless request.
	InsetSet bool
	// Chrome are the program-declared chrome styles (content.props). The
	// component interprets the keys it knows (PropBorder, PropTitle,
	// PropBorderFocus, PropBorderDead, PropBadge) and ignores the rest.
	Chrome map[string]string
}

// Inset reports the chrome inset the component draws at the given box size:
// the declared inset when the box is large enough to carry the border, 0
// when it is too small (content only). The runtime uses this single source
// of truth for PTY winsize and cursor remapping instead of assuming a fixed
// "width-2/height-2".
func (c *Component) Inset(width, height int) int {
	inset := c.props.Inset
	if inset <= 0 && !c.props.InsetSet {
		inset = DefaultInset
	}
	if inset == 0 {
		return 0
	}
	if width < 2*inset+1 || height < 2*inset+1 {
		return 0
	}
	return inset
}

// InsetFromProps parses the chrome.inset declaration from program props. It
// returns ok=false when the key is absent or invalid (caller keeps the
// default border); ok=true with value 0 requests a borderless component.
func InsetFromProps(chrome map[string]string) (int, bool) {
	value, ok := chrome[PropInset]
	if !ok {
		return 0, false
	}
	inset, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || inset < 0 {
		return 0, false
	}
	return inset, true
}

func (p Props) withDefaults() Props {
	if p.Inset <= 0 && !p.InsetSet {
		p.Inset = DefaultInset
	}
	if p.ScrollOffset < 0 {
		p.ScrollOffset = 0
	}
	p.Chrome = maps.Clone(p.Chrome)
	return p
}

// Source is the authoritative lifecycle snapshot of one terminal source as
// carried by a sources event (PROTOCOL §5, §9.2). The component must treat
// it as a full snapshot and reconcile by id.
type Source struct {
	ID       string
	Title    string
	Attached bool
	Exited   bool
	ExitCode int
	Health   string
}

// HistoryPort fetches scrollback rows. Window returns at most rows lines
// ending offset lines before the live bottom (offset 0 would be live; the
// component only calls it with offset > 0). The runtime injects the real
// history store behind this port.
type HistoryPort interface {
	Window(offset, rows int) ([]string, error)
}

// ClipboardPort writes text to the clipboard of the connection (view) that
// owns this component. The runtime implements it with OSC 52.
type ClipboardPort interface {
	Write(text string) error
}

// Errors returned by the component capability stubs when a port is missing.
var (
	// ErrNoHistory means scroll was requested without a history port.
	ErrNoHistory = errors.New("terminal: no history port")
	// ErrNoClipboard means copy was requested without a clipboard port.
	ErrNoClipboard = errors.New("terminal: no clipboard port")
)

// errNoHistoryRows is internal: the history store is empty above the live
// bottom, so a scroll request cannot move anywhere.
var errNoHistoryRows = errors.New("terminal: no history rows")

// Component is the terminal component model. It is not safe for concurrent
// use; the runtime serializes view updates on its own goroutine.
type Component struct {
	props   Props
	source  Source
	live    Screen
	window  Screen
	offset  int
	visible int
	history HistoryPort
	clip    ClipboardPort
}

// New creates a terminal component with injected data ports (either may be
// nil; the corresponding capability then fails with a typed error).
func New(history HistoryPort, clipboard ClipboardPort) *Component {
	return &Component{history: history, clip: clipboard}
}

// Props returns the current declarative state.
func (c *Component) Props() Props { return c.props }

// SetProps replaces the declarative state pushed by the host. When the host
// declares a scroll offset the component best-effort materializes the window
// through the history port so the rendered rows and the badge agree; without
// a port (or with empty history) the badge stays and the view falls back to
// the live screen.
func (c *Component) SetProps(props Props) {
	c.props = props.withDefaults()
	switch {
	case c.props.ScrollOffset <= 0:
		c.offset = 0
		c.window = Screen{}
		c.props.Scrolled = false
	case c.history == nil:
		c.offset = c.props.ScrollOffset
		c.props.Scrolled = true
	default:
		if err := c.loadWindow(c.props.ScrollOffset); err != nil {
			c.offset = c.props.ScrollOffset
			c.props.Scrolled = true
		}
	}
}

// Source returns the last observed sources snapshot.
func (c *Component) Source() Source { return c.source }

// ApplySource reconciles the component with a sources snapshot: lifecycle
// (attached/exited) and title fallback come from the host, never from the
// program. An exited terminal leaves scrollback and returns to live.
func (c *Component) ApplySource(source Source) {
	c.source = source
	c.props.Exited = source.Exited
	c.props.ExitCode = source.ExitCode
	if c.props.Title == "" && source.Title != "" {
		c.props.Title = source.Title
	}
	if source.Exited {
		c.effectiveScrollEnd()
	}
}

// SetScreen replaces the live screen snapshot.
func (c *Component) SetScreen(screen Screen) {
	c.live = screen
}

// Offset returns the current scrollback offset in lines (0 = live).
func (c *Component) Offset() int { return c.offset }

// Scroll moves the view delta lines into history (positive = older). It
// returns the resulting offset. delta 0 is a no-op. Scrolling to offset 0
// is equivalent to ScrollEnd. Without a history port the offset is left
// untouched and ErrNoHistory is returned.
func (c *Component) Scroll(delta int) (int, error) {
	next := c.offset + delta
	if next <= 0 {
		c.ScrollEnd()
		return 0, nil
	}
	err := c.loadWindow(next)
	if errors.Is(err, errNoHistoryRows) {
		return c.offset, nil
	}
	if err != nil {
		return c.offset, err
	}
	return c.offset, nil
}

// loadWindow fetches and installs the history window for an offset. It is
// the only place that mutates scroll state; errNoHistoryRows means the
// store cannot scroll further (the current window is kept).
func (c *Component) loadWindow(offset int) error {
	if c.history == nil {
		return ErrNoHistory
	}
	rows := c.visible
	if rows <= 0 {
		rows = 1
	}
	lines, err := c.history.Window(offset, rows)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return errNoHistoryRows
	}
	c.offset = offset
	c.window = ScreenFromText(lines, render.TokenDefault)
	c.syncScrollProps()
	return nil
}

// ScrollEnd returns the view to live and drops the history window.
func (c *Component) ScrollEnd() {
	c.effectiveScrollEnd()
}

func (c *Component) effectiveScrollEnd() {
	c.offset = 0
	c.window = Screen{}
	c.syncScrollProps()
}

func (c *Component) syncScrollProps() {
	c.props.Scrolled = c.offset > 0
	c.props.ScrollOffset = c.offset
}

// Copy writes the currently visible content rows to the clipboard port
// (PROTOCOL §9.3: default selection = what the view shows now, live or
// scrollback). Trailing spaces and trailing blank lines are trimmed.
func (c *Component) Copy() error {
	if c.clip == nil {
		return ErrNoClipboard
	}
	return c.clip.Write(strings.Join(c.VisibleLines(), "\n"))
}

// VisibleLines returns the visible content rows as trimmed plain text.
func (c *Component) VisibleLines() []string {
	screen := c.visibleScreen()
	lines := screen.TextLines()
	if c.visible > 0 && len(lines) > c.visible {
		lines = lines[:c.visible]
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, strings.TrimRight(line, " "))
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func (c *Component) visibleScreen() Screen {
	if c.offset > 0 && len(c.window.Lines) > 0 {
		return c.window
	}
	return c.live
}
