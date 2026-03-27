//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications -framework AppKit

#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <dispatch/dispatch.h>
#import <stdlib.h>
#import <string.h>

extern void pauseDarwinCaptureAuthorizationResult(int requestID, int granted, char *error);

static char *pauseNotifyCopyCString(NSString *value) {
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

static void pauseNotifySetError(char **errorOut, NSString *message) {
	if (errorOut == NULL) {
		return;
	}
	*errorOut = pauseNotifyCopyCString(message);
}

static void pauseRunOnMainSync(dispatch_block_t block) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_sync(dispatch_get_main_queue(), block);
}

static void pauseRunOnMainAsync(dispatch_block_t block) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_async(dispatch_get_main_queue(), block);
}

@interface PauseNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation PauseNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
	 didReceiveNotificationResponse:(UNNotificationResponse *)response
		  withCompletionHandler:(void (^)(void))completionHandler {
	(void)center;
	(void)response;
	if (completionHandler != nil) {
		completionHandler();
	}
}
@end

static PauseNotificationDelegate *pauseNotificationDelegate = nil;

static int pauseDarwinInstallNotificationDelegate(char **errorOut) {
	@autoreleasepool {
		if (!@available(macOS 10.14, *)) {
			pauseNotifySetError(errorOut, @"UserNotifications framework requires macOS 10.14+");
			return -1;
		}
		__block BOOL installed = NO;
		pauseRunOnMainSync(^{
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
			if (center == nil) {
				return;
			}
			if (pauseNotificationDelegate == nil) {
				pauseNotificationDelegate = [PauseNotificationDelegate new];
			}
			center.delegate = pauseNotificationDelegate;
			installed = YES;
		});
		if (!installed) {
			pauseNotifySetError(errorOut, @"UNUserNotificationCenter unavailable");
			return -1;
		}
		return 0;
	}
}

static int pauseDarwinGetAuthorizationStatus(int *statusOut, char **errorOut) {
	@autoreleasepool {
		if (!@available(macOS 10.14, *)) {
			pauseNotifySetError(errorOut, @"UserNotifications framework requires macOS 10.14+");
			return -1;
		}
		__block UNUserNotificationCenter *center = nil;
		pauseRunOnMainSync(^{
			center = [UNUserNotificationCenter currentNotificationCenter];
		});
		if (center == nil) {
			pauseNotifySetError(errorOut, @"UNUserNotificationCenter unavailable");
			return -1;
		}

		__block UNAuthorizationStatus authStatus = UNAuthorizationStatusNotDetermined;
		__block BOOL loaded = NO;
		dispatch_semaphore_t sem = dispatch_semaphore_create(0);
		pauseRunOnMainSync(^{
			[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
				if (settings != nil) {
					authStatus = settings.authorizationStatus;
					loaded = YES;
				}
				dispatch_semaphore_signal(sem);
			}];
		});
		long waitResult = dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
		if (waitResult != 0 || !loaded) {
			pauseNotifySetError(errorOut, @"Notification settings request timed out");
			return -1;
		}
		if (statusOut != NULL) {
			*statusOut = (int)authStatus;
		}
		return 0;
	}
}

static int pauseDarwinRequestAuthorizationAsync(int requestID, char **errorOut) {
	@autoreleasepool {
		if (!@available(macOS 10.14, *)) {
			pauseNotifySetError(errorOut, @"UserNotifications framework requires macOS 10.14+");
			return -1;
		}
		pauseRunOnMainAsync(^{
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
			if (center == nil) {
				char *message = pauseNotifyCopyCString(@"UNUserNotificationCenter unavailable");
				pauseDarwinCaptureAuthorizationResult(requestID, 0, message);
				if (message != NULL) {
					free(message);
				}
				return;
			}

			[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
			                      completionHandler:^(BOOL localGranted, NSError *localErr) {
				NSString *message = nil;
				if (localErr != nil) {
					if (!localGranted &&
					    [[localErr domain] isEqualToString:UNErrorDomain] &&
					    [localErr code] == UNErrorCodeNotificationsNotAllowed) {
						localErr = nil;
					} else {
						message = [localErr localizedDescription];
					}
				}
				char *copied = pauseNotifyCopyCString(message);
				pauseDarwinCaptureAuthorizationResult(requestID, localGranted ? 1 : 0, copied);
				if (copied != NULL) {
					free(copied);
				}
			}];
		});
		return 0;
	}
}

static int pauseDarwinOpenNotificationSettings(const char *bundleIDC, char **errorOut) {
	@autoreleasepool {
		NSString *bundleID = @"";
		if (bundleIDC != NULL && bundleIDC[0] != '\0') {
			NSString *tmp = [NSString stringWithUTF8String:bundleIDC];
			if (tmp != nil) {
				bundleID = tmp;
			}
		}

		NSMutableArray<NSString *> *candidates = [NSMutableArray array];
		if ([bundleID length] > 0) {
			[candidates addObject:[NSString stringWithFormat:@"x-apple.systempreferences:com.apple.Notifications-Settings.extension?bundleIdentifier=%@", bundleID]];
		}
		[candidates addObject:@"x-apple.systempreferences:com.apple.preference.notifications"];

		__block BOOL opened = NO;
		pauseRunOnMainSync(^{
			for (NSString *candidate in candidates) {
				NSURL *url = [NSURL URLWithString:candidate];
				if (url == nil) {
					continue;
				}
				if ([[NSWorkspace sharedWorkspace] openURL:url]) {
					opened = YES;
					break;
				}
			}
		});
		if (!opened) {
			pauseNotifySetError(errorOut, @"Failed to open notification settings");
			return -1;
		}
		return 0;
	}
}

static int pauseDarwinShowUserNotification(const char *titleC, const char *bodyC, char **errorOut) {
	@autoreleasepool {
		if (pauseDarwinInstallNotificationDelegate(errorOut) != 0) {
			return -1;
		}

		int authStatus = 0;
		if (pauseDarwinGetAuthorizationStatus(&authStatus, errorOut) != 0) {
			return -1;
		}
		if (!(authStatus == UNAuthorizationStatusAuthorized ||
		      authStatus == UNAuthorizationStatusProvisional)) {
			pauseNotifySetError(errorOut, @"Notification permission not granted");
			return -1;
		}

		NSString *title = @"Pause";
		if (titleC != NULL && titleC[0] != '\0') {
			NSString *tmp = [NSString stringWithUTF8String:titleC];
			if (tmp != nil && [tmp length] > 0) {
				title = tmp;
			}
		}
		NSString *body = @"Break started";
		if (bodyC != NULL && bodyC[0] != '\0') {
			NSString *tmp = [NSString stringWithUTF8String:bodyC];
			if (tmp != nil && [tmp length] > 0) {
				body = tmp;
			}
		}

		__block UNUserNotificationCenter *center = nil;
		pauseRunOnMainSync(^{
			center = [UNUserNotificationCenter currentNotificationCenter];
		});
		if (center == nil) {
			pauseNotifySetError(errorOut, @"UNUserNotificationCenter unavailable");
			return -1;
		}

		__block NSError *sendErr = nil;
		__block BOOL sent = NO;
		dispatch_semaphore_t sendSem = dispatch_semaphore_create(0);
		pauseRunOnMainSync(^{
			UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
			content.title = title;
			content.body = body;

			NSString *identifier = [[NSUUID UUID] UUIDString];
			UNTimeIntervalNotificationTrigger *trigger = [UNTimeIntervalNotificationTrigger triggerWithTimeInterval:0.1 repeats:NO];
			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:identifier content:content trigger:trigger];
			[center addNotificationRequest:request withCompletionHandler:^(NSError *addErr) {
				if (addErr != nil) {
					sendErr = addErr;
				} else {
					sent = YES;
				}
				dispatch_semaphore_signal(sendSem);
			}];
		});

		long sendWait = dispatch_semaphore_wait(sendSem, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
		if (sendWait != 0) {
			pauseNotifySetError(errorOut, @"Notification dispatch timed out");
			return -1;
		}
		if (!sent) {
			NSString *message = sendErr != nil ? [sendErr localizedDescription] : @"Notification dispatch failed";
			pauseNotifySetError(errorOut, message);
			return -1;
		}
		return 0;
	}
}
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"pause/internal/logx"
)

const (
	darwinNotificationStatusNotDetermined = 0
	darwinNotificationStatusDenied        = 1
	darwinNotificationStatusAuthorized    = 2
	darwinNotificationStatusProvisional   = 3
	darwinNotificationStatusEphemeral     = 4
)

type darwinNotificationAuthorizationResult struct {
	granted bool
	err     error
}

var (
	darwinNotificationAuthorizationMu      sync.Mutex
	darwinNotificationAuthorizationNextID  int
	darwinNotificationAuthorizationWaiters = map[int]chan darwinNotificationAuthorizationResult{}
)

func showDarwinUserNotification(title, body string) error {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	cBody := C.CString(body)
	defer C.free(unsafe.Pointer(cBody))

	var cErr *C.char
	rc := C.pauseDarwinShowUserNotification(cTitle, cBody, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
	}
	if rc != 0 {
		if cErr != nil {
			return fmt.Errorf("darwin user notification failed: %s", C.GoString(cErr))
		}
		return fmt.Errorf("darwin user notification failed")
	}
	return nil
}

func darwinNotificationAuthorizationStatus() (int, error) {
	var cErr *C.char
	var status C.int
	rc := C.pauseDarwinGetAuthorizationStatus(&status, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
	}
	if rc != 0 {
		if cErr != nil {
			err := fmt.Errorf("darwin notification status failed: %s", C.GoString(cErr))
			logx.Warnf("darwin.notification.status_request failed rc=%d err=%v", int(rc), err)
			return 0, err
		}
		err := fmt.Errorf("darwin notification status failed")
		logx.Warnf("darwin.notification.status_request failed rc=%d err=%v", int(rc), err)
		return 0, err
	}
	return int(status), nil
}

func darwinRequestNotificationAuthorization() (bool, error) {
	requestID, resultCh := registerDarwinNotificationAuthorizationWaiter()
	var cErr *C.char
	rc := C.pauseDarwinRequestAuthorizationAsync(C.int(requestID), &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
	}
	if rc != 0 {
		cleanupDarwinNotificationAuthorizationWaiter(requestID)
		if cErr != nil {
			err := fmt.Errorf("darwin notification authorization request failed: %s", C.GoString(cErr))
			logx.Warnf("darwin.notification.authorization_request failed rc=%d err=%v", int(rc), err)
			return false, err
		}
		err := fmt.Errorf("darwin notification authorization request failed")
		logx.Warnf("darwin.notification.authorization_request failed rc=%d err=%v", int(rc), err)
		return false, err
	}
	select {
	case result := <-resultCh:
		if result.err != nil {
			logx.Warnf("darwin.notification.authorization_request failed err=%v", result.err)
			return false, result.err
		}
		return result.granted, nil
	case <-time.After(180 * time.Second):
		cleanupDarwinNotificationAuthorizationWaiter(requestID)
		err := fmt.Errorf("darwin notification authorization request timed out")
		logx.Warnf("darwin.notification.authorization_request failed err=%v", err)
		return false, err
	}
}

func darwinOpenNotificationSettings(appID string) error {
	cAppID := C.CString(appID)
	defer C.free(unsafe.Pointer(cAppID))
	var cErr *C.char
	rc := C.pauseDarwinOpenNotificationSettings(cAppID, &cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
	}
	if rc != 0 {
		if cErr != nil {
			err := fmt.Errorf("darwin open notification settings failed: %s", C.GoString(cErr))
			logx.Warnf("darwin.notification.settings_open failed rc=%d err=%v", int(rc), err)
			return err
		}
		err := fmt.Errorf("darwin open notification settings failed")
		logx.Warnf("darwin.notification.settings_open failed rc=%d err=%v", int(rc), err)
		return err
	}
	return nil
}

func installDarwinNotificationClickDelegate() error {
	var cErr *C.char
	rc := C.pauseDarwinInstallNotificationDelegate(&cErr)
	if cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
	}
	if rc != 0 {
		if cErr != nil {
			err := fmt.Errorf("darwin install notification delegate failed: %s", C.GoString(cErr))
			logx.Warnf("darwin.notification.delegate_install failed rc=%d err=%v", int(rc), err)
			return err
		}
		err := fmt.Errorf("darwin install notification delegate failed")
		logx.Warnf("darwin.notification.delegate_install failed rc=%d err=%v", int(rc), err)
		return err
	}
	return nil
}

func registerDarwinNotificationAuthorizationWaiter() (int, chan darwinNotificationAuthorizationResult) {
	darwinNotificationAuthorizationMu.Lock()
	defer darwinNotificationAuthorizationMu.Unlock()

	requestID := darwinNotificationAuthorizationNextID
	darwinNotificationAuthorizationNextID++

	resultCh := make(chan darwinNotificationAuthorizationResult, 1)
	darwinNotificationAuthorizationWaiters[requestID] = resultCh
	return requestID, resultCh
}

func cleanupDarwinNotificationAuthorizationWaiter(requestID int) {
	darwinNotificationAuthorizationMu.Lock()
	resultCh, ok := darwinNotificationAuthorizationWaiters[requestID]
	if ok {
		delete(darwinNotificationAuthorizationWaiters, requestID)
	}
	darwinNotificationAuthorizationMu.Unlock()

	if ok {
		close(resultCh)
	}
}

func completeDarwinNotificationAuthorizationWaiter(requestID int, result darwinNotificationAuthorizationResult) {
	darwinNotificationAuthorizationMu.Lock()
	resultCh, ok := darwinNotificationAuthorizationWaiters[requestID]
	if ok {
		delete(darwinNotificationAuthorizationWaiters, requestID)
	}
	darwinNotificationAuthorizationMu.Unlock()

	if !ok {
		return
	}
	resultCh <- result
	close(resultCh)
}
