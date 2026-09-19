package gateway

import (
	"bufio"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// pairPrefix is the optional pre-shared token pre-handshake. It is not part of
// the access wire: it is only spoken when an operator explicitly configures a
// pair token, so the default gateway path remains byte-transparent.
const pairPrefix = "ANYTTY-PAIR "

const (
	pairOK     = "ANYTTY-PAIR OK\n"
	pairDenied = "ANYTTY-PAIR DENIED\n"
)

var (
	errPairTokenMissing = errors.New("pair token required")
	errPairTokenInvalid = errors.New("pair token rejected")
)

// authorize enforces the optional pre-shared pair token. Without a configured
// token it returns the connection unchanged (the fast transparent path).
// With a token it reads exactly one line, replies OK/DENIED and returns a
// reader that replays any over-read protocol bytes to the proxy.
func (g *Gateway) authorize(client net.Conn) (io.Reader, error) {
	if len(g.cfg.PairToken) == 0 {
		return client, nil
	}
	reader := bufio.NewReader(client)
	_ = client.SetReadDeadline(time.Now().Add(g.cfg.HandshakeTimeout))
	line, err := reader.ReadSlice('\n')
	_ = client.SetReadDeadline(time.Time{})
	if err != nil {
		if errors.Is(err, bufio.ErrBufferFull) {
			return nil, fmt.Errorf("%w: pre-handshake line too long", errPairTokenInvalid)
		}
		return nil, fmt.Errorf("%w: %v", errPairTokenMissing, err)
	}
	token, ok := strings.CutPrefix(string(line), pairPrefix)
	if !ok {
		_ = writeAll(client, pairDenied)
		return nil, fmt.Errorf("%w: missing %q prefix", errPairTokenInvalid, strings.TrimSpace(pairPrefix))
	}
	token = strings.TrimRight(token, "\r\n")
	if subtle.ConstantTimeCompare([]byte(token), g.cfg.PairToken) != 1 {
		_ = writeAll(client, pairDenied)
		return nil, errPairTokenInvalid
	}
	if err := writeAll(client, pairOK); err != nil {
		return nil, fmt.Errorf("write pair acknowledgement: %w", err)
	}
	return reader, nil
}

// writeAll writes payload completely, treating a short write as an error.
func writeAll(writer io.Writer, payload string) error {
	for len(payload) > 0 {
		written, err := io.WriteString(writer, payload)
		if err != nil {
			return err
		}
		if written <= 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}
