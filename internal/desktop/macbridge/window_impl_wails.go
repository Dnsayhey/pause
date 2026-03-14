//go:build darwin && wails

package macbridge

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

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

static NSWindow *PauseResolveMainWindow(void) {
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
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        NSWindow *window = PauseResolveMainWindow();
        PauseDisableWindowMinimize(window);
        PauseDisableWindowZoom(window);
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
        PauseDisableWindowZoom(window);
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
        PauseDisableWindowZoom(window);
        PauseConfigureWindowTitle(window);

        if ([window isMiniaturized]) {
            [window deminiaturize:nil];
        }

        [NSApp activateIgnoringOtherApps:YES];
        [window makeKeyAndOrderFront:nil];
        [window orderFrontRegardless];
    });
}

void PauseHideMainWindow(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *window = PauseResolveMainWindow();
        if (window == nil) {
            return;
        }
        [window orderOut:nil];
    });
}
*/
import "C"
