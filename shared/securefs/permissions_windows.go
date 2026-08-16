//go:build windows

// Package securefs 统一私有状态文件的平台权限边界。
package securefs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	accessAllowedCompoundACEType       = 0x4
	accessAllowedObjectACEType         = 0x5
	accessAllowedCallbackACEType       = 0x9
	accessAllowedCallbackObjectACEType = 0xb
)

// SecureDirectory 为 Windows 私有目录写入受保护且可继承的 DACL。
// 子项只允许当前用户、SYSTEM 和本机 Administrators，不能继承宽松的父目录写权限。
func SecureDirectory(path string) error { return secureWindowsPath(path, true) }

// OpenOrCreatePrivateDirectory establishes one app-owned private directory.
// New directories receive their protected DACL as part of CreateDirectory;
// existing directories are never repaired and must pass same-handle checks.
func OpenOrCreatePrivateDirectory(path string) (*os.File, error) {
	path = filepath.Clean(path)
	descriptor, err := privateWindowsSecurityDescriptor(true)
	if err != nil {
		return nil, err
	}
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	attributes := &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	createErr := windows.CreateDirectory(pathPointer, attributes)
	if createErr != nil && !errors.Is(createErr, windows.ERROR_ALREADY_EXISTS) {
		return nil, createErr
	}
	handle, err := windows.CreateFile(
		pathPointer,
		windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	directory := os.NewFile(uintptr(handle), path)
	if directory == nil {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("create private directory file handle")
	}
	if err := ValidatePrivateDirectoryHandle(directory); err != nil {
		_ = directory.Close()
		return nil, err
	}
	return directory, nil
}

var reOpenFile = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")

// SecureDirectoryHandle writes a protected private DACL through the identity
// of an already-open directory rather than resolving its path again.
func SecureDirectoryHandle(directory *os.File) error {
	return secureWindowsHandle(directory, true)
}

// SecureFile 为 Windows 私钥、credential 或 runtime record 写入受保护 DACL。
// 权限失败时调用方必须停止使用该秘密，不能回退到 chmod 的只读位映射。
func SecureFile(path string) error { return secureWindowsPath(path, false) }

// SecureFileHandle writes a protected private DACL through the identity of an
// already-open file rather than resolving its path again.
func SecureFileHandle(file *os.File) error {
	return secureWindowsHandle(file, false)
}

func secureWindowsHandle(file *os.File, directory bool) error {
	if file == nil {
		return errors.New("private object handle is required")
	}
	descriptor, err := privateWindowsSecurityDescriptor(directory)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	flags := uint32(0)
	if directory {
		flags = windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	handle, _, callErr := reOpenFile.Call(
		file.Fd(),
		uintptr(windows.WRITE_DAC),
		uintptr(windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE),
		uintptr(flags),
	)
	if windows.Handle(handle) == windows.InvalidHandle {
		if callErr != nil {
			return callErr
		}
		return errors.New("reopen private object handle for DACL update")
	}
	reopened := windows.Handle(handle)
	defer windows.CloseHandle(reopened)
	return windows.SetSecurityInfo(
		reopened,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

// IsPrivateFile verifies the opened file rather than querying security metadata by path.
func IsPrivateFile(path string, info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	return err == nil && os.SameFile(info, openedInfo) && ValidatePrivateFileHandle(file) == nil
}

// ValidatePrivateFileHandle validates owner and protected DACL on the supplied handle.
// Only current-user, SYSTEM, and Administrators allow ACEs are accepted.
func ValidatePrivateFileHandle(file *os.File) error {
	if file == nil {
		return errors.New("private file handle is required")
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("private file handle must reference a regular file")
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return err
	}
	return validatePrivateWindowsDescriptor(descriptor)
}

// ValidatePrivateDirectoryHandle validates owner, reparse state, and protected DACL.
func ValidatePrivateDirectoryHandle(directory *os.File) error {
	if directory == nil {
		return errors.New("private directory handle is required")
	}
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(directory.Fd()), &information); err != nil {
		return err
	}
	if information.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return errors.New("private directory handle must reference a directory")
	}
	if information.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return errors.New("private directory must not be a reparse point")
	}
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(directory.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return err
	}
	return validatePrivateWindowsDescriptor(descriptor)
}

func validatePrivateWindowsDescriptor(descriptor *windows.SECURITY_DESCRIPTOR) error {
	current, system, administrators, err := privateWindowsSIDs()
	if err != nil {
		return err
	}
	owner, protected, rules, err := parsePrivateWindowsDescriptor(descriptor)
	if err != nil {
		return err
	}
	if !privateDescriptorAllowed(owner, current.String(), system.String(), administrators.String(), protected, rules) {
		return errors.New("private object owner or protected DACL is unsafe")
	}
	return nil
}

func parsePrivateWindowsDescriptor(descriptor *windows.SECURITY_DESCRIPTOR) (string, bool, []privateAccessRule, error) {
	if descriptor == nil || !descriptor.IsValid() {
		return "", false, nil, errors.New("private object security descriptor is invalid")
	}
	owner, _, err := descriptor.Owner()
	if err != nil || owner == nil {
		return "", false, nil, errors.New("private object owner is unavailable")
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return "", false, nil, err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil {
		return "", false, nil, errors.New("private object DACL is unavailable")
	}
	rules := make([]privateAccessRule, 0, dacl.AceCount)
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil || ace == nil {
			return "", false, nil, errors.New("read private object DACL ACE")
		}
		switch ace.Header.AceType {
		case windows.ACCESS_ALLOWED_ACE_TYPE:
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
			if !sid.IsValid() {
				return "", false, nil, errors.New("private object DACL contains an invalid SID")
			}
			rules = append(rules, privateAccessRule{allow: true, trustee: sid.String()})
		case accessAllowedCompoundACEType, accessAllowedObjectACEType, accessAllowedCallbackACEType, accessAllowedCallbackObjectACEType:
			return "", false, nil, errors.New("private object DACL contains an unsupported allow ACE")
		default:
			rules = append(rules, privateAccessRule{allow: false})
		}
	}
	return owner.String(), control&windows.SE_DACL_PROTECTED != 0, rules, nil
}

func secureWindowsPath(path string, directory bool) error {
	descriptor, err := privateWindowsSecurityDescriptor(directory)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

func privateWindowsSecurityDescriptor(directory bool) (*windows.SECURITY_DESCRIPTOR, error) {
	current, _, _, err := privateWindowsSIDs()
	if err != nil {
		return nil, err
	}
	inheritance := ""
	if directory {
		inheritance = "OICI"
	}
	return windows.SecurityDescriptorFromString(fmt.Sprintf(
		"O:%sD:P(A;%s;FA;;;%s)(A;%s;FA;;;SY)(A;%s;FA;;;BA)",
		current.String(),
		inheritance,
		current.String(),
		inheritance,
		inheritance,
	))
}

func privateWindowsSIDs() (*windows.SID, *windows.SID, *windows.SID, error) {
	currentUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || currentUser == nil || currentUser.User.Sid == nil {
		return nil, nil, nil, fmt.Errorf("resolve current Windows user SID: %w", err)
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return nil, nil, nil, err
	}
	administrators, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, nil, nil, err
	}
	return currentUser.User.Sid, system, administrators, nil
}
