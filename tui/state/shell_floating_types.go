package state

type FloatingOverviewItem struct {
	FloatingID string
	Title      string
	PaneID     string
	PaneKind   PaneKind
	TerminalID string
	State      string
	Cols       int
	Rows       int
	Rect       FloatingRect
	Z          int
	Active     bool
	Collapsed  bool
	FitMode    FloatingFitMode
	Selected   bool
}

type FloatingCommandAction string

const (
	FloatingCommandCreate         FloatingCommandAction = "floating.create"
	FloatingCommandFocusRaise     FloatingCommandAction = "floating.focus-raise"
	FloatingCommandDeactivate     FloatingCommandAction = "floating.deactivate"
	FloatingCommandClose          FloatingCommandAction = "floating.close"
	FloatingCommandCenter         FloatingCommandAction = "floating.center"
	FloatingCommandToggleCollapse FloatingCommandAction = "floating.toggle-collapse"
	FloatingCommandSummon         FloatingCommandAction = "floating.summon"
	FloatingCommandMove           FloatingCommandAction = "floating.move"
	FloatingCommandPosition       FloatingCommandAction = "floating.position"
	FloatingCommandResize         FloatingCommandAction = "floating.resize"
	FloatingCommandToggleAll      FloatingCommandAction = "floating.toggle-all"
	FloatingCommandShowAll        FloatingCommandAction = "floating.show-all"
	FloatingCommandCollapseAll    FloatingCommandAction = "floating.collapse-all"
	FloatingCommandFit            FloatingCommandAction = "floating.fit"
	FloatingCommandToggleAutoFit  FloatingCommandAction = "floating.toggle-auto-fit"
	FloatingCommandRefreshAutoFit FloatingCommandAction = "floating.refresh-auto-fit"
)

type FloatingCommand struct {
	Action    FloatingCommandAction
	TargetID  string
	Pane      PaneState
	Title     string
	Rect      FloatingRect
	DeltaX    int
	DeltaY    int
	DeltaW    int
	DeltaH    int
	PositionX string
	PositionY string
	Index     int
	BoundsW   int
	BoundsH   int
	Source    PaneCommandSource
	FitCols   int
	FitRows   int
}

type FloatingCommandStatus string

const (
	FloatingCommandOK      FloatingCommandStatus = "ok"
	FloatingCommandInvalid FloatingCommandStatus = "invalid"
)

type FloatingCommandResult struct {
	Status FloatingCommandStatus
	Action FloatingCommandAction
	Reason string
	ID     string
}
