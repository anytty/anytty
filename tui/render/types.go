package render

// Frame 是 FrameSink 使用的线性输出适配结果。
//
// Render framework 的主输出是 RenderResult；Frame 只保留给现有
// TerminalHost、测试和 CLI smoke 适配层使用。
type Frame struct {
	Lines       []string
	StyledLines []Line
	ANSILines   []string
	Patch       *FramePatch
	Cursor      Cursor
	CursorRect  Rect
	Blink       bool
	HitRegions  []HitRegion
	LiveTargets []LiveRenderTarget
	Metadata    RenderMetadata
	Theme       Theme
}

// Clone 返回 detached frame，防止测试或 host 共享修改 frame lines。
func (frame Frame) Clone() Frame {
	cloned := Frame{
		Cursor:     frame.Cursor,
		CursorRect: frame.CursorRect,
		Blink:      frame.Blink,
		Metadata:   frame.Metadata,
		Theme:      frame.Theme,
	}
	if len(frame.Lines) > 0 {
		cloned.Lines = cloneStrings(frame.Lines)
	}
	if len(frame.ANSILines) > 0 {
		cloned.ANSILines = cloneStrings(frame.ANSILines)
	}
	if len(frame.StyledLines) > 0 {
		cloned.StyledLines = make([]Line, len(frame.StyledLines))
		for i, line := range frame.StyledLines {
			cloned.StyledLines[i] = line.Clone()
		}
	}
	if len(frame.HitRegions) > 0 {
		cloned.HitRegions = cloneHitRegions(frame.HitRegions)
	}
	if len(frame.LiveTargets) > 0 {
		cloned.LiveTargets = append([]LiveRenderTarget(nil), frame.LiveTargets...)
	}
	if frame.Patch != nil {
		patch := *frame.Patch
		if len(frame.Patch.LinesANSI) > 0 {
			patch.LinesANSI = cloneStrings(frame.Patch.LinesANSI)
		}
		cloned.Patch = &patch
	}
	return cloned
}

func FrameFromRenderResult(result RenderResult) Frame {
	return Frame{
		Lines:       result.Lines(),
		StyledLines: result.Content,
		ANSILines:   result.ANSILines(),
		Cursor:      result.Cursor,
		CursorRect:  result.CursorRect,
		Blink:       result.Blink,
		HitRegions:  cloneHitRegions(result.HitRegions),
		LiveTargets: append([]LiveRenderTarget(nil), result.LiveTargets...),
		Metadata:    result.Metadata,
		Theme:       result.Theme.WithFallback(),
	}
}

func ANSIFrameFromRenderResult(result RenderResult) Frame {
	return Frame{
		ANSILines:   result.ANSILines(),
		Cursor:      result.Cursor,
		CursorRect:  result.CursorRect,
		Blink:       result.Blink,
		HitRegions:  cloneHitRegions(result.HitRegions),
		LiveTargets: append([]LiveRenderTarget(nil), result.LiveTargets...),
		Metadata:    result.Metadata,
		Theme:       result.Theme.WithFallback(),
	}
}

// LiveRenderTarget records a live surface actually presented by this frame.
type LiveRenderTarget struct {
	EndpointID string
	TerminalID string
	Revision   uint64
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

// FrameSink 把渲染帧写入 host、recorder 或 test sink。
type FrameSink interface {
	WriteFrame(Frame) error
}

// FrameWriteCompletion 是 host 对单次写帧的本地完成结果。
// Written=false 且 Err=nil 表示 latest-only sink 丢弃了这帧；Err 表示真实写失败。
type FrameWriteCompletion struct {
	Written bool
	Err     error
}

// FrameSinkCompletion 允许真实 host 为单次写帧返回完成信号。
// 该信号只表达 TUI 本地写帧背压边界，不能被上传给 core 当作 rendered revision。
type FrameSinkCompletion interface {
	WriteFrameWithCompletion(Frame) (<-chan FrameWriteCompletion, error)
}

// FrameSinkPreference 允许真实 host 声明自己只需要 ANSI 输出。
// 测试 recorder 不实现该接口时保留完整 Frame，方便继续断言 plain/styled 内容。
type FrameSinkPreference interface {
	NeedsCompleteFrame() bool
}

// FramePatch 是真实 TTY 的增量绘制合同；测试 sink 默认仍消费完整 Frame。
type FramePatch struct {
	Rect Rect
	// CursorOnly 表示本帧只复投宿主光标，不重写 history 内容区。
	CursorOnly bool
	Rewrite    bool
	Dir        FramePatchScrollDirection
	LineY      int
	LineX      int
	LineWidth  int
	LineANSI   string
	LinesANSI  []string
}

type FramePatchScrollDirection int

const (
	FramePatchScrollUp FramePatchScrollDirection = iota + 1
	FramePatchScrollDown
)
