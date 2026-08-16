//go:build windows

package securefs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestOpenOrCreatePrivateDirectoryUsesProtectedPrivateDACL(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private-state")
	handle, err := OpenOrCreatePrivateDirectory(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if err := ValidatePrivateDirectoryHandle(handle); err != nil {
		t.Fatalf("validate created directory: %v", err)
	}

	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(handle.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}
	owner, protected, rules, err := parsePrivateWindowsDescriptor(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	current, system, administrators, err := privateWindowsSIDs()
	if err != nil {
		t.Fatal(err)
	}
	if owner != current.String() || !protected || !privateAllowRulesAllowed(current.String(), system.String(), administrators.String(), rules) {
		t.Fatalf("created directory descriptor owner=%s protected=%v rules=%v", owner, protected, rules)
	}
	reopened, err := OpenOrCreatePrivateDirectory(directory)
	if err != nil {
		t.Fatalf("open existing private directory: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}

	temporary, err := os.CreateTemp(directory, "inherited-*")
	if err != nil {
		t.Fatal(err)
	}
	defer temporary.Close()
	temporaryDescriptor, err := windows.GetSecurityInfo(
		windows.Handle(temporary.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}
	temporaryOwner, _, temporaryRules, err := parsePrivateWindowsDescriptor(temporaryDescriptor)
	if err != nil {
		t.Fatal(err)
	}
	if temporaryOwner != current.String() || !privateAllowRulesAllowed(current.String(), system.String(), administrators.String(), temporaryRules) {
		t.Fatalf("temporary inherited an unsafe allow ACE: owner=%s rules=%v", temporaryOwner, temporaryRules)
	}
	if err := SecureFile(temporary.Name()); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateFileHandle(temporary); err != nil {
		t.Fatalf("validate secured temporary handle: %v", err)
	}
}

func TestOpenOrCreatePrivateDirectoryRejectsUnsafeExistingDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "unsafe-state")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	current, _, _, err := privateWindowsSIDs()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;OICI;FA;;;%s)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FR;;;WD)", current.String()))
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(directory, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenOrCreatePrivateDirectory(directory); err == nil {
		t.Fatal("existing directory with an Everyone allow ACE was accepted")
	}
}

func TestOpenOrCreatePrivateDirectoryRejectsReparsePoint(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(target); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "state-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("create Windows directory symlink: %v", err)
	}
	if _, err := OpenOrCreatePrivateDirectory(link); err == nil {
		t.Fatal("directory reparse point was accepted")
	}
}

func TestValidatePrivateWindowsDescriptorRejectsUnsafeOwnerAndAllowACE(t *testing.T) {
	current, _, _, err := privateWindowsSIDs()
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]string{
		"unsafe owner":   fmt.Sprintf("O:WDD:P(A;;FA;;;%s)(A;;FA;;;SY)(A;;FA;;;BA)", current.String()),
		"Everyone allow": fmt.Sprintf("O:%sD:P(A;;FA;;;%s)(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)", current.String(), current.String()),
	}
	for name, sddl := range tests {
		t.Run(name, func(t *testing.T) {
			descriptor, descriptorErr := windows.SecurityDescriptorFromString(sddl)
			if descriptorErr != nil {
				t.Fatal(descriptorErr)
			}
			if err := validatePrivateWindowsDescriptor(descriptor); err == nil {
				t.Fatal("unsafe descriptor was accepted")
			}
		})
	}
}

func TestValidatePrivateFileHandleRejectsAdditionalAllowACE(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SecureDirectory(directory); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "state")
	if err := os.WriteFile(path, []byte("state"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SecureFile(path); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePrivateFileHandle(file); err != nil {
		_ = file.Close()
		t.Fatalf("validate secured handle: %v", err)
	}
	_ = file.Close()

	current, _, _, err := privateWindowsSIDs()
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;;FA;;;%s)(A;;FA;;;SY)(A;;FA;;;BA)(A;;FR;;;WD)", current.String()))
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := ValidatePrivateFileHandle(file); err == nil {
		t.Fatal("additional Everyone allow ACE was accepted")
	}
}
