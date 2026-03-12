//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

static NSWindow* PauseResolveMainWindow(void) {
    NSWindow *mainWindow = [NSApp mainWindow];
    if (mainWindow != nil) {
        return mainWindow;
    }

    NSArray<NSWindow *> *windows = [NSApp windows];
    for (NSWindow *window in windows) {
        if (window == nil) {
            continue;
        }
        return window;
    }
    return nil;
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
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        NSWindow *window = PauseResolveMainWindow();
        PauseDisableWindowMinimize(window);
        PauseConfigureWindowTitle(window);
    });
}

void PauseShowMainWindowNoActivate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            return;
        }

        PauseDisableWindowMinimize(window);
        PauseConfigureWindowTitle(window);

        if ([window isMiniaturized]) {
            [window deminiaturize:nil];
        }

        [window orderFrontRegardless];
    });
}

void PauseShowMainWindowActivate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            return;
        }

        PauseDisableWindowMinimize(window);
        PauseConfigureWindowTitle(window);

        if ([window isMiniaturized]) {
            [window deminiaturize:nil];
        }

        [NSApp activateIgnoringOtherApps:YES];
        [window makeKeyAndOrderFront:nil];
        [window orderFrontRegardless];
    });
}
*/
import "C"

import "context"

func configureDesktopWindowBehavior() {
	C.PauseConfigureDesktopWindowBehavior()
}

func showMainWindowFromStatusBar(_ context.Context) {
	C.PauseShowMainWindowActivate()
}

func showMainWindowForOverlay(_ context.Context) {
	C.PauseShowMainWindowNoActivate()
}
