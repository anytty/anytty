package webrtc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	directMuxFirstFrameTimeout = 5 * time.Second
	directMuxMaxPending        = 64
	directMuxMaxSignalFrame    = 4 << 20
	stunHeaderBytes            = 20
	stunMagicCookie            = 0x2112a442
)

// SplitDirectListener lets embedded signaling and ICE-TCP share one public TCP
// port. The first complete frame is inspected without consuming it, then the
// connection is handed to the existing protocol implementation unchanged.
func SplitDirectListener(listener net.Listener) (net.Listener, net.Listener, error) {
	if listener == nil {
		return nil, nil, fmt.Errorf("Direct protocol listener is required")
	}
	mux := &directProtocolMux{
		listener:  listener,
		signaling: make(chan net.Conn),
		ice:       make(chan net.Conn),
		done:      make(chan struct{}),
		pending:   make(chan struct{}, directMuxMaxPending),
		active:    make(map[net.Conn]struct{}),
	}
	mux.wg.Add(1)
	go mux.serve()
	return &directMuxListener{mux: mux, connections: mux.signaling}, &directMuxListener{mux: mux, connections: mux.ice}, nil
}

type directProtocolMux struct {
	listener  net.Listener
	signaling chan net.Conn
	ice       chan net.Conn
	done      chan struct{}
	pending   chan struct{}
	activeMu  sync.Mutex
	active    map[net.Conn]struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func (mux *directProtocolMux) serve() {
	defer mux.wg.Done()
	for {
		connection, err := mux.listener.Accept()
		if err != nil {
			mux.close()
			return
		}
		select {
		case mux.pending <- struct{}{}:
			mux.wg.Add(1)
			go mux.classify(connection)
		default:
			_ = connection.Close()
		}
	}
}

func (mux *directProtocolMux) classify(connection net.Conn) {
	defer mux.wg.Done()
	defer func() { <-mux.pending }()
	mux.activeMu.Lock()
	mux.active[connection] = struct{}{}
	mux.activeMu.Unlock()
	handedOff := false
	defer func() {
		mux.activeMu.Lock()
		delete(mux.active, connection)
		mux.activeMu.Unlock()
		if !handedOff {
			_ = connection.Close()
		}
	}()
	select {
	case <-mux.done:
		return
	default:
	}
	_ = connection.SetReadDeadline(time.Now().Add(directMuxFirstFrameTimeout))
	reader := bufio.NewReaderSize(connection, 512)
	header, err := reader.Peek(10)
	if err != nil {
		return
	}
	_ = connection.SetReadDeadline(time.Time{})
	buffered := &directBufferedConn{Conn: connection, reader: reader}
	var destination chan net.Conn
	switch {
	case isDirectICEFirstFrame(header):
		destination = mux.ice
	case isDirectSignalingFirstFrame(header):
		destination = mux.signaling
	default:
		return
	}
	select {
	case destination <- buffered:
		handedOff = true
	case <-mux.done:
	}
}

func (mux *directProtocolMux) close() {
	mux.closeOnce.Do(func() {
		close(mux.done)
		_ = mux.listener.Close()
		mux.activeMu.Lock()
		for connection := range mux.active {
			_ = connection.Close()
		}
		mux.activeMu.Unlock()
	})
}

func isDirectICEFirstFrame(header []byte) bool {
	if len(header) < 10 {
		return false
	}
	frameLength := int(binary.BigEndian.Uint16(header[:2]))
	messageType := binary.BigEndian.Uint16(header[2:4])
	messageLength := int(binary.BigEndian.Uint16(header[4:6]))
	return frameLength >= stunHeaderBytes && messageType&0xc000 == 0 &&
		messageLength+stunHeaderBytes == frameLength && binary.BigEndian.Uint32(header[6:10]) == stunMagicCookie
}

func isDirectSignalingFirstFrame(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	size := binary.BigEndian.Uint32(header[:4])
	return size > 0 && size <= directMuxMaxSignalFrame
}

type directMuxListener struct {
	mux         *directProtocolMux
	connections <-chan net.Conn
}

func (listener *directMuxListener) Accept() (net.Conn, error) {
	select {
	case connection := <-listener.connections:
		if connection == nil {
			return nil, net.ErrClosed
		}
		return connection, nil
	case <-listener.mux.done:
		return nil, net.ErrClosed
	}
}

func (listener *directMuxListener) Close() error {
	listener.mux.close()
	return nil
}

func (listener *directMuxListener) Addr() net.Addr { return listener.mux.listener.Addr() }

type directBufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (connection *directBufferedConn) Read(payload []byte) (int, error) {
	if connection.reader == nil {
		return 0, io.EOF
	}
	return connection.reader.Read(payload)
}

var _ net.Listener = (*directMuxListener)(nil)
var _ net.Conn = (*directBufferedConn)(nil)
