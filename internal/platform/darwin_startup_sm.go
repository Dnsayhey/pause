//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement

#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#import <objc/message.h>
#import <stdlib.h>
#import <string.h>

enum {
	pauseSMModeUnavailable = 0,
	pauseSMModeMainAppService = 1,
	pauseSMModeLoginItem = 2
};

static char *pauseCopyCString(NSString *value) {
	if (value == nil) {
		return NULL;
	}
	const char *utf8 = [value UTF8String];
	if (utf8 == NULL) {
		return NULL;
	}
	size_t n = strlen(utf8) + 1;
	char *out = (char *)malloc(n);
	if (out == NULL) {
		return NULL;
	}
	memcpy(out, utf8, n);
	return out;
}

static void pauseSetError(char **errorOut, NSError *error, NSString *fallback) {
	if (errorOut == NULL) {
		return;
	}
	NSString *message = nil;
	if (error != nil) {
		message = [error localizedDescription];
	}
	if (message == nil || [message length] == 0) {
		message = fallback;
	}
	*errorOut = pauseCopyCString(message);
}

static int pauseSMStartupMode(void) {
	if (@available(macOS 13.0, *)) {
		Class cls = NSClassFromString(@"SMAppService");
		SEL mainSel = NSSelectorFromString(@"mainAppService");
		if (cls != Nil && [cls respondsToSelector:mainSel]) {
			return pauseSMModeMainAppService;
		}
	}
	return pauseSMModeLoginItem;
}

static id pauseSMMainAppService(NSError **error) {
	Class cls = NSClassFromString(@"SMAppService");
	SEL mainSel = NSSelectorFromString(@"mainAppService");
	if (cls == Nil || ![cls respondsToSelector:mainSel]) {
		if (error != NULL) {
			NSString *msg = @"SMAppService mainAppService is unavailable";
			*error = [NSError errorWithDomain:@"pause.sm" code:1 userInfo:@{NSLocalizedDescriptionKey: msg}];
		}
		return nil;
	}
	id (*send0)(id, SEL) = (id (*)(id, SEL))objc_msgSend;
	return send0(cls, mainSel);
}

static CFStringRef pauseCreateHelperID(const char *helperBundleID, char **errorOut) {
	if (helperBundleID == NULL || helperBundleID[0] == '\0') {
		if (errorOut != NULL) {
			*errorOut = pauseCopyCString(@"Legacy login item helper bundle id is empty");
		}
		return NULL;
	}
	CFStringRef helper = CFStringCreateWithCString(NULL, helperBundleID, kCFStringEncodingUTF8);
	if (helper == NULL && errorOut != NULL) {
		*errorOut = pauseCopyCString(@"Invalid helper bundle id");
	}
	return helper;
}

int pauseSMSetLaunchAtLogin(const char *helperBundleID, int enabled, char **errorOut) {
	@autoreleasepool {
		int mode = pauseSMStartupMode();
		if (mode == pauseSMModeMainAppService) {
			NSError *error = nil;
			id service = pauseSMMainAppService(&error);
			if (service == nil) {
				pauseSetError(errorOut, error, @"Failed to resolve SMAppService");
				return -1;
			}

			SEL selector = enabled ? NSSelectorFromString(@"registerAndReturnError:") : NSSelectorFromString(@"unregisterAndReturnError:");
			if (![service respondsToSelector:selector]) {
				NSString *msg = enabled ? @"SMAppService register API is unavailable" : @"SMAppService unregister API is unavailable";
				pauseSetError(errorOut, nil, msg);
				return -1;
			}

			BOOL (*sendErr)(id, SEL, NSError **) = (BOOL (*)(id, SEL, NSError **))objc_msgSend;
			BOOL ok = sendErr(service, selector, &error);
			if (!ok) {
				NSString *msg = enabled ? @"Failed to register launch at login" : @"Failed to unregister launch at login";
				pauseSetError(errorOut, error, msg);
				return -1;
			}
			return 0;
		}

		if (mode == pauseSMModeLoginItem) {
			CFStringRef helper = pauseCreateHelperID(helperBundleID, errorOut);
			if (helper == NULL) {
				return -1;
			}
			Boolean ok = SMLoginItemSetEnabled(helper, enabled ? true : false);
			CFRelease(helper);
			if (!ok) {
				NSString *msg = enabled
					? @"SMLoginItemSetEnabled failed to enable login item (helper may be missing or unsigned)"
					: @"SMLoginItemSetEnabled failed to disable login item";
				if (errorOut != NULL) {
					*errorOut = pauseCopyCString(msg);
				}
				return -1;
			}
			return 0;
		}

		if (errorOut != NULL) {
			*errorOut = pauseCopyCString(@"No supported startup API is available");
		}
		return -1;
	}
}

int pauseSMGetLaunchAtLogin(const char *helperBundleID, int *enabledOut, char **errorOut) {
	@autoreleasepool {
		int mode = pauseSMStartupMode();
		if (mode == pauseSMModeMainAppService) {
			NSError *error = nil;
			id service = pauseSMMainAppService(&error);
			if (service == nil) {
				pauseSetError(errorOut, error, @"Failed to resolve SMAppService");
				return -1;
			}

			SEL statusSel = NSSelectorFromString(@"status");
			if (![service respondsToSelector:statusSel]) {
				pauseSetError(errorOut, nil, @"SMAppService status API is unavailable");
				return -1;
			}

			NSInteger (*sendStatus)(id, SEL) = (NSInteger (*)(id, SEL))objc_msgSend;
			NSInteger status = sendStatus(service, statusSel);
			// SMAppServiceStatusEnabled = 1.
			if (enabledOut != NULL) {
				*enabledOut = (status == 1) ? 1 : 0;
			}
			return 0;
		}

		if (mode == pauseSMModeLoginItem) {
			CFStringRef helper = pauseCreateHelperID(helperBundleID, errorOut);
			if (helper == NULL) {
				return -1;
			}
			CFDictionaryRef job = SMJobCopyDictionary(kSMDomainUserLaunchd, helper);
			CFRelease(helper);
			if (enabledOut != NULL) {
				*enabledOut = (job != NULL) ? 1 : 0;
			}
			if (job != NULL) {
				CFRelease(job);
			}
			return 0;
		}

		if (errorOut != NULL) {
			*errorOut = pauseCopyCString(@"No supported startup API is available");
		}
		return -1;
	}
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type smMode int

const (
	smModeUnavailable smMode = 0
	smModeMainApp     smMode = 1
	smModeLoginItem   smMode = 2
)

func smStartupMode() smMode {
	return smMode(C.pauseSMStartupMode())
}

func smSetLaunchAtLogin(helperBundleID string, enabled bool) error {
	cHelper := C.CString(helperBundleID)
	defer C.free(unsafe.Pointer(cHelper))
	var cErr *C.char
	flag := C.int(0)
	if enabled {
		flag = 1
	}
	rc := C.pauseSMSetLaunchAtLogin(cHelper, flag, &cErr)
	defer freeCString(cErr)
	if rc != 0 {
		return fmt.Errorf("ServiceManagement set launch-at-login failed: %s", cStringOrDefault(cErr, "unknown error"))
	}
	return nil
}

func smGetLaunchAtLogin(helperBundleID string) (bool, error) {
	cHelper := C.CString(helperBundleID)
	defer C.free(unsafe.Pointer(cHelper))
	var enabled C.int
	var cErr *C.char
	rc := C.pauseSMGetLaunchAtLogin(cHelper, &enabled, &cErr)
	defer freeCString(cErr)
	if rc != 0 {
		return false, fmt.Errorf("ServiceManagement get launch-at-login failed: %s", cStringOrDefault(cErr, "unknown error"))
	}
	return enabled == 1, nil
}

func freeCString(ptr *C.char) {
	if ptr == nil {
		return
	}
	C.free(unsafe.Pointer(ptr))
}

func cStringOrDefault(ptr *C.char, fallback string) string {
	if ptr == nil {
		return fallback
	}
	msg := C.GoString(ptr)
	if msg == "" {
		return fallback
	}
	return msg
}
