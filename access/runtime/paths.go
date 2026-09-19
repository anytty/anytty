package accessruntime

import (
	"path/filepath"

	"github.com/anytty/anytty/access/localstate"
)

// EnrollmentRecordPath 返回 Cloud enrollment record 路径。
func EnrollmentRecordPath() string {
	return filepath.Join(localstate.RemoteIdentityDir(), "cloud_enrollment.json")
}

// DisabledPath 返回 Cloud runtime disabled marker 路径。
func DisabledPath() string {
	return filepath.Join(localstate.RemoteIdentityDir(), "cloud_disabled.json")
}
