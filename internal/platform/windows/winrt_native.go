//go:build windows

package windows

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
)

var (
	combaseDLL         = syscall.NewLazyDLL("combase.dll")
	procRoInitialize   = combaseDLL.NewProc("RoInitialize")
	procRoUninitialize = combaseDLL.NewProc("RoUninitialize")
	iidToastManager    = ole.NewGUID("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
	iidXMLDocumentIO   = ole.NewGUID("{6CD0E74E-EE65-4489-9EBF-CA43E87BA637}")
)

const (
	roInitSingleThreaded = 0
	roInitMultiThreaded  = 1
	rpcEChangedMode      = 0x80010106

	shellExecuteSuccessThreshold = 32
	swShowNormal                 = 1
)

type notificationSetting uint32

const (
	notificationSettingEnabled notificationSetting = iota
	notificationSettingDisabledForApplication
	notificationSettingDisabledForUser
	notificationSettingDisabledByGroupPolicy
	notificationSettingDisabledForManifest
)

type toastNotificationManagerStatics struct {
	ole.IInspectable
}

type toastNotificationManagerStaticsVtbl struct {
	ole.IInspectableVtbl
	CreateToastNotifier       uintptr
	CreateToastNotifierWithID uintptr
	GetHistory                uintptr
	GetTemplateContent        uintptr
}

type toastNotifier struct {
	ole.IInspectable
}

type toastNotifierVtbl struct {
	ole.IInspectableVtbl
	Show               uintptr
	Hide               uintptr
	GetSetting         uintptr
	AddToSchedule      uintptr
	RemoveFromSchedule uintptr
}

type xmlDocumentIO struct {
	ole.IInspectable
}

type xmlDocumentIOVtbl struct {
	ole.IInspectableVtbl
	LoadXML             uintptr
	LoadXMLWithSettings uintptr
}

func (v *toastNotificationManagerStatics) VTable() *toastNotificationManagerStaticsVtbl {
	return (*toastNotificationManagerStaticsVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotificationManagerStatics) CreateToastNotifierWithID(appID string) (*toastNotifier, error) {
	hstring, err := ole.NewHString(appID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = ole.DeleteHString(hstring)
	}()

	var notifier *toastNotifier
	hr, _, _ := syscall.SyscallN(
		v.VTable().CreateToastNotifierWithID,
		uintptr(unsafe.Pointer(v)),
		uintptr(hstring),
		uintptr(unsafe.Pointer(&notifier)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return notifier, nil
}

func (v *toastNotifier) VTable() *toastNotifierVtbl {
	return (*toastNotifierVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotifier) Setting() (notificationSetting, error) {
	var setting notificationSetting
	hr, _, _ := syscall.SyscallN(
		v.VTable().GetSetting,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&setting)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return setting, nil
}

func (v *xmlDocumentIO) VTable() *xmlDocumentIOVtbl {
	return (*xmlDocumentIOVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *xmlDocumentIO) LoadXML(value string) error {
	hstring, err := ole.NewHString(value)
	if err != nil {
		return err
	}
	defer func() {
		_ = ole.DeleteHString(hstring)
	}()

	hr, _, _ := syscall.SyscallN(
		v.VTable().LoadXML,
		uintptr(unsafe.Pointer(v)),
		uintptr(hstring),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func queryWindowsToastSettingNative(appID string) (string, error) {
	var setting notificationSetting
	err := withWindowsRuntime(func() error {
		manager, err := getToastNotificationManagerStatics()
		if err != nil {
			return err
		}
		defer manager.Release()

		notifier, err := manager.CreateToastNotifierWithID(appID)
		if err != nil {
			return err
		}
		defer notifier.Release()

		setting, err = notifier.Setting()
		return err
	})
	if err != nil {
		return "", err
	}

	switch setting {
	case notificationSettingEnabled:
		return "Enabled", nil
	case notificationSettingDisabledForApplication:
		return "DisabledForApplication", nil
	case notificationSettingDisabledForUser:
		return "DisabledForUser", nil
	case notificationSettingDisabledByGroupPolicy:
		return "DisabledByGroupPolicy", nil
	case notificationSettingDisabledForManifest:
		return "DisabledForManifest", nil
	default:
		return fmt.Sprintf("Unknown(%d)", setting), nil
	}
}

func openWindowsNotificationSettingsNative() error {
	return openWindowsURI("ms-settings:notifications")
}

func openWindowsURI(target string) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		swShowNormal,
	)
	if ret <= shellExecuteSuccessThreshold {
		return fmt.Errorf("ShellExecuteW failed: %d", ret)
	}
	return nil
}

func getToastNotificationManagerStatics() (*toastNotificationManagerStatics, error) {
	factory, err := ole.RoGetActivationFactory("Windows.UI.Notifications.ToastNotificationManager", iidToastManager)
	if err != nil {
		return nil, err
	}
	return (*toastNotificationManagerStatics)(unsafe.Pointer(factory)), nil
}

func withWindowsRuntime(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	uninit, err := roInitialize()
	if err != nil {
		return err
	}
	if uninit {
		defer roUninitialize()
	}
	return fn()
}

func roInitialize() (bool, error) {
	hr, _, _ := procRoInitialize.Call(roInitMultiThreaded)
	switch uint32(hr) {
	case 0, 1:
		return true, nil
	case rpcEChangedMode:
		return false, nil
	default:
		return false, ole.NewError(hr)
	}
}

func roUninitialize() {
	procRoUninitialize.Call()
}
