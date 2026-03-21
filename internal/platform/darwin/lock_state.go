//go:build darwin

package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation -framework CoreGraphics

#import <Cocoa/Cocoa.h>
#import <CoreGraphics/CGSession.h>
#import <dispatch/dispatch.h>

static dispatch_queue_t pauseDarwinLockStateQueue;
static BOOL pauseDarwinSessionLocked;
static BOOL pauseDarwinLockObserverReady;
static id pauseDarwinSessionResignObserver;
static id pauseDarwinSessionBecomeObserver;
static id pauseDarwinScreenLockedObserver;
static id pauseDarwinScreenUnlockedObserver;

static void pauseDarwinEnsureLockStateQueue(void) {
	static dispatch_once_t onceToken;
	dispatch_once(&onceToken, ^{
		pauseDarwinLockStateQueue = dispatch_queue_create("pause.lock_state", DISPATCH_QUEUE_SERIAL);
	});
}

static void pauseDarwinSetSessionLocked(BOOL locked) {
	pauseDarwinEnsureLockStateQueue();
	dispatch_sync(pauseDarwinLockStateQueue, ^{
		pauseDarwinSessionLocked = locked;
	});
}

static BOOL pauseDarwinGetSessionLocked(void) {
	pauseDarwinEnsureLockStateQueue();
	__block BOOL locked = NO;
	dispatch_sync(pauseDarwinLockStateQueue, ^{
		locked = pauseDarwinSessionLocked;
	});
	return locked;
}

static void pauseDarwinRunOnMain(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_sync(dispatch_get_main_queue(), block);
}

static BOOL pauseDarwinProbeSessionLocked(void) {
	CFDictionaryRef session = CGSessionCopyCurrentDictionary();
	if (session == NULL) {
		return NO;
	}

	BOOL locked = NO;
	CFTypeRef lockedValue = CFDictionaryGetValue(session, CFSTR("CGSSessionScreenIsLocked"));
	if (lockedValue != NULL && CFGetTypeID(lockedValue) == CFBooleanGetTypeID()) {
		locked = CFBooleanGetValue((CFBooleanRef)lockedValue);
	}
	CFRelease(session);
	return locked;
}

static void pauseDarwinEnsureLockObserverOnMain(void) {
	if (pauseDarwinLockObserverReady) {
		return;
	}
	pauseDarwinLockObserverReady = YES;
	pauseDarwinSetSessionLocked(pauseDarwinProbeSessionLocked());

	NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
	if (workspace == nil) {
		return;
	}
	NSNotificationCenter *center = [workspace notificationCenter];
	if (center == nil) {
		return;
	}

	pauseDarwinSessionResignObserver = [center addObserverForName:NSWorkspaceSessionDidResignActiveNotification object:nil queue:nil usingBlock:^(NSNotification *note) {
		(void)note;
		pauseDarwinSetSessionLocked(YES);
	}];
	pauseDarwinSessionBecomeObserver = [center addObserverForName:NSWorkspaceSessionDidBecomeActiveNotification object:nil queue:nil usingBlock:^(NSNotification *note) {
		(void)note;
		pauseDarwinSetSessionLocked(NO);
	}];

	NSDistributedNotificationCenter *distributedCenter = [NSDistributedNotificationCenter defaultCenter];
	if (distributedCenter != nil) {
		pauseDarwinScreenLockedObserver = [distributedCenter addObserverForName:@"com.apple.screenIsLocked"
		                                                                 object:nil
		                                                                  queue:nil
		                                                             usingBlock:^(NSNotification *note) {
			(void)note;
			pauseDarwinSetSessionLocked(YES);
		}];
		pauseDarwinScreenUnlockedObserver = [distributedCenter addObserverForName:@"com.apple.screenIsUnlocked"
		                                                                   object:nil
		                                                                    queue:nil
		                                                               usingBlock:^(NSNotification *note) {
			(void)note;
			pauseDarwinSetSessionLocked(NO);
		}];
	}
}

static int pauseDarwinSessionIsLocked(void) {
	__block BOOL locked = NO;

	pauseDarwinRunOnMain(^{
		pauseDarwinEnsureLockObserverOnMain();
		locked = pauseDarwinGetSessionLocked();
	});

	return locked ? 1 : 0;
}
*/
import "C"

type darwinLockStateProvider struct{}

func (darwinLockStateProvider) IsScreenLocked() bool {
	return C.pauseDarwinSessionIsLocked() != 0
}
