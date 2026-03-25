//go:build wails && windows

package app

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32DLL                  = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32DLL.NewProc("GetUserDefaultLocaleName")
)

func detectPreferredLanguage() string {
	// LOCALE_NAME_MAX_LENGTH from Windows docs.
	buf := make([]uint16, 85)
	n, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if n == 0 {
		return ""
	}
	locale := syscall.UTF16ToString(buf)
	return strings.TrimSpace(locale)
}
