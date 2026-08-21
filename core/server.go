package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/core/history/linehist"
	"github.com/anytty/anytty/core/live"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/perftrace"
	"github.com/anytty/anytty/shared/runtimepath"
	"github.com/anytty/anytty/shared/transport"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
)

type ListenerFactory func(socketPath string) (transport.Listener, error)

type ServerOption func(*serverConfig)

type serverConfig struct {
	socketPath          string
	defaultSize         Size
	logger              *slog.Logger
	listenerFactory     ListenerFactory
	processFactory      ProcessFactory
	remoteService       RemoteService
	clientAccessService ClientAccessService
	eventBuffer         int
	historyStoreFactory HistoryStoreFactory
	historyStorageDir   string
	historyStorage      HistoryStorageConfig
	historyDisabled     bool
	terminalOutput      TerminalOutputBufferConfig
	outputResidentBytes int64
	applicationFactory  ApplicationExecutorFactory
	protocolLimits      ProtocolSessionLimits
	grantNow            func() time.Time
	grantAfterFunc      func(time.Duration, func()) grantTimer

	protocolRequestBudget int
}

const defaultProtocolRequestBudget = 512

// ProtocolSessionLimits bounds connection-owned goroutines and long-lived resources.
// Each resource kind has a concrete cap in addition to the aggregate resource cap.
type ProtocolSessionLimits struct {
	MaxInFlightRequests   int
	MaxResources          int
	MaxAttachments        int
	MaxFileTransfers      int
	MaxEventSubscriptions int
}

// DefaultProtocolSessionLimits returns the production per-connection bounds.
func DefaultProtocolSessionLimits() ProtocolSessionLimits {
	return ProtocolSessionLimits{
		MaxInFlightRequests:   64,
		MaxResources:          192,
		MaxAttachments:        128,
		MaxFileTransfers:      32,
		MaxEventSubscriptions: 64,
	}
}

func (limits ProtocolSessionLimits) normalized() ProtocolSessionLimits {
	defaults := DefaultProtocolSessionLimits()
	if limits.MaxInFlightRequests <= 0 {
		limits.MaxInFlightRequests = defaults.MaxInFlightRequests
	}
	if limits.MaxResources <= 0 {
		limits.MaxResources = defaults.MaxResources
	}
	if limits.MaxAttachments <= 0 {
		limits.MaxAttachments = defaults.MaxAttachments
	}
	if limits.MaxFileTransfers <= 0 {
		limits.MaxFileTransfers = defaults.MaxFileTransfers
	}
	if limits.MaxEventSubscriptions <= 0 {
		limits.MaxEventSubscriptions = defaults.MaxEventSubscriptions
	}
	return limits
}

const DefaultHistoryMaxBytesPerTerminal int64 = 512 << 20

const (
	HistoryCompressionZstd = "zstd"
	HistoryCompressionS2   = "s2"
	HistoryCompressionNone = "none"

	HistoryCompressionLevelFast     = "fast"
	HistoryCompressionLevelBalanced = "balanced"
	HistoryCompressionLevelBest     = "best"
)

// HistoryStorageConfig 控制每个 terminal 的历史物理保留上限、期限与块编码。
// MaxBytesPerTerminal=0、MaxAge=0 分别表示不设对应限制。
type HistoryStorageConfig struct {
	MaxBytesPerTerminal int64
	MaxAge              time.Duration
	Compression         string
	CompressionLevel    string
}

func DefaultHistoryStorageConfig() HistoryStorageConfig {
	return HistoryStorageConfig{
		MaxBytesPerTerminal: DefaultHistoryMaxBytesPerTerminal,
		Compression:         HistoryCompressionZstd,
		CompressionLevel:    HistoryCompressionLevelFast,
	}
}

// PrepareHistoryStorage 在 daemon 单实例锁内清理旧格式，并把所有当前块文件
// 收敛到新的按 terminal 上限。旧格式明确不迁移。
func PrepareHistoryStorage(dir string, storage HistoryStorageConfig) error {
	storage = storage.normalized()
	return linehist.PrepareDirectory(dir, linehist.CompressedLineFileOptions{
		MaxBytes:         storage.MaxBytesPerTerminal,
		MaxAge:           storage.MaxAge,
		Compression:      storage.Compression,
		CompressionLevel: storage.CompressionLevel,
	})
}

// DeleteTerminalHistory 删除指定 terminal 的全部本地 history 文件。
func DeleteTerminalHistory(dir string, terminalID string) (int, error) {
	return linehist.DeleteTerminalHistory(dir, terminalID)
}

// DeleteAllHistory 删除本地 history 目录内的全部 terminal history 文件。
func DeleteAllHistory(dir string) (int, error) {
	return linehist.DeleteAllHistory(dir)
}

// DeleteObsoleteCompactHistory 删除不再读取的早期 core-v2 history 文件。
func DeleteObsoleteCompactHistory(dir string) (int, error) {
	return linehist.DeleteObsoleteCompactHistory(dir)
}

func (config HistoryStorageConfig) normalized() HistoryStorageConfig {
	if config.MaxBytesPerTerminal < 0 {
		config.MaxBytesPerTerminal = DefaultHistoryMaxBytesPerTerminal
	}
	if config.MaxAge < 0 {
		config.MaxAge = 0
	}
	config.Compression = strings.ToLower(strings.TrimSpace(config.Compression))
	if config.Compression != HistoryCompressionNone && config.Compression != HistoryCompressionS2 {
		config.Compression = HistoryCompressionZstd
	}
	config.CompressionLevel = strings.ToLower(strings.TrimSpace(config.CompressionLevel))
	if config.CompressionLevel != HistoryCompressionLevelBalanced && config.CompressionLevel != HistoryCompressionLevelBest {
		config.CompressionLevel = HistoryCompressionLevelFast
	}
	return config
}

// HistoryStoreFactory 为每个 terminal 创建 core-v2 authoritative history store。
// domain owner 是 Server/Terminal lifecycle：工厂只决定 payload residency backend，
// 不能解释 terminal semantic transaction，也不能替代 HistoryStore 的 cursor truth。
type HistoryStoreFactory func(terminalID string) (history.HistoryStore, error)

type Server struct {
	cfg                    serverConfig
	registry               *terminalRegistry
	storage                *storageStore
	terminals              map[string]*Terminal
	events                 *eventBroker
	closed                 atomic.Bool
	shutdownOnce           sync.Once
	shutdownDone           chan struct{}
	shutdownErr            error
	nextProtocolSessionID  atomic.Uint64
	protocolRequestSlots   chan struct{}
	liveBaselineBytes      atomic.Int64
	lifecycleMu            sync.Mutex
	protocolAttachmentMu   sync.Mutex
	protocolAttachments    map[string]protocolAttachment
	protocolChannelIndex   map[protocolAttachmentKey]string
	protocolResizeOwners   map[string]string
	protocolResizeEpochs   map[string]uint64
	protocolSizeLocks      map[string]bool
	protocolOwnerEpoch     uint64
	remoteServiceMu        sync.RWMutex
	remoteService          RemoteService
	clientAccessServiceMu  sync.RWMutex
	clientAccessService    ClientAccessService
	fileTransferMu         sync.Mutex
	fileUploads            map[string]*uploadTransferRecord
	historyFallbackDirMu   sync.Mutex
	historyFallbackDir     string
	historyRetentionMu     sync.Mutex
	historyRetentionCancel context.CancelFunc
	historyRetentionWG     sync.WaitGroup
	outputBudget           *terminalOutputResidentBudget
	grantOperationsMu      sync.Mutex
	grantOperations        map[string]*grantOperation
	grantNow               func() time.Time
	grantAfterFunc         func(time.Duration, func()) grantTimer
	mu                     sync.Mutex
	listeners              []transport.Listener
	transports             map[*trackedTransport]struct{}
	wgMu                   sync.Mutex
	wg                     sync.WaitGroup
}

func NewServer(opts ...ServerOption) *Server {
	cfg := serverConfig{
		socketPath:          defaultSocketPath(),
		defaultSize:         Size{Cols: 80, Rows: 24},
		logger:              slog.Default(),
		listenerFactory:     unixListenerFactory,
		processFactory:      newPTYProcessFactory(),
		eventBuffer:         64,
		protocolLimits:      DefaultProtocolSessionLimits(),
		historyStorage:      DefaultHistoryStorageConfig(),
		terminalOutput:      DefaultTerminalOutputBufferConfig(),
		outputResidentBytes: DefaultTerminalOutputResidentBudgetBytes,
		grantNow:            time.Now,
		grantAfterFunc: func(delay time.Duration, callback func()) grantTimer {
			return time.AfterFunc(delay, callback)
		},

		protocolRequestBudget: defaultProtocolRequestBudget,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}
	if cfg.listenerFactory == nil {
		cfg.listenerFactory = unixListenerFactory
	}
	if cfg.processFactory == nil {
		cfg.processFactory = newPTYProcessFactory()
	}
	cfg.terminalOutput = cfg.terminalOutput.normalized()
	if cfg.outputResidentBytes < MinTerminalOutputResidentBudgetBytes || cfg.outputResidentBytes > MaxTerminalOutputResidentBudgetBytes {
		cfg.outputResidentBytes = DefaultTerminalOutputResidentBudgetBytes
	}
	cfg.historyStorage = cfg.historyStorage.normalized()
	cfg.protocolLimits = cfg.protocolLimits.normalized()
	if cfg.protocolRequestBudget <= 0 {
		cfg.protocolRequestBudget = defaultProtocolRequestBudget
	}
	return &Server{
		cfg:                  cfg,
		registry:             newTerminalRegistry(),
		storage:              newStorageStore(),
		terminals:            make(map[string]*Terminal),
		events:               newEventBroker(cfg.eventBuffer),
		protocolAttachments:  make(map[string]protocolAttachment),
		protocolChannelIndex: make(map[protocolAttachmentKey]string),
		protocolResizeOwners: make(map[string]string),
		protocolResizeEpochs: make(map[string]uint64),
		protocolSizeLocks:    make(map[string]bool),
		remoteService:        cfg.remoteService,
		clientAccessService:  cfg.clientAccessService,
		fileUploads:          make(map[string]*uploadTransferRecord),
		grantOperations:      make(map[string]*grantOperation),
		grantNow:             cfg.grantNow,
		grantAfterFunc:       cfg.grantAfterFunc,
		transports:           make(map[*trackedTransport]struct{}),
		outputBudget:         newTerminalOutputResidentBudget(cfg.outputResidentBytes),
		protocolRequestSlots: make(chan struct{}, cfg.protocolRequestBudget),
		shutdownDone:         make(chan struct{}),
	}
}

func WithSocketPath(path string) ServerOption {
	return func(cfg *serverConfig) {
		if path != "" {
			cfg.socketPath = path
		}
	}
}

// WithApplicationExecutorFactory 注入 connection-bound API Layer 装配。
// core 只保存 factory contract；具体 Proto validation、mapping 与 controller 组合必须位于 core 外部。
func WithApplicationExecutorFactory(factory ApplicationExecutorFactory) ServerOption {
	return func(cfg *serverConfig) {
		cfg.applicationFactory = factory
	}
}

// WithProtocolSessionLimits configures concrete per-connection protocol bounds.
func WithProtocolSessionLimits(limits ProtocolSessionLimits) ServerOption {
	return func(cfg *serverConfig) {
		cfg.protocolLimits = limits.normalized()
	}
}

func WithDefaultSize(cols, rows uint16) ServerOption {
	return func(cfg *serverConfig) {
		cfg.defaultSize = Size{Cols: cols, Rows: rows}
	}
}

func WithLogger(logger *slog.Logger) ServerOption {
	return func(cfg *serverConfig) {
		cfg.logger = logger
	}
}

func WithListenerFactory(factory ListenerFactory) ServerOption {
	return func(cfg *serverConfig) {
		cfg.listenerFactory = factory
	}
}

func WithProcessFactory(factory ProcessFactory) ServerOption {
	return func(cfg *serverConfig) {
		cfg.processFactory = factory
	}
}

func WithRemoteService(service RemoteService) ServerOption {
	return func(cfg *serverConfig) {
		cfg.remoteService = service
	}
}

// WithClientAccessService 注入 daemon-owned DeviceIdentity、PairingTicket 与 client grant 管理边界。
// 该 service 只能由认证后的 scoped protocol session 调用；core 不复制 AccessStore state，也不为缺失实现提供 bearer fallback。
func WithClientAccessService(service ClientAccessService) ServerOption {
	return func(cfg *serverConfig) {
		cfg.clientAccessService = service
	}
}

func WithEventBuffer(size int) ServerOption {
	return func(cfg *serverConfig) {
		cfg.eventBuffer = size
	}
}

// WithHistoryStoreFactory 注入 terminal history store 工厂。调用边界是
// RegisterTerminal；工厂返回错误时 terminal 创建必须失败，不能静默回退到内存
// store 后继续占用大量 RSS。
func WithHistoryStoreFactory(factory HistoryStoreFactory) ServerOption {
	return func(cfg *serverConfig) {
		cfg.historyStoreFactory = factory
	}
}

// WithHistoryStorageDir 记录生产 daemon 的 history storage 目录。
// R436 后默认 history 引擎是 linehist（logical-line 文件存储）：配置目录时
// 每 terminal 的 logical-line 文件落在该目录；未配置时 Server 在系统临时
// 目录下创建私有目录兜底，保证默认路径仍是文件为真值。
func WithHistoryStorageDir(dir string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.historyStorageDir = dir
		cfg.historyStoreFactory = nil
		cfg.historyDisabled = false
	}
}

// WithHistoryStorageConfig 配置生产块存储。它不改变 history semantic truth，
// 只决定物理编码和 oldest-first retention。
func WithHistoryStorageConfig(storage HistoryStorageConfig) ServerOption {
	return func(cfg *serverConfig) {
		cfg.historyStorage = storage.normalized()
	}
}

// WithHistoryDisabled 关闭 core-v2 authoritative history owner。
// domain owner 是 Server/Terminal lifecycle：关闭后 terminal 只维护 native live
// screen 与 live revision，PTY 写入热路径不会创建 renderer/store mutation，
// history.window/copy/freeze/release 必须返回 ErrHistoryDisabled。
func WithHistoryDisabled() ServerOption {
	return func(cfg *serverConfig) {
		cfg.historyDisabled = true
		cfg.historyStoreFactory = nil
		cfg.historyStorageDir = ""
	}
}

// WithTerminalOutputBufferConfig configures the shared per-generation PTY buffer.
func WithTerminalOutputBufferConfig(output TerminalOutputBufferConfig) ServerOption {
	return func(cfg *serverConfig) {
		cfg.terminalOutput = output.normalized()
	}
}

// WithTerminalOutputResidentBudget sets the daemon-wide actual resident-byte cap.
func WithTerminalOutputResidentBudget(bytes int64) ServerOption {
	return func(cfg *serverConfig) {
		if bytes >= MinTerminalOutputResidentBudgetBytes && bytes <= MaxTerminalOutputResidentBudgetBytes {
			cfg.outputResidentBytes = bytes
		}
	}
}

// SetRemoteService 允许 daemon 完成 core-v2 server 构造后再注入 remote runtime hook。
func (server *Server) SetRemoteService(service RemoteService) {
	server.remoteServiceMu.Lock()
	defer server.remoteServiceMu.Unlock()
	server.remoteService = service
}

func (server *Server) RemoteService() RemoteService {
	server.remoteServiceMu.RLock()
	defer server.remoteServiceMu.RUnlock()
	return server.remoteService
}

// ClientAccessService 返回当前 daemon 装配的 client access service。
// 返回 nil 表示 identity/access runtime 不可用，protocol session 必须返回明确 unavailable，不能自行读写 daemon 文件。
func (server *Server) ClientAccessService() ClientAccessService {
	server.clientAccessServiceMu.RLock()
	defer server.clientAccessServiceMu.RUnlock()
	return server.clientAccessService
}

func (server *Server) SocketPath() string {
	return server.cfg.socketPath
}

func (server *Server) DefaultSize() Size {
	return server.cfg.defaultSize
}

// HistoryStorageDir 返回 daemon 配置的 history payload 目录。
// R436 后它是默认 linehist 引擎每 terminal logical-line 文件的落盘位置；
// 未配置时默认引擎会退到 server 级临时目录（见 historyStoreDir）。
func (server *Server) HistoryStorageDir() string {
	return server.cfg.historyStorageDir
}

// HistoryStorageConfig 返回 daemon 当前按 terminal 的物理存储策略。
func (server *Server) HistoryStorageConfig() HistoryStorageConfig {
	return server.cfg.historyStorage
}

func (server *Server) TerminalOutputBufferConfig() TerminalOutputBufferConfig {
	return server.cfg.terminalOutput.normalized()
}

func (server *Server) TerminalOutputResidentBudget() int64 {
	return server.cfg.outputResidentBytes
}

func (server *Server) RegisterTerminal(record TerminalRecord) (TerminalInfo, error) {
	finishTotal := perftrace.Measure("core.server.register_terminal.total")
	defer finishTotal(0)
	if server.closed.Load() {
		return TerminalInfo{}, ErrServerClosed
	}
	server.lifecycleMu.Lock()
	defer server.lifecycleMu.Unlock()
	if server.closed.Load() {
		return TerminalInfo{}, ErrServerClosed
	}
	finishRegistry := perftrace.Measure("core.server.register_terminal.registry")
	info, err := server.registry.register(record, server.cfg.defaultSize)
	finishRegistry(0)
	if err != nil {
		return TerminalInfo{}, err
	}
	var historyStore history.HistoryStore
	historyEnabled := !server.cfg.historyDisabled
	if historyEnabled {
		var err error
		finishHistory := perftrace.Measure("core.server.register_terminal.history_store")
		historyStore, err = server.newHistoryStore(info.ID)
		finishHistory(0)
		if err != nil {
			_, _ = server.registry.remove(info.ID)
			return TerminalInfo{}, err
		}
	}
	finishSpawn := perftrace.Measure("core.server.register_terminal.spawn")
	process, err := server.cfg.processFactory.Spawn(context.Background(), processSpecFromTerminal(info, record.Options))
	finishSpawn(0)
	if err != nil {
		_, _ = server.registry.remove(info.ID)
		return TerminalInfo{}, err
	}
	finishNewTerminal := perftrace.Measure("core.server.register_terminal.new_terminal")
	terminal := newTerminal(info, record.Options, process, server.events, server.updateTerminalInfo, historyStore, historyEnabled, server.cfg.terminalOutput, server.outputBudget, server.cfg.logger)
	server.mu.Lock()
	server.terminals[info.ID] = terminal
	server.mu.Unlock()
	finishNewTerminal(0)
	finishPublish := perftrace.Measure("core.server.register_terminal.publish")
	server.publishTerminalEvent(EventTerminalCreated, info)
	finishPublish(0)
	return info, nil
}

func (server *Server) newHistoryStore(terminalID string) (history.HistoryStore, error) {
	if server.cfg.historyStoreFactory != nil {
		return server.cfg.historyStoreFactory(terminalID)
	}
	// 中文说明：R436 后默认 history 引擎是 linehist——vterm emulator 是唯一
	// 屏幕真值，滚出行 seal-on-eviction 落 logical-line 文件，查询时投影。
	dir, err := server.historyStoreDir()
	if err != nil {
		return nil, err
	}
	return LineHistoryStoreFactoryWithConfig(dir, server.cfg.historyStorage)(terminalID)
}

// historyStoreDir 返回默认 linehist 引擎的落盘目录。未配置
// WithHistoryStorageDir 时（测试/临时 daemon）在系统临时目录下创建
// server 级私有目录兜底：默认路径的 truth 仍是文件，不提供内存实现。
func (server *Server) historyStoreDir() (string, error) {
	if dir := strings.TrimSpace(server.cfg.historyStorageDir); dir != "" {
		return dir, nil
	}
	server.historyFallbackDirMu.Lock()
	defer server.historyFallbackDirMu.Unlock()
	if server.historyFallbackDir == "" {
		dir, err := os.MkdirTemp("", "anytty-linehist-")
		if err != nil {
			return "", err
		}
		server.historyFallbackDir = dir
	}
	return server.historyFallbackDir, nil
}

// LineHistoryStoreFactory 返回 R433 linehist（logical-line 文件存储）的
// history store factory。Terminal 识别到
// *linehist.Store 后走单一真值链路（tap 事务 EvictedRows 落盘 + 查询时
// 投影 emulator 当前屏），旧 journal/classifier fanout 被旁路。
func LineHistoryStoreFactory(dir string) HistoryStoreFactory {
	return LineHistoryStoreFactoryWithConfig(dir, DefaultHistoryStorageConfig())
}

// LineHistoryStoreFactoryWithConfig 创建带压缩与按 terminal retention 的生产
// store factory。旧格式不迁移，首次打开时直接开始新的压缩块文件。
func LineHistoryStoreFactoryWithConfig(dir string, storage HistoryStorageConfig) HistoryStoreFactory {
	storage = storage.normalized()
	return func(terminalID string) (history.HistoryStore, error) {
		finish := perftrace.Measure("core.linehist.open_store")
		file, err := linehist.OpenCompressedLineFile(dir, terminalID, linehist.CompressedLineFileOptions{
			MaxBytes:         storage.MaxBytesPerTerminal,
			MaxAge:           storage.MaxAge,
			Compression:      storage.Compression,
			CompressionLevel: storage.CompressionLevel,
		})
		finish(0)
		if err != nil {
			return nil, err
		}
		return linehist.NewStore(terminalID, linehist.NewEngine(file)), nil
	}
}

func processSpecFromTerminal(info TerminalInfo, options TerminalCreateOptions) ProcessSpec {
	return ProcessSpec{
		TerminalID:         info.ID,
		Command:            append([]string(nil), info.Command...),
		Size:               info.Size,
		Dir:                options.Dir,
		Env:                append([]string(nil), options.Env...),
		ScrollbackSize:     options.ScrollbackSize,
		ScrollbackMaxBytes: options.ScrollbackMaxBytes,
		ScrollbackMaxAge:   options.ScrollbackMaxAge,
	}
}

func (server *Server) GetTerminal(id string) (TerminalInfo, error) {
	return server.registry.get(id)
}

func (server *Server) ListTerminals() []TerminalInfo {
	items := server.registry.list()
	if len(items) == 0 {
		return items
	}
	server.mu.Lock()
	terminals := make(map[string]*Terminal, len(server.terminals))
	for id, terminal := range server.terminals {
		terminals[id] = terminal
	}
	server.mu.Unlock()
	processIDs := make([]int, 0, len(items))
	for index := range items {
		terminal := terminals[items[index].ID]
		if terminal == nil {
			continue
		}
		// 中文说明：resources 是 Terminal 对象持续采样的诊断投影，不写回 registry；
		// registry 仍只保存 lifecycle/metadata truth。
		items[index].Resources, items[index].ResourceHistory = terminal.ResourceSnapshot()
		items[index].LastOutputAt = terminal.LastOutputAt()
		if items[index].State == TerminalStateRunning && items[index].Resources.PID > 0 {
			processIDs = append(processIDs, items[index].Resources.PID)
		}
	}
	foregroundProcesses := foregroundProcessSnapshot(processIDs)
	for index := range items {
		items[index].ForegroundProcess = foregroundProcesses[items[index].Resources.PID]
	}
	return items
}

func (server *Server) StorageGet(ctx context.Context, appID string, scope StorageScope, ownerID string, key string) (StorageEntry, error) {
	_ = ctx
	return server.storage.get(appID, scope, ownerID, key)
}

func (server *Server) StoragePut(ctx context.Context, request StoragePutRequest) (StorageEntry, error) {
	_ = ctx
	entry, err := server.storage.put(request)
	if err != nil {
		return StorageEntry{}, err
	}
	server.publishStorageEvent(StorageChanged{
		AppID:   entry.AppID,
		Scope:   entry.Scope,
		OwnerID: entry.OwnerID,
		Key:     entry.Key,
		Version: entry.Version,
		Op:      StorageOpPut,
	})
	return entry, nil
}

func (server *Server) StorageDelete(ctx context.Context, request StorageDeleteRequest) (StorageDeleteResult, error) {
	_ = ctx
	result, err := server.storage.delete(request)
	if err != nil {
		return StorageDeleteResult{}, err
	}
	server.publishStorageEvent(StorageChanged{
		AppID:   result.AppID,
		Scope:   result.Scope,
		OwnerID: result.OwnerID,
		Key:     result.Key,
		Version: result.Version,
		Op:      StorageOpDelete,
	})
	return result, nil
}

func (server *Server) StorageList(ctx context.Context, appID string, scope StorageScope, ownerID string, prefix string) []StorageEntry {
	_ = ctx
	return server.storage.list(appID, scope, ownerID, prefix)
}

func (server *Server) SetMetadata(ctx context.Context, id string, name string, tags map[string]string) (TerminalInfo, error) {
	_ = ctx
	if server.closed.Load() {
		return TerminalInfo{}, ErrServerClosed
	}
	server.lifecycleMu.Lock()
	defer server.lifecycleMu.Unlock()
	if server.closed.Load() {
		return TerminalInfo{}, ErrServerClosed
	}
	// 中文说明：metadata rename 会改变 name-as-key 的查找身份；
	// 必须先经过 registry 的 daemon-local 唯一性检查，再同步到 running Terminal。
	info, err := server.registry.setMetadata(id, name, tags)
	if err != nil {
		return TerminalInfo{}, err
	}
	server.mu.Lock()
	terminal := server.terminals[id]
	server.mu.Unlock()
	if terminal != nil {
		return terminal.SetMetadata(info.Name, info.Tags), nil
	}
	server.publishTerminalEvent(EventTerminalMetadataChanged, info)
	return info, nil
}

func (server *Server) RemoveTerminal(id string) error {
	if server.closed.Load() {
		return ErrServerClosed
	}
	server.lifecycleMu.Lock()
	defer server.lifecycleMu.Unlock()
	if server.closed.Load() {
		return ErrServerClosed
	}
	info, err := server.registry.remove(id)
	if err != nil {
		return err
	}
	terminal := server.removeTerminalHandle(id)
	var closeErr error
	if terminal != nil {
		closeErr = terminal.Close()
	}
	server.publishTerminalEvent(EventTerminalRemoved, info)
	return closeErr
}

func (server *Server) updateTerminalInfo(info TerminalInfo) {
	_ = server.registry.replace(info)
}

func (server *Server) Terminal(id string) (*Terminal, error) {
	server.mu.Lock()
	defer server.mu.Unlock()
	terminal, ok := server.terminals[id]
	if !ok {
		return nil, ErrTerminalNotFound
	}
	return terminal, nil
}

func (server *Server) WriteInput(ctx context.Context, id string, data []byte) error {
	_ = ctx
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	return terminal.Input(data)
}

func (server *Server) IngestOutput(ctx context.Context, id string, output string) error {
	_ = ctx
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	return terminal.IngestOutput(output)
}

func (server *Server) ResizeTerminal(ctx context.Context, id string, cols, rows uint16) error {
	_ = ctx
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	return terminal.Resize(Size{Cols: cols, Rows: rows})
}

func (server *Server) KillTerminal(ctx context.Context, id string) error {
	_ = ctx
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	return terminal.Kill()
}

func (server *Server) RestartTerminal(ctx context.Context, id string) error {
	if server.closed.Load() {
		return ErrServerClosed
	}
	server.lifecycleMu.Lock()
	defer server.lifecycleMu.Unlock()
	if server.closed.Load() {
		return ErrServerClosed
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	before := terminal.Info()
	coreLifecycleTrace(server.cfg.logger, "server.restart.request", coreTerminalInfoAttrs(before)...)
	if err := terminal.Restart(ctx, server.cfg.processFactory); err != nil {
		coreLifecycleTrace(server.cfg.logger, "server.restart.result",
			"terminal_id", id,
			"error", err.Error(),
			"state_before", string(before.State),
			"exit_code_before", coreTraceExitCode(before.ExitCode),
			"exited_at_before", coreTraceTime(before.ExitedAt),
		)
		return err
	}
	after := terminal.Info()
	attrs := coreTerminalInfoAttrs(after)
	attrs = append(attrs,
		"state_before", string(before.State),
		"exited_at_before", coreTraceTime(before.ExitedAt),
	)
	coreLifecycleTrace(server.cfg.logger, "server.restart.result", attrs...)
	return nil
}

func (server *Server) LiveRows(id string) ([]string, error) {
	terminal, err := server.Terminal(id)
	if err != nil {
		return nil, err
	}
	return terminal.LiveRows(), nil
}

func (server *Server) LiveSnapshot(id string) (live.SurfaceSnapshot, error) {
	terminal, err := server.Terminal(id)
	if err != nil {
		return live.SurfaceSnapshot{}, err
	}
	return terminal.LiveSnapshot(), nil
}

// TerminalHistoryBacklogStatus 返回 terminal history consumer 的 output buffer 诊断。
// 它是只读诊断入口，不触发 FlushHistory，也不读取 live snapshot 或组装 window；
// copy/history 入口必须继续通过统一的 history.window 获取 authoritative rows。
func (server *Server) TerminalHistoryBacklogStatus(id string) (HistoryBacklogStatus, error) {
	terminal, err := server.Terminal(id)
	if err != nil {
		return HistoryBacklogStatus{}, err
	}
	return terminal.HistoryBacklogStatus(), nil
}

// nextLiveScreenWithBaseline returns one canonical latest-screen response.
// Revision zero bootstraps; an observed current revision waits for the next
// invalidation before comparing against this protocol session's confirmed frame.
func (server *Server) nextLiveScreenWithBaseline(ctx context.Context, id string, observedRevision LiveRevision, base *nativeScreenBaseline) (NativeScreenSnapshot, *nativeScreenBaseline, error) {
	terminal, err := server.Terminal(id)
	if err != nil {
		return NativeScreenSnapshot{}, nil, err
	}
	current := terminal.LiveRevision()
	switch {
	case observedRevision == 0:
		if err := terminal.flushLiveOutput(ctx); err != nil {
			return NativeScreenSnapshot{}, nil, err
		}
	case observedRevision <= current:
		if _, err := server.NextLiveInvalidation(ctx, id, observedRevision); err != nil {
			return NativeScreenSnapshot{}, nil, err
		}
	default:
		if err := terminal.flushLiveOutput(ctx); err != nil {
			return NativeScreenSnapshot{}, nil, err
		}
	}
	registered, err := server.Terminal(id)
	if err != nil {
		return NativeScreenSnapshot{}, nil, err
	}
	if registered != terminal {
		return NativeScreenSnapshot{}, nil, ErrTerminalNotFound
	}
	snapshot, currentBase := terminal.nativeScreenSnapshotSinceBaseline(id, observedRevision, base)
	return snapshot, currentBase, nil
}

// NextLiveInvalidation 等待指定 terminal 的下一次 live invalidation。
// observedRevision 是客户端已从 core 看到的 native screen revision，不是 TUI 已渲染
// revision；core 只用它补 one-shot arm 间隙丢失的 wake，仍不维护客户端渲染进度。
func (server *Server) NextLiveInvalidation(ctx context.Context, id string, observedRevision LiveRevision) (Event, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return Event{}, err
	}
	// Subscribe before checking the revision so output cannot land between the
	// check and the one-shot subscription.
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := server.Events(waitCtx, EventFilter{
		TerminalID: id,
		Types:      []EventType{EventTerminalLiveInvalidated, EventTerminalRemoved},
	})
	if revision := terminal.LiveRevision(); revision > observedRevision {
		// 中文说明：这是 latest native screen 的边沿补偿，不是事件回放队列。
		// 返回 wake 前只等待调用时已经进入 output buffer 的 PTY payload，
		// 把同一批 burst 合并到当前 latest revision；不等待 future output、
		// history backlog、TUI render，也不携带 screen payload。
		if err := terminal.flushLiveOutput(ctx); err != nil {
			return Event{}, err
		}
		registered, err := server.Terminal(id)
		if err != nil {
			return Event{}, err
		}
		if registered != terminal {
			return Event{}, ErrTerminalNotFound
		}
		revision = terminal.LiveRevision()
		return Event{
			Type:       EventTerminalLiveInvalidated,
			TerminalID: id,
			Live: &LiveScreenInvalidated{
				TerminalID: id,
				Revision:   revision,
			},
		}, nil
	}
	for {
		select {
		case <-ctx.Done():
			return Event{}, ctx.Err()
		case event, ok := <-events:
			if !ok {
				return Event{}, context.Canceled
			}
			if event.Type == EventTerminalRemoved {
				return Event{}, ErrTerminalNotFound
			}
			if event.Live == nil {
				continue
			}
			if event.Live.Revision <= observedRevision {
				continue
			}
			// 中文说明：one-shot wake 是唤醒边界，不是 frame delivery。
			// 这里 coalesce 当前已入队 PTY，避免 TUI 每个中间 revision 都拉一次
			// live.screen.get；screen rows 仍由客户端随后 pull latest。
			if err := terminal.flushLiveOutput(ctx); err != nil {
				return Event{}, err
			}
			registered, err := server.Terminal(id)
			if err != nil {
				return Event{}, err
			}
			if registered != terminal {
				return Event{}, ErrTerminalNotFound
			}
			event.Live.Revision = terminal.LiveRevision()
			return event, nil
		}
	}
}

// TerminalHistoryWindow 返回 core-v2 terminal 内部 authoritative history domain
// projection。调用边界包括 protocol history.window 和 domain harness；实现只能读取
// core history store，不能 fallback 到 live rows 或 snapshot。
func (server *Server) TerminalHistoryWindow(ctx context.Context, id string, req history.HistoryWindowRequest) (history.HistoryWindow, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Token == "" && req.Mode != "" && req.Mode != history.HistoryWindowModeLatest {
		return history.HistoryWindow{}, history.ErrHistoryInvalidMutation
	}
	mode := historyWindowPerfMode(req.Mode)
	terminal, err := server.Terminal(id)
	if err != nil {
		return history.HistoryWindow{}, err
	}
	if req.Token == "" {
		// Live windows only need the consumer fence for a consistent view. Freeze
		// owns the durability fence; token pagination must not repeat file.Sync.
		finishFlush := perftrace.Measure("core.server.history_window." + mode + ".flush")
		if err := terminal.waitForHistory(ctx); err != nil {
			finishFlush(0)
			return history.HistoryWindow{}, err
		}
		finishFlush(0)
	}
	if req.TerminalID == "" {
		req.TerminalID = id
	}
	finishStore := perftrace.Measure("core.server.history_window." + mode + ".terminal")
	window, err := terminal.HistoryWindow(req)
	finishStore(len(window.Rows))
	perftrace.Count("core.server.history_window."+mode+".rows", len(window.Rows))
	if err != nil {
		return history.HistoryWindow{}, err
	}
	return window, nil
}

// TerminalHistoryCopy 从 core-v2 terminal 内部 authoritative history domain 复制文本。
// 调用边界包括 protocol history.copy；复制文本必须来自 tokenized history payload，
// 不能由 TUI rows 或 live surface cache 组装。
func (server *Server) TerminalHistoryCopy(_ context.Context, id string, req history.HistoryCopyRequest) (string, error) {
	if req.Token == "" {
		return "", history.ErrHistoryInvalidMutation
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return "", err
	}
	if req.TerminalID == "" {
		req.TerminalID = id
	}
	return terminal.HistoryCopy(req)
}

func (server *Server) TerminalHistoryCopyChunk(ctx context.Context, id string, req history.HistoryCopyChunkRequest) (history.HistoryCopyChunkResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Token == "" {
		return history.HistoryCopyChunkResult{}, history.ErrHistoryInvalidMutation
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return history.HistoryCopyChunkResult{}, err
	}
	if req.TerminalID == "" {
		req.TerminalID = id
	}
	return terminal.HistoryCopyChunk(ctx, req)
}

func (server *Server) TerminalHistorySearch(ctx context.Context, id string, req history.HistorySearchRequest) (history.HistorySearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Token == "" {
		return history.HistorySearchResult{}, history.ErrHistoryInvalidMutation
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return history.HistorySearchResult{}, err
	}
	if req.TerminalID == "" {
		req.TerminalID = id
	}
	return terminal.HistorySearch(ctx, req)
}

// TerminalHistoryFreeze 创建 core-v2 terminal 内部 frozen history boundary。它为
// protocol latest/copy 建立 core-owned token，后续 repaint 不得改写该 token 的复制边界。
func (server *Server) TerminalHistoryFreeze(ctx context.Context, id string, req history.FreezeHistoryRequest) (history.FrozenHistorySnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	terminal, err := server.Terminal(id)
	if err != nil {
		return history.FrozenHistorySnapshot{}, err
	}
	finishFlush := perftrace.Measure("core.server.history_freeze.flush")
	if err := terminal.FlushHistory(ctx); err != nil {
		finishFlush(0)
		return history.FrozenHistorySnapshot{}, err
	}
	finishFlush(0)
	if req.TerminalID == "" {
		req.TerminalID = id
	}
	finishStore := perftrace.Measure("core.server.history_freeze.terminal")
	snapshot, err := terminal.HistoryFreeze(req)
	finishStore(int(snapshot.CommittedUpperBound))
	return snapshot, err
}

func historyWindowPerfMode(mode history.HistoryWindowMode) string {
	switch mode {
	case "", history.HistoryWindowModeLatest:
		return "latest"
	case history.HistoryWindowModeOlder:
		return "older"
	case history.HistoryWindowModeNewer:
		return "newer"
	case history.HistoryWindowModeOldest:
		return "oldest"
	default:
		return "other"
	}
}

// TerminalHistoryRelease 释放 core-v2 authoritative history token。调用边界是
// protocol history.release；它只回收 frozen/window 资源，不删除 committed history truth。
func (server *Server) TerminalHistoryRelease(ctx context.Context, id string, token history.HistoryToken) error {
	_ = ctx
	terminal, err := server.Terminal(id)
	if err != nil {
		return err
	}
	return terminal.HistoryRelease(token)
}

func (server *Server) Events(ctx context.Context, filter EventFilter) <-chan Event {
	if ctx == nil {
		ctx = context.Background()
	}
	return server.events.subscribe(ctx, filter)
}

func (server *Server) ListenAndServe(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	server.wgMu.Lock()
	if server.closed.Load() {
		server.wgMu.Unlock()
		return ErrServerClosed
	}
	server.wg.Add(1)
	server.wgMu.Unlock()
	defer server.wg.Done()
	defer server.startShutdown()

	listener, err := server.cfg.listenerFactory(server.cfg.socketPath)
	if err != nil {
		return err
	}
	server.wgMu.Lock()
	server.mu.Lock()
	late := server.closed.Load()
	if !late {
		server.listeners = append(server.listeners, listener)
	}
	server.mu.Unlock()
	server.wgMu.Unlock()
	if late {
		_ = listener.Close()
		return ErrServerClosed
	}
	server.events.publish(Event{Type: EventServerListening, SocketPath: listener.Addr()})
	server.cfg.logger.Info("core-v2 server listening", "socket_path", listener.Addr())
	server.startHistoryRetention()
	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, transport.ErrListenerClosed) || server.closed.Load() {
				return nil
			}
			return err
		}
		if !server.startTransport(ctx, conn) {
			return nil
		}
	}
}

func (server *Server) ServeTransport(ctx context.Context, conn transport.Transport) error {
	return server.ServeScopedTransport(ctx, conn, fullDaemonTransportScope())
}

// TransportLifecycleObserver 是 core protocol transport 的最小生命周期观察边界。
// HelloAccepted 只表示合法 Hello response 已成功写入当前 transport；observer 不得读取业务 payload、
// 修改 core state 或决定授权，调用方必须自行保证回调快速且不阻塞 protocol loop。
type TransportLifecycleObserver interface {
	HelloAccepted()
}

// ServeScopedTransport 按已验证 scope 服务 transport，不暴露生命周期回调。
// remote managed session 需要 READY 证据时必须调用 ServeScopedTransportObserved。
func (server *Server) ServeScopedTransport(ctx context.Context, conn transport.Transport, scope TransportScope) error {
	return server.ServeScopedTransportObserved(ctx, conn, scope, nil)
}

// ServeScopedTransportObserved 按已验证 scope 服务 transport，并在 protocol Hello 真正接受后通知 observer。
// observer 只属于当前连接；Hello 前请求、重复 Hello 或发送失败都不会触发回调。
func (server *Server) ServeScopedTransportObserved(ctx context.Context, conn transport.Transport, scope TransportScope, observer TransportLifecycleObserver) error {
	if ctx == nil {
		ctx = context.Background()
	}
	scope = scope.normalized()
	if err := scope.validate(); err != nil {
		_ = conn.Close()
		return err
	}
	tracked, err := server.beginTrackedTransport(conn)
	if err != nil {
		return err
	}
	defer server.finishTrackedTransport(tracked)
	if err := server.admitTransport(ctx, tracked, scope); err != nil {
		return err
	}
	return server.serveTrackedTransportObserved(ctx, tracked, scope, observer)
}

func (server *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	server.startShutdown()
	select {
	case <-server.shutdownDone:
		return server.shutdownErr
	default:
	}
	select {
	case <-server.shutdownDone:
		return server.shutdownErr
	case <-ctx.Done():
		select {
		case <-server.shutdownDone:
			return server.shutdownErr
		default:
			return ctx.Err()
		}
	}
}

func (server *Server) startShutdown() {
	server.shutdownOnce.Do(func() {
		server.closed.Store(true)
		go server.runShutdown()
	})
}

func (server *Server) runShutdown() {
	var result error

	server.wgMu.Lock()
	server.mu.Lock()
	listeners := append([]transport.Listener(nil), server.listeners...)
	server.listeners = nil
	transports := make([]*trackedTransport, 0, len(server.transports))
	for conn := range server.transports {
		transports = append(transports, conn)
	}
	server.mu.Unlock()
	server.wgMu.Unlock()
	for _, listener := range listeners {
		result = errors.Join(result, listener.Close())
	}
	for _, conn := range transports {
		result = errors.Join(result, conn.closeWithReason(transportCloseShutdown))
	}

	server.lifecycleMu.Lock()
	server.mu.Lock()
	terminals := make([]*Terminal, 0, len(server.terminals))
	for _, terminal := range server.terminals {
		terminals = append(terminals, terminal)
	}
	server.terminals = make(map[string]*Terminal)
	server.mu.Unlock()
	server.lifecycleMu.Unlock()
	for _, terminal := range terminals {
		result = errors.Join(result, terminal.closeWithReason())
	}

	server.stopHistoryRetention()
	server.stopGrantOperations()
	server.outputBudget.close()
	server.wg.Wait()

	server.registry.clear()
	server.events.publish(Event{Type: EventServerStopped, SocketPath: server.cfg.socketPath})
	server.events.close()
	server.pruneGrantOperations()
	server.cfg.logger.Info("core-v2 server stopped", "socket_path", server.cfg.socketPath)
	server.shutdownErr = result
	close(server.shutdownDone)
}

func (server *Server) startHistoryRetention() {
	if server == nil || server.cfg.historyDisabled || server.cfg.historyStorage.MaxAge <= 0 {
		return
	}
	server.historyRetentionMu.Lock()
	if server.closed.Load() || server.historyRetentionCancel != nil {
		server.historyRetentionMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	server.historyRetentionCancel = cancel
	server.historyRetentionWG.Add(1)
	server.historyRetentionMu.Unlock()
	interval := server.cfg.historyStorage.MaxAge / 4
	if interval < time.Minute {
		interval = time.Minute
	}
	if interval > time.Hour {
		interval = time.Hour
	}
	go func() {
		defer server.historyRetentionWG.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := server.pruneActiveHistory(); err != nil {
					server.cfg.logger.Warn("history retention prune failed", "error", err)
				}
			}
		}
	}()
}

func (server *Server) stopHistoryRetention() {
	if server == nil {
		return
	}
	server.historyRetentionMu.Lock()
	cancel := server.historyRetentionCancel
	server.historyRetentionCancel = nil
	server.historyRetentionMu.Unlock()
	if cancel != nil {
		cancel()
		server.historyRetentionWG.Wait()
	}
}

func (server *Server) pruneActiveHistory() error {
	server.lifecycleMu.Lock()
	defer server.lifecycleMu.Unlock()
	if server.closed.Load() {
		return nil
	}
	server.mu.Lock()
	terminals := make([]*Terminal, 0, len(server.terminals))
	for _, terminal := range server.terminals {
		terminals = append(terminals, terminal)
	}
	server.mu.Unlock()
	var result error
	for _, terminal := range terminals {
		if err := terminal.pruneHistoryRetention(); err != nil {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (server *Server) removeTerminalHandle(id string) *Terminal {
	server.mu.Lock()
	defer server.mu.Unlock()
	terminal := server.terminals[id]
	delete(server.terminals, id)
	return terminal
}

func (server *Server) handleTransport(ctx context.Context, conn *trackedTransport) {
	defer server.finishTrackedTransport(conn)
	if err := server.serveTrackedTransport(ctx, conn, fullDaemonTransportScope()); err != nil && !errors.Is(err, transport.ErrListenerClosed) {
		server.cfg.logger.Debug("core-v2 protocol session stopped", "error", err)
	}
}

func (server *Server) serveTrackedTransport(ctx context.Context, conn *trackedTransport, scope TransportScope) error {
	return server.serveTrackedTransportObserved(ctx, conn, scope, nil)
}

func (server *Server) serveTrackedTransportObserved(ctx context.Context, conn *trackedTransport, scope TransportScope, observer TransportLifecycleObserver) error {
	session := newProtocolSessionObserved(server, conn, scope, observer)
	return session.run(ctx)
}

func (server *Server) startTransport(ctx context.Context, conn transport.Transport) bool {
	tracked, err := server.beginTrackedTransport(conn)
	if err != nil {
		return false
	}
	if err := server.admitTransport(ctx, tracked, fullDaemonTransportScope()); err != nil {
		server.finishTrackedTransport(tracked)
		return false
	}
	go server.handleTransport(ctx, tracked)
	return true
}

func (server *Server) publishTerminalEvent(typ EventType, info TerminalInfo) {
	terminal := info.Clone()
	server.events.publish(Event{
		Type:       typ,
		TerminalID: info.ID,
		Terminal:   &terminal,
	})
}

func (server *Server) publishStorageEvent(change StorageChanged) {
	server.events.publish(Event{Type: EventStorageChanged, Storage: &change})
}

func unixListenerFactory(socketPath string) (transport.Listener, error) {
	if socketPath == "" {
		return nil, fmt.Errorf("core-v2 server: empty socket path")
	}
	return unixtransport.NewListener(socketPath)
}

func defaultSocketPath() string {
	return runtimepath.SocketPath(fmt.Sprintf("anytty-v2-wire%d.sock", wire.Version))
}
