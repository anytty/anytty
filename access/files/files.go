package files

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/anytty/anytty/access/files/transfer"
	"github.com/anytty/anytty/shared/userdirs"
)

// Config 是 access 文件服务装配输入。
type Config struct {
	// Resolver 配置统一路径解析（roots/BaseDir/Home/大小写）。
	Resolver ResolverConfig
	// TransferDir 是断点续传记录的持久化目录；空时使用
	// $XDG_STATE_HOME/anytty/transfers，保证跨进程重启可续。
	TransferDir string
	// Logger 缺省用 slog.Default。
	Logger *slog.Logger
}

// Service 是 access-local 文件服务：metadata 操作 + 可续传 transfer。
// 它是 server 级单例，跨 session 共享 upload 记录与断点续传 token。
type Service struct {
	resolver  *Resolver
	store     *transfer.Store
	logger    *slog.Logger
	estimator *estimator

	mu        sync.Mutex
	uploads   map[string]*upload
	downloads map[string]*download
}

// NewService 创建文件服务并打开持久化 transfer 记录。
func NewService(config Config) (*Service, error) {
	resolver, err := NewResolver(config.Resolver)
	if err != nil {
		return nil, err
	}
	dir := config.TransferDir
	if dir == "" {
		dir = defaultTransferDir()
	}
	store, err := transfer.Open(dir)
	if err != nil {
		return nil, err
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	service := &Service{
		resolver:  resolver,
		store:     store,
		logger:    logger,
		estimator: newEstimator(),
		uploads:   make(map[string]*upload),
		downloads: make(map[string]*download),
	}
	service.pruneExpired(time.Now().UTC())
	return service, nil
}

// Resolver 返回统一路径解析器。
func (service *Service) Resolver() *Resolver {
	if service == nil {
		return nil
	}
	return service.resolver
}

// StorePath 返回断点续传记录文件路径（诊断用）。
func (service *Service) StorePath() string {
	if service == nil || service.store == nil {
		return ""
	}
	return service.store.Path()
}

// Close 停止全部活动 transfer；未完成 upload 的断点记录保留，供重启后续传。
func (service *Service) Close() error {
	if service == nil {
		return nil
	}
	for _, active := range service.snapshotUploads() {
		_ = active.Close()
	}
	for _, active := range service.snapshotDownloads() {
		_ = active.Close()
	}
	return nil
}

// pruneExpired 清理过期 upload 记录与其临时文件。
func (service *Service) pruneExpired(now time.Time) int {
	removed := 0
	for _, record := range service.store.All() {
		if record.Direction != transfer.DirectionUpload || !record.Expired(now) {
			continue
		}
		service.mu.Lock()
		_, active := service.uploads[record.ID]
		service.mu.Unlock()
		if active {
			continue
		}
		if record.TempPath != "" {
			_ = os.Remove(record.TempPath)
		}
		if err := service.store.Delete(record.ID); err == nil {
			removed++
		}
	}
	return removed
}

func defaultTransferDir() string {
	return filepath.Join(userdirs.StateHome(), "anytty", "transfers")
}
