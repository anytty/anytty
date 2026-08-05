package terminalhost

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	xterm "github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
	"github.com/anytty/anytty/tui/input"
	"github.com/anytty/anytty/tui/render"
)

const (
	enterAltScreen      = "\x1b[?1049h"
	exitAltScreen       = "\x1b[?1049l"
	hideCursor          = "\x1b[?25l"
	showCursor          = "\x1b[?25h"
	enableBracketPaste  = "\x1b[?2004h"
	disableBracketPaste = "\x1b[?2004l"
	// Kitty keyboard protocol flag 1 只消除歧义键编码；普通文本仍保持 UTF-8，不改变 PTY 输入语义。
	enableKeyboardDisambiguation = "\x1b[>1u"
	disableKeyboardEnhancements  = "\x1b[<u"
	queryKeyboardEnhancements    = "\x1b[?u"
	enableMouseCell              = "\x1b[?1000h"
	disableMouseCell             = "\x1b[?1000l"
	enableMouseButton            = "\x1b[?1002h"
	disableMouseButton           = "\x1b[?1002l"
	enableMouseSGR               = "\x1b[?1006h"
	disableMouseSGR              = "\x1b[?1006l"
	requestHostFG                = "\x1b]10;?\x07"
	requestHostBG                = "\x1b]11;?\x07"
	escapeSequenceDelay          = 25 * time.Millisecond
)

var (
	ErrNotTerminal = errors.New("terminalhost: input fd is not a terminal")
)

// TerminalState 是 raw mode 前的宿主终端状态句柄。
type TerminalState interface{}

// TerminalOps 隔离真实 TTY 操作，测试通过 fake ops 验证 raw/restore/size。
type TerminalOps interface {
	IsTerminal(fd uintptr) bool
	MakeRaw(fd uintptr) (TerminalState, error)
	Restore(fd uintptr, state TerminalState) error
	GetSize(fd uintptr) (cols int, rows int, err error)
}

type xTermOps struct{}

func (xTermOps) IsTerminal(fd uintptr) bool {
	return xterm.IsTerminal(fd)
}

func (xTermOps) MakeRaw(fd uintptr) (TerminalState, error) {
	return xterm.MakeRaw(fd)
}

func (xTermOps) Restore(fd uintptr, state TerminalState) error {
	typed, ok := state.(*xterm.State)
	if !ok {
		return fmt.Errorf("terminalhost: unexpected terminal state %T", state)
	}
	return xterm.Restore(fd, typed)
}

func (xTermOps) GetSize(fd uintptr) (int, int, error) {
	return xterm.GetSize(fd)
}

// CancelReader 是可被 Host.Close/上下文取消打断的输入 reader。
type CancelReader interface {
	io.Reader
	Cancel() bool
}

// ResizeSignalFactory 建立外部 terminal resize 信号流和对应清理函数。
type ResizeSignalFactory func() (<-chan os.Signal, func())

// Option 配置 Host，主要供 harness 注入 fake TTY。
type Option func(*Host)

// WithInput 设置宿主输入 reader 和可选 fd。
func WithInput(reader io.Reader, fd uintptr) Option {
	return func(host *Host) {
		host.input = reader
		host.fd = fd
	}
}

// WithSizeFD 设置宿主可见窗口尺寸的查询句柄。
// Windows 输入句柄不能提供 ConsoleScreenBufferInfo，真实 Host 因此默认使用 stdout；该选项只供嵌入宿主和 harness 显式覆盖。
func WithSizeFD(fd uintptr) Option {
	return func(host *Host) {
		host.sizeFD = fd
	}
}

// WithOutput 设置宿主输出 writer。
func WithOutput(writer io.Writer) Option {
	return func(host *Host) {
		host.output = writer
	}
}

// WithTerminalOps 注入 TTY 操作实现。
func WithTerminalOps(ops TerminalOps) Option {
	return func(host *Host) {
		if ops != nil {
			host.ops = ops
		}
	}
}

// WithCancelReaderFactory 注入可取消输入 reader，避免测试阻塞在真实 stdin。
func WithCancelReaderFactory(factory func(io.Reader) (CancelReader, error)) Option {
	return func(host *Host) {
		if factory != nil {
			host.cancelReaderFactory = factory
		}
	}
}

// WithInputBuffer 设置输入事件缓冲大小。
func WithInputBuffer(size int) Option {
	return func(host *Host) {
		if size > 0 {
			host.inputBuffer = size
		}
	}
}

// WithResizeSignalFactory 注入 resize 信号源，测试可用它避免依赖真实 SIGWINCH。
func WithResizeSignalFactory(factory ResizeSignalFactory) Option {
	return func(host *Host) {
		if factory != nil {
			host.resizeSignalFactory = factory
		}
	}
}

// WithResizeSignalChannel 注入固定 signal channel，供 deterministic harness 使用。
func WithResizeSignalChannel(signals <-chan os.Signal) Option {
	return WithResizeSignalFactory(func() (<-chan os.Signal, func()) {
		return signals, func() {}
	})
}

// WithThemeProbe 控制宿主 palette/theme probe，测试可关闭以断言 enter 序列。
func WithThemeProbe(enabled bool) Option {
	return func(host *Host) {
		host.themeProbe = enabled
	}
}

// WithLogger 打开 host/frame-sink 诊断日志；实际采样仍由 ANYTTY_TUI_DIAG 控制。
func WithLogger(logger *slog.Logger) Option {
	return func(host *Host) {
		host.logger = logger
	}
}

func (host *Host) SetLogger(logger *slog.Logger) {
	host.mu.Lock()
	defer host.mu.Unlock()
	host.logger = logger
	if host.latestSink != nil {
		host.latestSink.SetLogger(logger)
	}
}

// Host 是 TUI-v3 真实 TerminalHost，实现 app.TerminalHost 所需契约。
type Host struct {
	input  io.Reader
	output io.Writer
	fd     uintptr
	sizeFD uintptr
	ops    TerminalOps

	cancelReaderFactory func(io.Reader) (CancelReader, error)
	cancelReader        CancelReader
	resizeSignalFactory ResizeSignalFactory
	resizeSignalStop    func()

	inputBuffer int
	events      chan input.InputEvent
	ready       chan struct{}
	sink        render.FrameSink
	latestSink  *LatestFrameSink
	themeProbe  bool
	logger      *slog.Logger

	mu             sync.Mutex
	state          TerminalState
	entered        bool
	closed         bool
	keyboardPushed bool
	done           chan struct{}
	wg             sync.WaitGroup
}

// New 建立真实 host。调用 Enter 后才会进入 raw mode 和启动输入循环。
func New(options ...Option) *Host {
	host := &Host{
		input:       os.Stdin,
		output:      os.Stdout,
		fd:          os.Stdin.Fd(),
		sizeFD:      os.Stdout.Fd(),
		ops:         xTermOps{},
		inputBuffer: 64,
		themeProbe:  true,
	}
	host.cancelReaderFactory = func(reader io.Reader) (CancelReader, error) {
		return cancelreader.NewReader(reader)
	}
	for _, option := range options {
		option(host)
	}
	if host.resizeSignalFactory == nil {
		sizeFD := host.sizeFD
		host.resizeSignalFactory = func() (<-chan os.Signal, func()) {
			return defaultResizeSignalFactory(sizeFD)
		}
	}
	host.events = make(chan input.InputEvent, host.inputBuffer)
	host.ready = make(chan struct{}, 1)
	host.sink = NewFrameSink(host.output)
	return host
}

// Enter 进入 raw mode，打开 TUI 需要的终端模式，并启动输入循环。
func (host *Host) Enter(ctx context.Context) error {
	host.mu.Lock()
	if host.entered {
		host.mu.Unlock()
		return nil
	}
	if !host.ops.IsTerminal(host.fd) {
		host.mu.Unlock()
		return ErrNotTerminal
	}
	state, err := host.ops.MakeRaw(host.fd)
	if err != nil {
		host.mu.Unlock()
		return err
	}
	host.state = state
	host.entered = true
	host.closed = false
	host.done = make(chan struct{})
	sequence := enterSequence()
	if host.themeProbe {
		// theme probe 必须在 alt-screen/raw mode 后发送，避免宿主把响应回到普通 shell。
		sequence += hostThemeProbeSequence()
	}
	if err := writeFullString(host.output, sequence); err != nil {
		_ = host.restoreLocked()
		host.mu.Unlock()
		return err
	}
	if err := writeFullString(host.output, enableKeyboardDisambiguation); err != nil {
		_ = host.restoreLocked()
		host.mu.Unlock()
		return err
	}
	host.keyboardPushed = true
	if err := writeFullString(host.output, queryKeyboardEnhancements); err != nil {
		_ = host.restoreLocked()
		host.mu.Unlock()
		return err
	}
	cancelReader, err := host.cancelReaderFactory(host.input)
	if err != nil {
		_ = host.restoreLocked()
		host.mu.Unlock()
		return err
	}
	host.cancelReader = cancelReader
	host.latestSink = NewLatestFrameSink(NewFrameSink(host.output))
	host.latestSink.SetLogger(host.logger)
	host.sink = host.latestSink
	done := host.done
	inputReads := make(chan inputReadResult, 1)
	host.wg.Add(2)
	go host.readInputChunks(ctx, done, cancelReader, inputReads)
	go host.parseInput(ctx, done, inputReads)
	if host.resizeSignalFactory != nil {
		resizeSignals, stop := host.resizeSignalFactory()
		host.resizeSignalStop = stop
		if resizeSignals != nil {
			host.wg.Add(1)
			go host.readResizeSignals(ctx, done, resizeSignals)
		}
	}
	go func() {
		select {
		case <-ctx.Done():
			_ = host.Close()
		case <-done:
		}
	}()
	host.mu.Unlock()
	return nil
}

// Close 恢复终端模式并停止输入循环。该方法可重复调用。
func (host *Host) Close() error {
	host.mu.Lock()
	err := host.restoreLocked()
	host.mu.Unlock()
	host.wg.Wait()
	return err
}

func (host *Host) restoreLocked() error {
	if host.closed {
		return nil
	}
	host.closed = true
	var closeErr error
	if host.cancelReader != nil {
		host.cancelReader.Cancel()
		if closer, ok := host.cancelReader.(io.Closer); ok {
			closeErr = closer.Close()
		}
		host.cancelReader = nil
	}
	if host.resizeSignalStop != nil {
		host.resizeSignalStop()
		host.resizeSignalStop = nil
	}
	if host.latestSink != nil {
		host.latestSink.Close()
		host.latestSink = nil
		host.sink = NewFrameSink(host.output)
	}
	if host.done != nil {
		close(host.done)
		host.done = nil
	}
	var writeErr error
	if host.entered {
		writeErr = writeFullString(host.output, exitSequence(host.keyboardPushed))
	}
	var restoreErr error
	if host.entered && host.state != nil {
		restoreErr = host.ops.Restore(host.fd, host.state)
	}
	host.entered = false
	host.keyboardPushed = false
	host.state = nil
	if writeErr != nil {
		return writeErr
	}
	if restoreErr != nil {
		return restoreErr
	}
	return closeErr
}

func writeFullString(writer io.Writer, value string) error {
	written, err := io.WriteString(writer, value)
	if err != nil {
		return err
	}
	if written != len(value) {
		return io.ErrShortWrite
	}
	return nil
}

// Size 返回当前宿主终端尺寸，单位是 cols/rows。
func (host *Host) Size() (int, int, error) {
	return host.ops.GetSize(host.sizeFD)
}

// InputEvents 返回宿主输入事件流。
func (host *Host) InputEvents() <-chan input.InputEvent {
	return host.events
}

// EventsReady 只负责唤醒 runtime，不直接消费输入事件。
func (host *Host) EventsReady() <-chan struct{} {
	return host.ready
}

// FrameSink 返回真实 TTY frame sink。
func (host *Host) FrameSink() render.FrameSink {
	return host.sink
}

type inputReadResult struct {
	chunk []byte
	err   error
}

func (host *Host) readInputChunks(ctx context.Context, done <-chan struct{}, reader io.Reader, reads chan<- inputReadResult) {
	defer host.wg.Done()
	defer close(reads)
	buffer := make([]byte, 256)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		n, err := reader.Read(buffer)
		result := inputReadResult{err: err}
		if n > 0 {
			result.chunk = append([]byte(nil), buffer[:n]...)
		}
		if n > 0 || err != nil {
			select {
			case reads <- result:
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (host *Host) parseInput(ctx context.Context, done <-chan struct{}, reads <-chan inputReadResult) {
	defer host.wg.Done()
	parser := NewInputParser()
	var escapeTimer *time.Timer
	var escapeTimeout <-chan time.Time
	stopEscapeTimer := func() {
		if escapeTimer == nil {
			return
		}
		if !escapeTimer.Stop() {
			select {
			case <-escapeTimer.C:
			default:
			}
		}
		escapeTimeout = nil
	}
	defer stopEscapeTimer()
	emit := func(events []input.InputEvent) bool {
		for _, event := range events {
			select {
			case host.events <- event:
				host.signalReady()
			case <-ctx.Done():
				return false
			case <-done:
				return false
			}
		}
		return true
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-escapeTimeout:
			escapeTimeout = nil
			if !emit(parser.FlushEscape()) {
				return
			}
		case result, ok := <-reads:
			if !ok {
				emit(parser.FlushEscape())
				return
			}
			stopEscapeTimer()
			if !emit(parser.Feed(result.chunk)) {
				return
			}
			if parser.hasPendingEscape() {
				if escapeTimer == nil {
					escapeTimer = time.NewTimer(escapeSequenceDelay)
				} else {
					escapeTimer.Reset(escapeSequenceDelay)
				}
				escapeTimeout = escapeTimer.C
			}
			if result.err != nil {
				emit(parser.FlushEscape())
				return
			}
		}
	}
}

func (host *Host) readResizeSignals(ctx context.Context, done <-chan struct{}, signals <-chan os.Signal) {
	defer host.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case _, ok := <-signals:
			if !ok {
				return
			}
			cols, rows, err := host.Size()
			if err != nil || cols <= 0 || rows <= 0 {
				continue
			}
			event := input.InputEvent{Kind: input.EventKindResize, Cols: cols, Rows: rows}
			select {
			case host.events <- event:
				host.signalReady()
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}
}

func enterSequence() string {
	return enterAltScreen + hideCursor + enableBracketPaste + enableMouseCell + enableMouseButton + enableMouseSGR
}

func hostThemeProbeSequence() string {
	var builder strings.Builder
	builder.WriteString(requestHostFG)
	builder.WriteString(requestHostBG)
	for index := 0; index < 16; index++ {
		builder.WriteString(fmt.Sprintf("\x1b]4;%d;?\x07", index))
	}
	return builder.String()
}

func exitSequence(keyboardPushed bool) string {
	// 恢复顺序与进入顺序相反，避免退出时遗留鼠标或隐藏光标状态。
	sequence := disableMouseSGR + disableMouseButton + disableMouseCell + disableBracketPaste
	if keyboardPushed {
		sequence += disableKeyboardEnhancements
	}
	return sequence + showCursor + exitAltScreen
}

func (host *Host) signalReady() {
	select {
	case host.ready <- struct{}{}:
	default:
	}
}
