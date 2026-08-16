//go:build !unix && !windows

package runtimepath

func userDiscriminator() string { return "unknown-user" }
