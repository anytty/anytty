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
	"strconv"
	"sync"
	"time"
	"unsafe"
)

const (
	androidDiagnosticMaxFileBytes  = int64(512 * 1024)
	androidDiagnosticRetainedFiles = 4
	androidDiagnosticMaxEntryBytes = 4 * 1024
)

var (
	cloudTimingPrefix        = []byte("anytty cloud connect ")
	cloudFailurePrefix       = []byte("anytty cloud failure ")
	cloudPresencePrefix      = []byte("anytty cloud presence ")
	directTimingPrefix       = []byte("anytty direct connect ")
	directFailurePrefix      = []byte("anytty direct failure ")
	webRTCDiagnosticPrefix   = []byte("anytty webrtc ")
	endpointSupervisorPrefix = []byte("anytty endpoint_supervisor ")
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
		!bytes.HasPrefix(payload, webRTCDiagnosticPrefix) &&
		!bytes.HasPrefix(payload, endpointSupervisorPrefix) {
		return len(payload), nil
	}
	privatePayload := sanitizeAndroidDiagnostic(payload)
	if len(privatePayload) > androidDiagnosticMaxEntryBytes {
		privatePayload = privatePayload[:androidDiagnosticMaxEntryBytes]
	}
	message := C.CString(string(privatePayload))
	defer C.free(unsafe.Pointer(message))
	tag := C.CString("AnyTTYCloud")
	defer C.free(unsafe.Pointer(tag))
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, message)
	androidDebugLog.Lock()
	path := androidDebugLog.path
	if path != "" {
		line := append([]byte(time.Now().UTC().Format(time.RFC3339Nano)+" "), privatePayload...)
		rotateAndroidDiagnosticLog(path, int64(len(line)+1))
		if file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			_, _ = file.Write(append(line, '\n'))
			_ = file.Close()
		}
	}
	androidDebugLog.Unlock()
	return len(payload), nil
}

func rotateAndroidDiagnosticLog(path string, incomingBytes int64) {
	info, err := os.Stat(path)
	if err != nil || info.Size()+incomingBytes <= androidDiagnosticMaxFileBytes {
		return
	}
	_ = os.Remove(path + "." + strconv.Itoa(androidDiagnosticRetainedFiles-1))
	for index := androidDiagnosticRetainedFiles - 2; index >= 1; index-- {
		_ = os.Rename(path+"."+strconv.Itoa(index), path+"."+strconv.Itoa(index+1))
	}
	_ = os.Rename(path, path+".1")
}

func sanitizeAndroidDiagnostic(payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if !bytes.HasPrefix(payload, endpointSupervisorPrefix) {
		return bytes.Join(bytes.Fields(trimmed), []byte(" "))
	}
	if index := bytes.Index(trimmed, []byte(" invalidate_error=")); index >= 0 {
		trimmed = append(bytes.Clone(trimmed[:index]), []byte(" invalidate_failed=true")...)
	}
	fields := bytes.Fields(trimmed)
	filtered := fields[:0]
	for _, field := range fields {
		if bytes.HasPrefix(field, []byte("endpoint=")) {
			continue
		}
		filtered = append(filtered, field)
	}
	return bytes.Join(filtered, []byte(" "))
}

// Android emits only structured, allowlisted connection diagnostics. Raw errors,
// identifiers, addresses, SDP, and credentials remain outside logcat.
func configureAndroidLogging() {
	log.SetFlags(0)
	log.SetOutput(androidTimingWriter{})
}
