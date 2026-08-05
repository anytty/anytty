//go:build android && cgo

package main

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"log"
	"unsafe"
)

var cloudTimingPrefix = []byte("anytty cloud connect ")

type androidTimingWriter struct{}

func (androidTimingWriter) Write(payload []byte) (int, error) {
	if !bytes.HasPrefix(payload, cloudTimingPrefix) {
		return len(payload), nil
	}
	message := C.CString(string(bytes.TrimSpace(payload)))
	defer C.free(unsafe.Pointer(message))
	tag := C.CString("AnyTTYCloud")
	defer C.free(unsafe.Pointer(tag))
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, message)
	return len(payload), nil
}

// Android remains silent except for coarse Cloud stage timings. The allowlist keeps
// binding errors, Pion diagnostics, addresses, SDP, and credentials out of logcat.
func configureAndroidLogging() {
	log.SetFlags(0)
	log.SetOutput(androidTimingWriter{})
}
