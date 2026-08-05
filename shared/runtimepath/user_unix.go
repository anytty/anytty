//go:build unix

package runtimepath

import (
	"os"
	"strconv"
)

func userDiscriminator() string { return strconv.Itoa(os.Getuid()) }
