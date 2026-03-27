//go:build windows

package windows

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"

	"pause/internal/logx"
)

var (
	combaseDLL         = syscall.NewLazyDLL("combase.dll")
	procRoInitialize   = combaseDLL.NewProc("RoInitialize")
	procRoUninitialize = combaseDLL.NewProc("RoUninitialize")
	iidToastManager    = ole.NewGUID("{50AC103F-D235-4598-BBEF-98FE4D1A3AD4}")
	iidToastFactory    = ole.NewGUID("{04124B20-82C6-4229-B109-FD9ED4662B53}")
	iidXMLDocumentIO   = ole.NewGUID("{6CD0E74E-EE65-4489-9EBF-CA43E87BA637}")
)

const (
	roInitSingleThreaded = 0
	roInitMultiThreaded  = 1
	rpcEChangedMode      = 0x80010106
	hResultNotFound      = 0x80070490

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

type toastNotificationFactory struct {
	ole.IInspectable
}

type toastNotificationFactoryVtbl struct {
	ole.IInspectableVtbl
	CreateToastNotification uintptr
}

type toastNotification struct {
	ole.IInspectable
}

type xmlDocument struct {
	ole.IInspectable
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

func (v *toastNotifier) Show(notification *toastNotification) error {
	hr, _, _ := syscall.SyscallN(
		v.VTable().Show,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(notification)),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *toastNotificationFactory) VTable() *toastNotificationFactoryVtbl {
	return (*toastNotificationFactoryVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *toastNotificationFactory) CreateToastNotification(doc *xmlDocument) (*toastNotification, error) {
	var notification *toastNotification
	hr, _, _ := syscall.SyscallN(
		v.VTable().CreateToastNotification,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(doc)),
		uintptr(unsafe.Pointer(&notification)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return notification, nil
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
			return fmt.Errorf("toast manager activation failed: %w", err)
		}
		defer manager.Release()

		notifier, err := manager.CreateToastNotifierWithID(appID)
		if err != nil {
			return fmt.Errorf("toast notifier creation failed: %w", err)
		}
		defer notifier.Release()

		setting, err = notifier.Setting()
		if err != nil {
			if isHResultNotFound(err) {
				logx.Infof("windows.notification.get_setting_not_found_assume_enabled app_id=%s", appID)
				setting = notificationSettingEnabled
				return nil
			}
			return fmt.Errorf("toast setting query failed: %w", err)
		}
		return nil
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
	target := "ms-settings:notifications"
	err := openWindowsURI(target)
	if err != nil {
		logx.Warnf("windows.notification.settings_open_failed target=%s err=%v", target, err)
		return err
	}
	return nil
}

func showWinRTToastReminderNative(appID, title, body string) error {
	payload, err := buildWindowsToastXML(title, body)
	if err != nil {
		return fmt.Errorf("toast xml build failed: %w", err)
	}

	err = withWindowsRuntime(func() error {
		manager, err := getToastNotificationManagerStatics()
		if err != nil {
			return fmt.Errorf("toast manager activation failed: %w", err)
		}
		defer manager.Release()

		notifier, err := manager.CreateToastNotifierWithID(appID)
		if err != nil {
			return fmt.Errorf("toast notifier creation failed: %w", err)
		}
		defer notifier.Release()

		doc, err := createToastXMLDocument(payload)
		if err != nil {
			return fmt.Errorf("toast xml document creation failed: %w", err)
		}
		defer doc.Release()

		factory, err := getToastNotificationFactory()
		if err != nil {
			return fmt.Errorf("toast factory activation failed: %w", err)
		}
		defer factory.Release()

		notification, err := factory.CreateToastNotification(doc)
		if err != nil {
			return fmt.Errorf("toast object creation failed: %w", err)
		}
		defer notification.Release()

		if err := notifier.Show(notification); err != nil {
			return fmt.Errorf("toast send failed: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
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

func isHResultNotFound(err error) bool {
	var oleErr *ole.OleError
	if errors.As(err, &oleErr) {
		return oleErr.Code() == uintptr(hResultNotFound)
	}
	return false
}

func getToastNotificationManagerStatics() (*toastNotificationManagerStatics, error) {
	factory, err := ole.RoGetActivationFactory("Windows.UI.Notifications.ToastNotificationManager", iidToastManager)
	if err != nil {
		return nil, err
	}
	return (*toastNotificationManagerStatics)(unsafe.Pointer(factory)), nil
}

func getToastNotificationFactory() (*toastNotificationFactory, error) {
	factory, err := ole.RoGetActivationFactory("Windows.UI.Notifications.ToastNotification", iidToastFactory)
	if err != nil {
		return nil, err
	}
	return (*toastNotificationFactory)(unsafe.Pointer(factory)), nil
}

func createToastXMLDocument(payload string) (*xmlDocument, error) {
	instance, err := ole.RoActivateInstance("Windows.Data.Xml.Dom.XmlDocument")
	if err != nil {
		return nil, err
	}

	doc := (*xmlDocument)(unsafe.Pointer(instance))
	var xmlIO *xmlDocumentIO
	if err := instance.PutQueryInterface(iidXMLDocumentIO, &xmlIO); err != nil {
		doc.Release()
		return nil, err
	}
	defer xmlIO.Release()

	if err := xmlIO.LoadXML(payload); err != nil {
		doc.Release()
		return nil, err
	}
	return doc, nil
}

func buildWindowsToastXML(title, body string) (string, error) {
	escapedTitle, err := escapeXMLText(title)
	if err != nil {
		return "", err
	}
	escapedBody, err := escapeXMLText(body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"<toast><visual><binding template=\"ToastGeneric\"><text>%s</text><text>%s</text></binding></visual></toast>",
		escapedTitle,
		escapedBody,
	), nil
}

func escapeXMLText(value string) (string, error) {
	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(value)); err != nil {
		return "", err
	}
	return buf.String(), nil
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
