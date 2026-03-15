//go:build darwin && wails

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#include <stdlib.h>

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

static void PauseThemeRefreshOnMain(void);

static NSString *pauseThemeCached;
static NSObject *pauseThemeObserver;
static dispatch_queue_t pauseThemeStateQueue;

@interface PauseThemeKVOObserver : NSObject
@end

@implementation PauseThemeKVOObserver
- (void)observeValueForKeyPath:(NSString *)keyPath
                      ofObject:(id)object
                        change:(NSDictionary<NSKeyValueChangeKey,id> *)change
                       context:(void *)context {
    (void)keyPath;
    (void)object;
    (void)change;
    (void)context;
    PauseThemeRefreshOnMain();
}
@end

static void PauseThemeEnsureStateQueue(void) {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        pauseThemeStateQueue = dispatch_queue_create("pause.theme.state", DISPATCH_QUEUE_SERIAL);
    });
}

static void PauseThemeRunOnMain(void (^block)(void)) {
    if ([NSThread isMainThread]) {
        block();
        return;
    }
    dispatch_sync(dispatch_get_main_queue(), block);
}

static NSString *PauseThemeResolveOnMain(void) {
    if (@available(macOS 10.14, *)) {
        NSAppearance *appearance = [NSApp effectiveAppearance];
        if (appearance != nil) {
            NSArray<NSAppearanceName> *names = @[NSAppearanceNameAqua, NSAppearanceNameDarkAqua];
            NSAppearanceName best = [appearance bestMatchFromAppearancesWithNames:names];
            if ([best isEqualToString:NSAppearanceNameDarkAqua]) {
                return @"dark";
            }
            if ([best isEqualToString:NSAppearanceNameAqua]) {
                return @"light";
            }
        }
    }
    return nil;
}

static void PauseThemeSetCachedOnMain(NSString *theme) {
    PauseThemeEnsureStateQueue();
    NSString *next = (theme != nil) ? [theme copy] : nil;
    dispatch_sync(pauseThemeStateQueue, ^{
        if (pauseThemeCached != nil) {
            [pauseThemeCached release];
        }
        pauseThemeCached = next;
    });
}

static NSString *PauseThemeGetCachedCopy(void) {
    PauseThemeEnsureStateQueue();
    __block NSString *snapshot = nil;
    dispatch_sync(pauseThemeStateQueue, ^{
        if (pauseThemeCached != nil) {
            snapshot = [pauseThemeCached copy];
        }
    });
    return snapshot;
}

static void PauseThemeRefreshOnMain(void) {
    PauseThemeSetCachedOnMain(PauseThemeResolveOnMain());
}

static void PauseThemeEnsureObserverOnMain(void) {
    if (pauseThemeObserver != nil) {
        return;
    }
    PauseThemeRefreshOnMain();
    if (NSApp != nil) {
        pauseThemeObserver = [PauseThemeKVOObserver new];
        [NSApp addObserver:pauseThemeObserver
                forKeyPath:@"effectiveAppearance"
                   options:NSKeyValueObservingOptionNew
                   context:NULL];
    }
}

void PauseThemeProviderInit(void) {
    PauseThemeRunOnMain(^{
        PauseThemeEnsureObserverOnMain();
    });
}

void PauseThemeProviderDestroy(void) {
    PauseThemeRunOnMain(^{
        if (pauseThemeObserver != nil) {
            if (NSApp != nil) {
                @try {
                    [NSApp removeObserver:pauseThemeObserver forKeyPath:@"effectiveAppearance"];
                } @catch (NSException *exception) {
                    (void)exception;
                }
            }
            [pauseThemeObserver release];
            pauseThemeObserver = nil;
        }
        PauseThemeSetCachedOnMain(nil);
    });
}

const char *PausePreferredTheme(void) {
    @autoreleasepool {
        NSString *snapshot = PauseThemeGetCachedCopy();
        if (snapshot == nil) {
            PauseThemeRunOnMain(^{
                PauseThemeEnsureObserverOnMain();
            });
            snapshot = PauseThemeGetCachedCopy();
        }
        if (snapshot == nil) {
            return NULL;
        }
        const char *utf8 = [snapshot UTF8String];
        if (utf8 == NULL) {
            [snapshot release];
            return NULL;
        }
        char *out = strdup(utf8);
        [snapshot release];
        return out;
    }
}
*/
import "C"

import (
	"sync"
	"unsafe"
)

var preferredThemeProviderInitOnce sync.Once

func initPreferredThemeProvider() {
	preferredThemeProviderInitOnce.Do(func() {
		C.PauseThemeProviderInit()
	})
}

func shutdownPreferredThemeProvider() {
	C.PauseThemeProviderDestroy()
}

func detectPreferredTheme() string {
	initPreferredThemeProvider()
	ptr := C.PausePreferredTheme()
	if ptr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoString(ptr)
}
