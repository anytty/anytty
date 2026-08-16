package render

// PaneChromeGlyphs 集中管理 pane/floating chrome 的动作和状态字形。
// 默认 chrome 使用常见的单 cell Unicode；测试和用户配置仍可通过 SetPaneChromeGlyphs 覆盖。
type PaneChromeGlyphs struct {
	ActionLeft          string
	ActionLeftSet       bool
	ActionRight         string
	ActionRightSet      bool
	ActionSeparator     string
	ActionSeparatorSet  bool
	ActionGroupLeft     string
	ActionGroupLeftSet  bool
	ActionGroupRight    string
	ActionGroupRightSet bool
	OwnerLeft           string
	OwnerLeftSet        bool
	OwnerRight          string
	OwnerRightSet       bool
	Owner               string
	OwnerSet            bool
	OwnerPending        string
	OwnerPendingSet     bool
	TakeOwner           string
	TakeOwnerSet        bool
	Zoom                string
	// Unzoom 是 zoom 状态下同一个 pane.zoom toggle action 的展示 glyph。
	// ActionID 不变，renderer 只用它区分“进入 zoom”和“退出 zoom”的视觉状态。
	Unzoom           string
	SplitVertical    string
	SplitHorizontal  string
	Close            string
	SizeLock         string
	SizeUnlock       string
	CenterFloating   string
	CollapseFloating string
	Running          string
	Waiting          string
	Exited           string
	Killed           string
	// OverflowLeft/Right/Top/Bottom 是内容裁切提示的展示 glyph；裁切 truth 仍来自 ContentOverflow。
	OverflowLeft      string
	OverflowLeftSet   bool
	OverflowRight     string
	OverflowRightSet  bool
	OverflowTop       string
	OverflowTopSet    bool
	OverflowBottom    string
	OverflowBottomSet bool
	// OverflowStyle 只控制裁切提示颜色/样式，不参与内容窗口或 resize owner 判定。
	OverflowStyle    string
	OverflowStyleSet bool
	// ExtentPlaceholder 是 live surface 尺寸小于 pane 时用于占位的单元 glyph。
	ExtentPlaceholder         string
	ExtentPlaceholderSet      bool
	ExtentPlaceholderStyle    string
	ExtentPlaceholderStyleSet bool
}

var defaultPaneChromeGlyphs = PaneChromeGlyphs{
	ActionLeft:        "[",
	ActionRight:       "]",
	ActionSeparator:   "─",
	ActionGroupLeft:   "",
	ActionGroupRight:  "",
	OwnerLeft:         "",
	OwnerRight:        "",
	Owner:             "◆ owner",
	OwnerPending:      "◆ owner?",
	TakeOwner:         "◇ follow",
	Zoom:              "↗",
	Unzoom:            "↙",
	SplitVertical:     "│",
	SplitHorizontal:   "─",
	Close:             "×",
	SizeLock:          "■",
	SizeUnlock:        "□",
	CenterFloating:    "◎",
	CollapseFloating:  "▾",
	Running:           "●",
	Waiting:           "○",
	Exited:            "×",
	Killed:            "×",
	OverflowLeft:      "<",
	OverflowRight:     ">",
	OverflowTop:       "^",
	OverflowBottom:    "v",
	ExtentPlaceholder: "·",
	// 默认把 live extent 占位点降为 muted，避免它被误读成 terminal 正文。
	ExtentPlaceholderStyle: string(StyleMuted),
}

var paneChromeGlyphs = defaultPaneChromeGlyphs

func DefaultPaneChromeGlyphs() PaneChromeGlyphs {
	return defaultPaneChromeGlyphs
}

func SetPaneChromeGlyphs(glyphs PaneChromeGlyphs) {
	next := defaultPaneChromeGlyphs
	if glyphs.ActionLeftSet {
		next.ActionLeft = glyphs.ActionLeft
	}
	if glyphs.ActionRightSet {
		next.ActionRight = glyphs.ActionRight
	}
	if glyphs.ActionSeparatorSet {
		next.ActionSeparator = glyphs.ActionSeparator
	}
	if glyphs.ActionGroupLeftSet {
		next.ActionGroupLeft = glyphs.ActionGroupLeft
	}
	if glyphs.ActionGroupRightSet {
		next.ActionGroupRight = glyphs.ActionGroupRight
	}
	if glyphs.OwnerLeftSet {
		next.OwnerLeft = glyphs.OwnerLeft
	}
	if glyphs.OwnerRightSet {
		next.OwnerRight = glyphs.OwnerRight
	}
	if glyphs.OwnerSet {
		next.Owner = glyphs.Owner
	}
	if glyphs.OwnerPendingSet {
		next.OwnerPending = glyphs.OwnerPending
	}
	if glyphs.TakeOwnerSet {
		next.TakeOwner = glyphs.TakeOwner
	}
	if glyphs.Zoom != "" {
		next.Zoom = glyphs.Zoom
	}
	if glyphs.Unzoom != "" {
		next.Unzoom = glyphs.Unzoom
	}
	if glyphs.SplitVertical != "" {
		next.SplitVertical = glyphs.SplitVertical
	}
	if glyphs.SplitHorizontal != "" {
		next.SplitHorizontal = glyphs.SplitHorizontal
	}
	if glyphs.Close != "" {
		next.Close = glyphs.Close
	}
	if glyphs.SizeLock != "" {
		next.SizeLock = glyphs.SizeLock
	}
	if glyphs.SizeUnlock != "" {
		next.SizeUnlock = glyphs.SizeUnlock
	}
	if glyphs.CenterFloating != "" {
		next.CenterFloating = glyphs.CenterFloating
	}
	if glyphs.CollapseFloating != "" {
		next.CollapseFloating = glyphs.CollapseFloating
	}
	if glyphs.Running != "" {
		next.Running = glyphs.Running
	}
	if glyphs.Waiting != "" {
		next.Waiting = glyphs.Waiting
	}
	if glyphs.Exited != "" {
		next.Exited = glyphs.Exited
	}
	if glyphs.Killed != "" {
		next.Killed = glyphs.Killed
	}
	if glyphs.OverflowLeftSet {
		next.OverflowLeft = glyphs.OverflowLeft
		next.OverflowLeftSet = true
	}
	if glyphs.OverflowRightSet {
		next.OverflowRight = glyphs.OverflowRight
		next.OverflowRightSet = true
	}
	if glyphs.OverflowTopSet {
		next.OverflowTop = glyphs.OverflowTop
		next.OverflowTopSet = true
	}
	if glyphs.OverflowBottomSet {
		next.OverflowBottom = glyphs.OverflowBottom
		next.OverflowBottomSet = true
	}
	if glyphs.OverflowStyleSet {
		next.OverflowStyle = glyphs.OverflowStyle
		next.OverflowStyleSet = true
	}
	if glyphs.ExtentPlaceholderSet {
		next.ExtentPlaceholder = glyphs.ExtentPlaceholder
		next.ExtentPlaceholderSet = true
	}
	if glyphs.ExtentPlaceholderStyleSet {
		next.ExtentPlaceholderStyle = glyphs.ExtentPlaceholderStyle
		next.ExtentPlaceholderStyleSet = true
	}
	paneChromeGlyphs = next
}

func ResetPaneChromeGlyphs() {
	paneChromeGlyphs = defaultPaneChromeGlyphs
}

func paneChromeCloseGlyph() string {
	return paneChromeGlyphs.Close
}

func paneChromeSizeLockGlyph() string {
	return paneChromeGlyphs.SizeLock
}

func paneChromeSizeUnlockGlyph() string {
	return paneChromeGlyphs.SizeUnlock
}

func paneChromeZoomGlyph() string {
	return paneChromeGlyphs.Zoom
}

func paneChromeUnzoomGlyph() string {
	return paneChromeGlyphs.Unzoom
}

func paneChromeSplitVerticalGlyph() string {
	return paneChromeGlyphs.SplitVertical
}

func paneChromeSplitHorizontalGlyph() string {
	return paneChromeGlyphs.SplitHorizontal
}

func paneChromeFloatingCenterGlyph() string {
	return paneChromeGlyphs.CenterFloating
}

func paneChromeFloatingCollapseGlyph() string {
	return paneChromeGlyphs.CollapseFloating
}

func paneChromeRunningGlyph() string {
	return paneChromeGlyphs.Running
}

func paneChromeActionLeft() string {
	return paneChromeGlyphs.ActionLeft
}

func paneChromeActionRight() string {
	return paneChromeGlyphs.ActionRight
}

func paneChromeActionSeparator() string {
	return paneChromeGlyphs.ActionSeparator
}

func paneChromeActionGroupLeft() string {
	return paneChromeGlyphs.ActionGroupLeft
}

func paneChromeActionGroupRight() string {
	return paneChromeGlyphs.ActionGroupRight
}

func paneChromeOwnerLeft() string {
	return paneChromeGlyphs.OwnerLeft
}

func paneChromeOwnerRight() string {
	return paneChromeGlyphs.OwnerRight
}

func paneChromeOwnerText() string {
	return paneChromeGlyphs.Owner
}

func paneChromeOwnerPendingText() string {
	return paneChromeGlyphs.OwnerPending
}

func paneChromeTakeOwnerText() string {
	return paneChromeGlyphs.TakeOwner
}

func paneChromeOverflowLeftGlyph() string {
	return paneChromeGlyphs.OverflowLeft
}

func paneChromeOverflowRightGlyph() string {
	return paneChromeGlyphs.OverflowRight
}

func paneChromeOverflowTopGlyph() string {
	return paneChromeGlyphs.OverflowTop
}

func paneChromeOverflowBottomGlyph() string {
	return paneChromeGlyphs.OverflowBottom
}

func paneChromeOverflowStyle(fallback StyleToken) StyleToken {
	if paneChromeGlyphs.OverflowStyleSet {
		return StyleToken(paneChromeGlyphs.OverflowStyle)
	}
	return fallback
}

func paneChromeExtentPlaceholderGlyph() string {
	return paneChromeGlyphs.ExtentPlaceholder
}

func paneChromeExtentPlaceholderStyle() StyleToken {
	return StyleToken(paneChromeGlyphs.ExtentPlaceholderStyle)
}
