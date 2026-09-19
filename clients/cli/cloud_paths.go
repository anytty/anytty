package cli

import (
	accessruntime "github.com/anytty/anytty/access/runtime"
)

// v3CloudEnrollmentRecordPath 返回 Cloud enrollment record 路径。
func v3CloudEnrollmentRecordPath() string {
	return accessruntime.EnrollmentRecordPath()
}

// v3CloudDisabledPath 返回 Cloud runtime disabled marker 路径。
func v3CloudDisabledPath() string {
	return accessruntime.DisabledPath()
}
