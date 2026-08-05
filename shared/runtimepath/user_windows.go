//go:build windows

package runtimepath

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/sys/windows"
)

func userDiscriminator() string {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || user == nil || user.User.Sid == nil {
		return "unknown-user"
	}
	sum := sha256.Sum256([]byte(user.User.Sid.String()))
	return hex.EncodeToString(sum[:8])
}
