//go:build darwin && cgo

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework UserNotifications

#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#import <dispatch/dispatch.h>
#import <errno.h>
#import <stdlib.h>
#import <string.h>

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

static int pauseDarwinShowUserNotification(const char *titleC, const char *bodyC, char **errorOut) {
	@autoreleasepool {
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

		if (@available(macOS 10.14, *)) {
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
			if (center == nil) {
				if (errorOut != NULL) {
					*errorOut = pauseNotifyCopyCString(@"UNUserNotificationCenter unavailable");
				}
				return -1;
			}

			__block UNAuthorizationStatus authStatus = UNAuthorizationStatusNotDetermined;
			__block BOOL settingsLoaded = NO;
			dispatch_semaphore_t settingsSem = dispatch_semaphore_create(0);
			dispatch_async(dispatch_get_main_queue(), ^{
				[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
					if (settings != nil) {
						authStatus = settings.authorizationStatus;
						settingsLoaded = YES;
					}
					dispatch_semaphore_signal(settingsSem);
				}];
			});
			(void)dispatch_semaphore_wait(settingsSem, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
			if (settingsLoaded && authStatus == UNAuthorizationStatusDenied) {
				if (errorOut != NULL) {
					*errorOut = pauseNotifyCopyCString(@"Notification permission denied");
				}
				return -1;
			}

			if (!settingsLoaded || authStatus == UNAuthorizationStatusNotDetermined) {
				__block NSError *authErr = nil;
				__block BOOL granted = NO;
				dispatch_semaphore_t authSem = dispatch_semaphore_create(0);
				dispatch_async(dispatch_get_main_queue(), ^{
					[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
					                      completionHandler:^(BOOL localGranted, NSError *localErr) {
						granted = localGranted;
						authErr = localErr;
						dispatch_semaphore_signal(authSem);
					}];
				});

				int authWait = dispatch_semaphore_wait(authSem, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
				if (authWait != 0) {
					// User may still be deciding on the permission prompt.
					return 1;
				}
				if (authErr != nil) {
					if (errorOut != NULL) {
						*errorOut = pauseNotifyCopyCString([authErr localizedDescription]);
					}
					return -1;
				}
				if (!granted) {
					if (errorOut != NULL) {
						*errorOut = pauseNotifyCopyCString(@"Notification permission denied");
					}
					return -1;
				}
			}

			__block NSError *sendErr = nil;
			__block BOOL sent = NO;
			dispatch_semaphore_t sendSem = dispatch_semaphore_create(0);
			dispatch_async(dispatch_get_main_queue(), ^{
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

			int sendWait = dispatch_semaphore_wait(sendSem, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
			if (sendWait != 0) {
				if (errorOut != NULL) {
					*errorOut = pauseNotifyCopyCString(@"Notification dispatch timed out");
				}
				return -1;
			}
			if (!sent) {
				if (errorOut != NULL) {
					NSString *msg = sendErr != nil ? [sendErr localizedDescription] : @"Notification dispatch failed";
					*errorOut = pauseNotifyCopyCString(msg);
				}
				return -1;
			}
			return 0;
		}

		if (errorOut != NULL) {
			*errorOut = pauseNotifyCopyCString(@"UserNotifications framework requires macOS 10.14+");
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
		if rc > 0 {
			return nil
		}
		if cErr != nil {
			return fmt.Errorf("darwin user notification failed: %s", C.GoString(cErr))
		}
		return fmt.Errorf("darwin user notification failed")
	}
	return nil
}
