//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void windowDebugEventGo(int eventID, int visible, int key, int mainWindow, int popover);

enum {
    PauseWindowDebugEventConfigure = 1,
    PauseWindowDebugEventShowActivate = 2,
    PauseWindowDebugEventShowNoActivate = 3,
    PauseWindowDebugEventHide = 4,
    PauseWindowDebugEventDidBecomeKey = 5,
    PauseWindowDebugEventDidBecomeMain = 6,
    PauseWindowDebugEventDidDeminiaturize = 7,
    PauseWindowDebugEventDidMiniaturize = 8,
    PauseWindowDebugEventDidResignKey = 9,
    PauseWindowDebugEventDidResignMain = 10,
    PauseWindowDebugEventDidExpose = 11,
    PauseWindowDebugEventDidChangeOcclusionState = 12,
    PauseWindowDebugEventAppDidBecomeActive = 13,
    PauseWindowDebugEventAppDidResignActive = 14,
    PauseWindowDebugEventSnapshotChanged = 15
};

static BOOL PauseWindowIsPopoverWindow(NSWindow *window) {
    if (window == nil) {
        return NO;
    }
    NSString *className = NSStringFromClass([window class]);
    if (className == nil) {
        return NO;
    }
    NSRange r = [className rangeOfString:@"popover" options:NSCaseInsensitiveSearch];
    return r.location != NSNotFound;
}

static NSWindow* PauseResolveMainWindow(void) {
    NSWindow *mainWindow = [NSApp mainWindow];
    if (mainWindow != nil && !PauseWindowIsPopoverWindow(mainWindow)) {
        return mainWindow;
    }

    NSArray<NSWindow *> *windows = [NSApp windows];
    for (NSWindow *window in windows) {
        if (window == nil || PauseWindowIsPopoverWindow(window)) {
            continue;
        }
        return window;
    }
    return nil;
}

static NSMutableArray *pauseWindowDebugObserverTokens;
static dispatch_source_t pauseWindowDebugSnapshotSource;
static int pauseWindowDebugHasSnapshot;
static int pauseWindowDebugLastVisible;
static int pauseWindowDebugLastKey;
static int pauseWindowDebugLastMain;

static void PauseEmitWindowDebugEvent(int eventID, NSWindow *window) {
    NSWindow *target = window;
    if (target == nil) {
        target = PauseResolveMainWindow();
    }

    int visible = 0;
    int key = 0;
    int mainWindow = 0;
    int popover = 0;
    if (target != nil) {
        visible = [target isVisible] ? 1 : 0;
        key = [target isKeyWindow] ? 1 : 0;
        mainWindow = [target isMainWindow] ? 1 : 0;
        popover = PauseWindowIsPopoverWindow(target) ? 1 : 0;
    }

    windowDebugEventGo(eventID, visible, key, mainWindow, popover);
}

static void PauseEmitWindowSnapshotIfChanged(void) {
    NSWindow *window = PauseResolveMainWindow();

    int visible = 0;
    int key = 0;
    int mainWindow = 0;
    if (window != nil) {
        visible = [window isVisible] ? 1 : 0;
        key = [window isKeyWindow] ? 1 : 0;
        mainWindow = [window isMainWindow] ? 1 : 0;
    }

    if (!pauseWindowDebugHasSnapshot ||
        pauseWindowDebugLastVisible != visible ||
        pauseWindowDebugLastKey != key ||
        pauseWindowDebugLastMain != mainWindow) {
        pauseWindowDebugHasSnapshot = 1;
        pauseWindowDebugLastVisible = visible;
        pauseWindowDebugLastKey = key;
        pauseWindowDebugLastMain = mainWindow;
        windowDebugEventGo(PauseWindowDebugEventSnapshotChanged, visible, key, mainWindow, 0);
    }
}

static void PauseInstallWindowDebugObservers(void) {
    if (pauseWindowDebugObserverTokens != nil) {
        return;
    }

    pauseWindowDebugObserverTokens = [[NSMutableArray alloc] init];
    NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
    NSOperationQueue *q = [NSOperationQueue mainQueue];

    id tokenKey = [nc addObserverForName:NSWindowDidBecomeKeyNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidBecomeKey, (NSWindow *)note.object);
    }];
    if (tokenKey != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenKey];
    }

    id tokenMain = [nc addObserverForName:NSWindowDidBecomeMainNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidBecomeMain, (NSWindow *)note.object);
    }];
    if (tokenMain != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenMain];
    }

    id tokenDemini = [nc addObserverForName:NSWindowDidDeminiaturizeNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidDeminiaturize, (NSWindow *)note.object);
    }];
    if (tokenDemini != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenDemini];
    }

    id tokenMini = [nc addObserverForName:NSWindowDidMiniaturizeNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidMiniaturize, (NSWindow *)note.object);
    }];
    if (tokenMini != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenMini];
    }

    id tokenResignKey = [nc addObserverForName:NSWindowDidResignKeyNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidResignKey, (NSWindow *)note.object);
    }];
    if (tokenResignKey != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenResignKey];
    }

    id tokenResignMain = [nc addObserverForName:NSWindowDidResignMainNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidResignMain, (NSWindow *)note.object);
    }];
    if (tokenResignMain != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenResignMain];
    }

    id tokenExpose = [nc addObserverForName:NSWindowDidExposeNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidExpose, (NSWindow *)note.object);
    }];
    if (tokenExpose != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenExpose];
    }

    id tokenOcclusion = [nc addObserverForName:NSWindowDidChangeOcclusionStateNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        PauseEmitWindowDebugEvent(PauseWindowDebugEventDidChangeOcclusionState, (NSWindow *)note.object);
    }];
    if (tokenOcclusion != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenOcclusion];
    }

    id tokenAppActive = [nc addObserverForName:NSApplicationDidBecomeActiveNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        (void)note;
        PauseEmitWindowDebugEvent(PauseWindowDebugEventAppDidBecomeActive, nil);
    }];
    if (tokenAppActive != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenAppActive];
    }

    id tokenAppResign = [nc addObserverForName:NSApplicationDidResignActiveNotification object:nil queue:q usingBlock:^(NSNotification *note) {
        (void)note;
        PauseEmitWindowDebugEvent(PauseWindowDebugEventAppDidResignActive, nil);
    }];
    if (tokenAppResign != nil) {
        [pauseWindowDebugObserverTokens addObject:tokenAppResign];
    }

    PauseEmitWindowSnapshotIfChanged();
    if (pauseWindowDebugSnapshotSource == nil) {
        pauseWindowDebugSnapshotSource = dispatch_source_create(DISPATCH_SOURCE_TYPE_TIMER, 0, 0, dispatch_get_main_queue());
        if (pauseWindowDebugSnapshotSource != nil) {
            dispatch_source_set_timer(
                pauseWindowDebugSnapshotSource,
                dispatch_time(DISPATCH_TIME_NOW, 500 * NSEC_PER_MSEC),
                500 * NSEC_PER_MSEC,
                50 * NSEC_PER_MSEC
            );
            dispatch_source_set_event_handler(pauseWindowDebugSnapshotSource, ^{
                PauseEmitWindowSnapshotIfChanged();
            });
            dispatch_resume(pauseWindowDebugSnapshotSource);
        }
    }
}

static void PauseDisableWindowMinimize(NSWindow *window) {
    if (window == nil) {
        return;
    }

    NSUInteger styleMask = [window styleMask];
    styleMask &= ~NSWindowStyleMaskMiniaturizable;
    [window setStyleMask:styleMask];

    NSButton *minimizeButton = [window standardWindowButton:NSWindowMiniaturizeButton];
    if (minimizeButton != nil) {
        [minimizeButton setEnabled:NO];
        [minimizeButton setHidden:NO];
    }
}

static void PauseDisableWindowZoom(NSWindow *window) {
    if (window == nil) {
        return;
    }

    NSButton *zoomButton = [window standardWindowButton:NSWindowZoomButton];
    if (zoomButton != nil) {
        [zoomButton setEnabled:NO];
        [zoomButton setHidden:NO];
    }
}

static void PauseConfigureWindowTitle(NSWindow *window) {
    if (window == nil) {
        return;
    }
    [window setTitle:@""];
    if ([window respondsToSelector:@selector(setTitleVisibility:)]) {
        [window setTitleVisibility:NSWindowTitleHidden];
    }
    if ([window respondsToSelector:@selector(setTitlebarAppearsTransparent:)]) {
        [window setTitlebarAppearsTransparent:YES];
    }
}

void PauseConfigureDesktopWindowBehavior(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PauseInstallWindowDebugObservers();
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        NSWindow *window = PauseResolveMainWindow();
        PauseDisableWindowMinimize(window);
        PauseDisableWindowZoom(window);
        PauseConfigureWindowTitle(window);
        PauseEmitWindowDebugEvent(PauseWindowDebugEventConfigure, window);
    });
}

void PauseShowMainWindowNoActivate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PauseInstallWindowDebugObservers();
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            PauseEmitWindowDebugEvent(PauseWindowDebugEventShowNoActivate, nil);
            return;
        }

        PauseDisableWindowMinimize(window);
        PauseDisableWindowZoom(window);
        PauseConfigureWindowTitle(window);

        if ([window isMiniaturized]) {
            [window deminiaturize:nil];
        }

        [window orderFrontRegardless];
        PauseEmitWindowDebugEvent(PauseWindowDebugEventShowNoActivate, window);
    });
}

void PauseShowMainWindowActivate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PauseInstallWindowDebugObservers();
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            PauseEmitWindowDebugEvent(PauseWindowDebugEventShowActivate, nil);
            return;
        }

        PauseDisableWindowMinimize(window);
        PauseDisableWindowZoom(window);
        PauseConfigureWindowTitle(window);

        if ([window isMiniaturized]) {
            [window deminiaturize:nil];
        }

        [NSApp activateIgnoringOtherApps:YES];
        [window makeKeyAndOrderFront:nil];
        [window orderFrontRegardless];
        PauseEmitWindowDebugEvent(PauseWindowDebugEventShowActivate, window);
    });
}

void PauseHideMainWindow(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        PauseInstallWindowDebugObservers();
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            PauseEmitWindowDebugEvent(PauseWindowDebugEventHide, nil);
            return;
        }
        [window orderOut:nil];
        PauseEmitWindowDebugEvent(PauseWindowDebugEventHide, window);
    });
}
*/
import "C"

import (
	"context"

	"pause/internal/diag"
)

func configureDesktopWindowBehavior() {
	diag.Logf("window.configure darwin")
	C.PauseConfigureDesktopWindowBehavior()
}

func showMainWindowFromStatusBar(_ context.Context) {
	diag.Logf("window.show activate source=status_bar_open")
	C.PauseShowMainWindowActivate()
}

func showMainWindowForOverlay(_ context.Context) {
	diag.Logf("window.show no_activate source=overlay_fallback")
	C.PauseShowMainWindowNoActivate()
}

func hideMainWindowForOverlay(_ context.Context) {
	diag.Logf("window.hide source=overlay_native")
	C.PauseHideMainWindow()
}
