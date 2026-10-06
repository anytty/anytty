package core

// 所有权边界（架构审查 ②）：
//   - CLI 拥有主机布局：负责解析 history storage 目录与 obsolete compact 目录
//     （XDG state 等路径规则），只把目录交给本类型；
//   - pool/core 拥有 history store 语义：文件规则、清理范围、编码/保留策略都
//     封装在 Prepare/DeleteAll/DeleteTerminal/DeleteObsolete 之后，CLI 不再直接
//     调用 linehist 或裸 history 删除函数，避免 on-disk layout 变成
//     CLI<->pool 的隐式契约。
//
// 离线维护要求 pool 已停止：调用方必须通过 pool runtime record 锁确认没有
// 活跃 store 在写这些目录（CLI 的 history delete/prune 即如此）。
type HistoryMaintenance struct {
	Dir string
}

// NewHistoryMaintenance 绑定 pool 的 history storage 目录。Dir 为空时各方法
// 与原底层函数一致返回 os.ErrInvalid。
func NewHistoryMaintenance(dir string) HistoryMaintenance {
	return HistoryMaintenance{Dir: dir}
}

// Prepare 在 pool 单实例锁内清理旧格式，并把当前块文件收敛到 config 描述的
// 按 terminal 上限/期限/编码。旧格式明确不迁移。
func (m HistoryMaintenance) Prepare(config HistoryStorageConfig) error {
	return PrepareHistoryStorage(m.Dir, config)
}

// DeleteAll 删除 history storage 目录内的全部 terminal history 文件，
// 返回实际删除的文件数。
func (m HistoryMaintenance) DeleteAll() (int, error) {
	return DeleteAllHistory(m.Dir)
}

// DeleteTerminal 删除指定 terminal 的全部 history 文件，返回实际删除的文件数。
func (m HistoryMaintenance) DeleteTerminal(terminalID string) (int, error) {
	return DeleteTerminalHistory(m.Dir, terminalID)
}

// DeleteObsolete 删除 obsoleteDir 下早期 core-v2 history 的 .compact 文件，
// 返回实际删除的文件数。obsoleteDir 由调用方按主机布局解析；本方法不递归，
// 也不删除未知文件。
func (m HistoryMaintenance) DeleteObsolete(obsoleteDir string) (int, error) {
	return DeleteObsoleteCompactHistory(obsoleteDir)
}
