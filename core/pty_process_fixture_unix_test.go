//go:build !windows

package core

func ptyInteractiveFixture() ([]string, []byte) {
	return []string{"/bin/sh", "-c", "printf 'alpha\\n'; read line; printf \"echo:%s\\n\" \"$line\""}, []byte("beta\n")
}

func ptyEnvironmentFixture() []string {
	return []string{"/bin/sh", "-c", "printf 'cwd:%s env:%s\\n' \"$PWD\" \"$ANYTTY_REMOTE_TEST\""}
}

func ptyLongRunningFixture() []string {
	return []string{"/bin/sh", "-c", "while true; do sleep 1; done"}
}
