//go:build windows && wails

package platform

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	kernel32DLLPreferredLanguage                  = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleNamePreferredLanguage = kernel32DLLPreferredLanguage.NewProc("GetUserDefaultLocaleName")
)

func DetectPreferredLanguage() string {
	// LOCALE_NAME_MAX_LENGTH from Windows docs.
	buf := make([]uint16, 85)
	n, _, _ := procGetUserDefaultLocaleNamePreferredLanguage.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if n == 0 {
		return ""
	}
	locale := syscall.UTF16ToString(buf)
	return strings.TrimSpace(locale)
}
