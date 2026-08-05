package port

import (
	"context"

	"github.com/anytty/anytty/tui/state"
)

// PathService 是 TUI prompt/workdir 查询 owning endpoint daemon 文件系统的 application port。
// 它只用于 prompt/workdir 这类 endpoint-scoped path completion 和创建默认值；
// 目录、默认 shell 与默认 cwd truth 来自 owning daemon 机器，service 不缓存、
// 不持久化，也不改写 terminal lifecycle。
type PathService interface {
	ListDirectories(context.Context, PathListDirectoriesRequest) (PathListDirectoriesResult, error)
	Defaults(context.Context, PathDefaultsRequest) (PathDefaultsResult, error)
}

// PathListDirectoriesRequest 描述一次目录候选查询。
// EndpointID 只在 client runtime adapter 层用于路由；进入单 daemon adapter 前必须剥离。
type PathListDirectoriesRequest struct {
	EndpointID state.EndpointID
	Prefix     string
	Limit      int
}

// PathDirectoryEntry 是可直接写回 prompt 的目录候选投影。
// Path 保留用户输入风格（例如 ~/ 或相对路径），Name 只用于排序和测试断言。
type PathDirectoryEntry struct {
	Name string
	Path string
}

// PathListDirectoriesResult 是目录候选查询结果。
// Missing 表示 base path 不存在或不可读，属于 prompt 空态；EndpointID 由 client runtime
// adapter 回填，确保异步结果不会覆盖其它 endpoint 的输入。
type PathListDirectoriesResult struct {
	EndpointID state.EndpointID
	BasePath   string
	Entries    []PathDirectoryEntry
	Missing    bool
	Truncated  bool
}

// PathDefaultsRequest 描述一次 endpoint 创建默认值查询。
// EndpointID 只在 client runtime adapter 层用于选择 daemon；进入 protocol adapter 前必须清空。
type PathDefaultsRequest struct {
	EndpointID state.EndpointID
}

// PathDefaultsResult 是 endpoint daemon 返回给 TUI 的创建默认值投影。
// DefaultCommand/DefaultCWD 来自 daemon 进程所在机器，TUI 不得用本地 SHELL 或 cwd 覆盖。
type PathDefaultsResult struct {
	EndpointID     state.EndpointID
	DefaultCommand []string
	DefaultCWD     string
}
