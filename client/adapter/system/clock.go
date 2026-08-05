// Package system 提供 native Go host 使用的无领域状态平台 primitive。
package system

import (
	"time"

	"github.com/anytty/anytty/client/port"
)

// Clock 是 native CLI/Desktop composition 注入 client runtime 的系统时间 primitive。
// runtime 只能通过该对象安排 hedge timer；它不拥有 endpoint、route、session 或重连状态。
type Clock struct{}

// Now 返回当前 UTC 时间。
func (Clock) Now() time.Time { return time.Now().UTC() }

// NewTimer 创建可停止的单次系统 timer。
func (Clock) NewTimer(delay time.Duration) port.Timer { return &timer{value: time.NewTimer(delay)} }

type timer struct{ value *time.Timer }

// C 返回系统 timer 的单次触发 channel。
func (value *timer) C() <-chan time.Time { return value.value.C }

// Stop 幂等停止尚未触发的 timer。
func (value *timer) Stop() bool { return value.value.Stop() }

var _ port.Clock = Clock{}
