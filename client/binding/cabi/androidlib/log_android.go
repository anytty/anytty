//go:build android && cgo

package main

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

import (
	"bytes"
	"log"
	"os"
	"sync"
	"unsafe"
)

var (
	cloudTimingPrefix      = []byte("anytty cloud connect ")
	cloudFailurePrefix     = []byte("anytty cloud failure ")
	cloudPresencePrefix    = []byte("anytty cloud presence ")
	directTimingPrefix     = []byte("anytty direct connect ")
	directFailurePrefix    = []byte("anytty direct failure ")
	webRTCDiagnosticPrefix = []byte("anytty webrtc ")
)

type androidTimingWriter struct{}

var androidDebugLog struct {
	sync.Mutex
	path string
}

//export anytty_debug_log_set_path
func anytty_debug_log_set_path(value *C.char) {
	androidDebugLog.Lock()
	androidDebugLog.path = C.GoString(value)
	androidDebugLog.Unlock()
}

func (androidTimingWriter) Write(payload []byte) (int, error) {
	if !bytes.HasPrefix(payload, cloudTimingPrefix) &&
		!bytes.HasPrefix(payload, cloudFailurePrefix) &&
		!bytes.HasPrefix(payload, cloudPresencePrefix) &&
		!bytes.HasPrefix(payload, directTimingPrefix) &&
		!bytes.HasPrefix(payload, directFailurePrefix) &&
		!bytes.HasPrefix(payload, webRTCDiagnosticPrefix) {
		return len(payload), nil
	}
	message := C.CString(string(bytes.TrimSpace(payload)))
	defer C.free(unsafe.Pointer(message))
	tag := C.CString("AnyTTYCloud")
	defer C.free(unsafe.Pointer(tag))
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, message)
	androidDebugLog.Lock()
	path := androidDebugLog.path
	if path != "" {
		if info, err := os.Stat(path); err == nil && info.Size() >= 2*1024*1024 {
			_ = os.Truncate(path, 0)
		}
		if file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			_, _ = file.Write(append(bytes.TrimSpace(payload), '\n'))
			_ = file.Close()
		}
	}
	androidDebugLog.Unlock()
	return len(payload), nil
}

// Android emits only structured, allowlisted connection diagnostics. Raw errors,
// identifiers, addresses, SDP, and credentials remain outside logcat.
func configureAndroidLogging() {
	log.SetFlags(0)
	log.SetOutput(androidTimingWriter{})
}
