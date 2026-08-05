package port

import "time"

// Clock 是 client runtime 使用的时间来源。
// route hedge、deadline projection 和测试时间只能通过该接口取得，runtime 不直接读取系统时钟。
type Clock interface {
	Now() time.Time
	NewTimer(time.Duration) Timer
}

// Timer 是 Clock 创建的单次计时资源。
// Stop 必须幂等；runtime 取消 route race 时必须停止未触发 timer，避免后台资源泄漏。
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}
