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
	// PropDimmed asks the component to render an unfocused panel with a gray
	// dim layer while preserving terminal text and geometry.
	PropDimmed = "chrome.dim"

	// Content framing props (program-declared). The terminal source has ONE
	// authoritative extent (the PTY cols/rows); the program's view-local layout
	// positions that extent inside the content area without resizing the PTY:
	//   content.offset = "x,y"  the extent origin (display cells) relative to
	//                           the content area. Content cell (row,col) shows
	//                           screen cell (row-y, col-x); x/y may be negative
	//                           (the screen is clipped at the top/left).
	//   content.size   = "cols,rows"  the extent footprint inside the content
	//                           area. Cells in [y,y+rows)x[x,x+cols) are the
	//                           terminal; cells outside get the placeholder.
	//   chrome.placeholder  explicit style for the outside-footprint fill.
	// Absent props are inert: offset (0,0), no placeholder, today's output.
	PropContentOffset = "content.offset"
	PropContentSize   = "content.size"
	PropPlaceholder   = "chrome.placeholder"

	// Copy-mode overlay props (program-declared). The copy scene is a program
	// state machine over the terminal text; the host paints its state:
	//   copy.cursor = "row,col"                     (viewport cell)
	//   copy.sel    = "row,c1,c2;row,c1,c2;..."     (inclusive columns)
	//   copy.match  = "row,c1,c2;..."               (search matches)
	//   copy.style.cursor / copy.style.sel / copy.style.match are explicit
	//   styles; absent styles fall back to the built-in tokens.
	PropCopyCursor         = "copy.cursor"
	PropCopySelection      = "copy.sel"
	PropCopyMatch          = "copy.match"
	PropCopyMatchCurrent   = "copy.match_current"
	PropCopyStyleCursor    = "copy.style.cursor"
	PropCopyStyleSelection = "copy.style.sel"
	PropCopyStyleMatch     = "copy.style.match"
	PropCopyStyleMatchCur  = "copy.style.match_cur"
)

// Props are the declarative inputs the host pushes from the view: title,
// focus, lifecycle badge, scroll badge, chrome inset, inactive-panel dim state
// and the program-declared chrome styles.
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
	Dimmed bool
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

// DimmedFromProps parses the optional inactive-panel marker.
func DimmedFromProps(chrome map[string]string) bool {
	value := strings.ToLower(strings.TrimSpace(chrome[PropDimmed]))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// ContentOffset is the program-declared extent origin inside the content area
// plus the extent footprint size (cols/rows). The zero value (offset 0,0 with
// no size) is inert: the component renders the screen from its top-left and the
// footprint is taken to be the screen size.
type ContentOffset struct {
	X, Y  int
	Cols  int
	Rows  int
	Sized bool // a content.size prop was present
}

// ContentOffsetFromProps parses content.offset ("x,y") and content.size
// ("cols,rows"). A missing or malformed value leaves the corresponding field
// unset, so a default props set stays byte-identical to the legacy render.
func ContentOffsetFromProps(chrome map[string]string) ContentOffset {
	var offset ContentOffset
	if value, ok := chrome[PropContentOffset]; ok {
		if x, y, ok := parsePair(value); ok {
			offset.X, offset.Y = x, y
		}
	}
	if value, ok := chrome[PropContentSize]; ok {
		if cols, rows, ok := parsePair(value); ok && cols >= 0 && rows >= 0 {
			offset.Cols, offset.Rows, offset.Sized = cols, rows, true
		}
	}
	return offset
}

// FramingFromProps returns the content.offset shift the render applies and
// whether it is non-trivial. A zero offset is inert and keeps any program cursor
// untouched; a non-zero offset means the host must shift the PTY cursor by the
// same amount (and hide it when it leaves the content area).
func FramingFromProps(chrome map[string]string) (dx, dy int, shifted bool) {
	parsed := ContentOffsetFromProps(chrome)
	return parsed.X, parsed.Y, parsed.X != 0 || parsed.Y != 0
}

// parsePair parses "a,b" into two ints.
func parsePair(value string) (int, int, bool) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	b, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return a, b, true
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
