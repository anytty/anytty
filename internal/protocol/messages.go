package protocol

import "fmt"

const (
	ProtocolErrorCodeBadRequest        = 400
	ProtocolErrorCodeResourceExhausted = 429
)

// Hello 是 transport 建连后的最小版本握手 envelope。
type Hello struct {
	Version int
	Client  string
	Server  string
}

// Request 是 framing 层 control request；Method 的生产值只允许 api.execute。
type Request struct {
	ID     uint64
	Method string
	Params []byte
}

// Response 是 framing 层 control response，不解释 application result。
type Response struct {
	ID     uint64
	Result []byte
}

// ProtocolError 是 framing 层稳定错误码与脱敏消息。
type ProtocolError struct {
	Code    int
	Message string
}

// RequestError 是 daemon 对单次 control request 返回的稳定错误分类。
type RequestError struct {
	Code    int
	Message string
}

// Error 返回适合诊断的协议错误文本；机器判断必须读取 Code。
func (err *RequestError) Error() string {
	if err == nil {
		return "protocol request failed"
	}
	return fmt.Sprintf("protocol error %d: %s", err.Code, err.Message)
}

// PeerError reports a typed connection-level protocol violation by the remote peer.
type PeerError struct {
	Code    int
	Message string
}

func (err *PeerError) Error() string {
	if err == nil {
		return "protocol peer failed"
	}
	return fmt.Sprintf("protocol peer error %d: %s", err.Code, err.Message)
}

// ErrorMessage 是 framing 层带 correlation 的错误 envelope。
type ErrorMessage struct {
	ID    uint64
	Error ProtocolError
}
