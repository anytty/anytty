//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func prepareUpdateExecutable(path string) error {
	output, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--identifier", "com.anytty.cli", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("apply local macOS code signature: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
