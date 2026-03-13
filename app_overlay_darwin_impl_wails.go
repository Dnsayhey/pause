//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void overlaySkipCallbackGo(void);

static NSMutableArray<NSWindow *> *pauseOverlayWindows;
static NSMutableArray<NSTextField *> *pauseOverlayCountdownLabels;
static BOOL pauseOverlayVisible;
static BOOL pauseOverlayAllowSkip;
static NSString *pauseOverlaySkipButtonTitle;
static NSString *pauseOverlayCountdownText;
static NSString *pauseOverlayTheme;
static const NSTimeInterval pauseOverlayFadeDuration = 2.0;

@interface PauseOverlayHandler : NSObject
- (void)onSkipButtonClick:(id)sender;
@end

@implementation PauseOverlayHandler
- (void)onSkipButtonClick:(id)sender {
    (void)sender;
    overlaySkipCallbackGo();
}
@end

static PauseOverlayHandler *pauseOverlayHandler;

static void PauseOverlayRunOnMain(void (^block)(void)) {
    if ([NSThread isMainThread]) {
        block();
        return;
    }
    dispatch_sync(dispatch_get_main_queue(), block);
}

static void PauseOverlayEnsureHandler(void) {
    if (pauseOverlayHandler == nil) {
        pauseOverlayHandler = [PauseOverlayHandler new];
    }
}

static BOOL PauseOverlayThemeIsDark(NSString *theme) {
    if (theme == nil) {
        return YES;
    }
    return [[theme lowercaseString] isEqualToString:@"dark"];
}

static NSColor *PauseOverlayBackgroundColorForTheme(NSString *theme) {
    if (PauseOverlayThemeIsDark(theme)) {
        return [NSColor colorWithSRGBRed:0.0 green:0.0 blue:0.0 alpha:1.0];
    }
    return [NSColor colorWithSRGBRed:1.0 green:1.0 blue:1.0 alpha:1.0];
}

static NSColor *PauseOverlayCountdownColorForTheme(NSString *theme) {
    if (PauseOverlayThemeIsDark(theme)) {
        return [NSColor colorWithSRGBRed:0.90 green:0.91 blue:0.93 alpha:1.0];
    }
    return [NSColor colorWithSRGBRed:0.08 green:0.08 blue:0.08 alpha:1.0];
}

static NSColor *PauseOverlayButtonBackgroundColorForTheme(NSString *theme) {
    if (PauseOverlayThemeIsDark(theme)) {
        return [NSColor colorWithSRGBRed:0.13 green:0.14 blue:0.16 alpha:0.98];
    }
    return [NSColor colorWithSRGBRed:0.96 green:0.96 blue:0.96 alpha:0.98];
}

static NSColor *PauseOverlayButtonBorderColorForTheme(NSString *theme) {
    if (PauseOverlayThemeIsDark(theme)) {
        return [NSColor colorWithSRGBRed:0.36 green:0.37 blue:0.40 alpha:0.92];
    }
    return [NSColor colorWithSRGBRed:0.20 green:0.20 blue:0.20 alpha:0.22];
}

static NSColor *PauseOverlayButtonTextColorForTheme(NSString *theme) {
    if (PauseOverlayThemeIsDark(theme)) {
        return [NSColor colorWithSRGBRed:0.92 green:0.93 blue:0.95 alpha:1.0];
    }
    return [NSColor colorWithSRGBRed:0.08 green:0.08 blue:0.08 alpha:1.0];
}

static NSButton *PauseOverlayBuildSkipButton(NSString *title, NSString *theme) {
    NSButton *button = [NSButton buttonWithTitle:title target:pauseOverlayHandler action:@selector(onSkipButtonClick:)];
    [button setBezelStyle:NSBezelStyleRegularSquare];
    [button setBordered:NO];
    [button setControlSize:NSControlSizeRegular];
    [button setFont:[NSFont systemFontOfSize:14 weight:NSFontWeightSemibold]];
    [button setTranslatesAutoresizingMaskIntoConstraints:NO];
    [button setWantsLayer:YES];
    [button.layer setCornerRadius:10.0];
    [button.layer setBorderWidth:1.0];
    [button.layer setMasksToBounds:YES];
    [button.layer setBackgroundColor:[PauseOverlayButtonBackgroundColorForTheme(theme) CGColor]];
    [button.layer setBorderColor:[PauseOverlayButtonBorderColorForTheme(theme) CGColor]];
    NSDictionary *attrs = @{
        NSForegroundColorAttributeName: PauseOverlayButtonTextColorForTheme(theme),
        NSFontAttributeName: [NSFont systemFontOfSize:14 weight:NSFontWeightSemibold]
    };
    NSAttributedString *styledTitle = [[[NSAttributedString alloc] initWithString:title attributes:attrs] autorelease];
    [button setAttributedTitle:styledTitle];
    return button;
}

static void PauseOverlayUpdateCountdownTextOnMain(NSString *countdownText) {
    if (countdownText == nil) {
        countdownText = @"";
    }

    BOOL sameText = (pauseOverlayCountdownText != nil && [pauseOverlayCountdownText isEqualToString:countdownText]);
    if (!sameText) {
        if (pauseOverlayCountdownText != nil) {
            [pauseOverlayCountdownText release];
        }
        pauseOverlayCountdownText = [countdownText copy];
    }

    if (pauseOverlayCountdownLabels == nil) {
        return;
    }
    for (NSTextField *label in pauseOverlayCountdownLabels) {
        [label setStringValue:pauseOverlayCountdownText];
    }
}

static NSWindow *PauseOverlayBuildWindowForScreen(NSScreen *screen, BOOL allowSkip, NSString *skipButtonTitle, NSString *countdownText, NSString *theme, NSTextField **outCountdownLabel) {
    NSRect screenFrame = [screen frame];
    NSRect initialRect = NSMakeRect(0, 0, screenFrame.size.width, screenFrame.size.height);
    NSWindow *window = [[NSWindow alloc] initWithContentRect:initialRect styleMask:NSWindowStyleMaskBorderless backing:NSBackingStoreBuffered defer:NO];
    [window setFrame:screenFrame display:NO];
    [window setOpaque:YES];
    [window setBackgroundColor:PauseOverlayBackgroundColorForTheme(theme)];
    [window setHasShadow:NO];
    [window setIgnoresMouseEvents:NO];
    [window setLevel:NSScreenSaverWindowLevel];
    [window setCollectionBehavior:(NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorStationary | NSWindowCollectionBehaviorFullScreenAuxiliary)];
    [window setMovable:NO];
    [window setReleasedWhenClosed:NO];
    [window setAlphaValue:0.0];

    NSView *contentView = [window contentView];
    NSTextField *countdownLabel = nil;
    if (contentView != nil) {
        countdownLabel = [NSTextField labelWithString:(countdownText != nil ? countdownText : @"")];
        [countdownLabel setFont:[NSFont monospacedDigitSystemFontOfSize:30 weight:NSFontWeightMedium]];
        [countdownLabel setTextColor:PauseOverlayCountdownColorForTheme(theme)];
        [countdownLabel setAlignment:NSTextAlignmentCenter];
        [countdownLabel setLineBreakMode:NSLineBreakByTruncatingTail];
        [countdownLabel setTranslatesAutoresizingMaskIntoConstraints:NO];
        [contentView addSubview:countdownLabel];
    }

    if (allowSkip) {
        if (contentView != nil) {
            NSButton *skipButton = PauseOverlayBuildSkipButton(skipButtonTitle, theme);
            [contentView addSubview:skipButton];

            [NSLayoutConstraint activateConstraints:@[
                [skipButton.centerXAnchor constraintEqualToAnchor:contentView.centerXAnchor],
                [skipButton.centerYAnchor constraintEqualToAnchor:contentView.centerYAnchor],
                [skipButton.heightAnchor constraintEqualToConstant:36],
                [skipButton.widthAnchor constraintGreaterThanOrEqualToConstant:170],
                [countdownLabel.centerXAnchor constraintEqualToAnchor:contentView.centerXAnchor],
                [countdownLabel.bottomAnchor constraintEqualToAnchor:skipButton.topAnchor constant:-16],
            ]];
        }
    } else if (contentView != nil && countdownLabel != nil) {
        [NSLayoutConstraint activateConstraints:@[
            [countdownLabel.centerXAnchor constraintEqualToAnchor:contentView.centerXAnchor],
            [countdownLabel.centerYAnchor constraintEqualToAnchor:contentView.centerYAnchor],
        ]];
    }

    if (outCountdownLabel != NULL) {
        *outCountdownLabel = countdownLabel;
    }
    return window;
}

static void PauseOverlayHideOnMain(void) {
    if (pauseOverlaySkipButtonTitle != nil) {
        [pauseOverlaySkipButtonTitle release];
        pauseOverlaySkipButtonTitle = nil;
    }
    if (pauseOverlayCountdownText != nil) {
        [pauseOverlayCountdownText release];
        pauseOverlayCountdownText = nil;
    }
    if (pauseOverlayTheme != nil) {
        [pauseOverlayTheme release];
        pauseOverlayTheme = nil;
    }
    if (pauseOverlayCountdownLabels != nil) {
        [pauseOverlayCountdownLabels release];
        pauseOverlayCountdownLabels = nil;
    }

    if (pauseOverlayWindows == nil || [pauseOverlayWindows count] == 0) {
        pauseOverlayVisible = NO;
        pauseOverlayAllowSkip = NO;
        return;
    }

    NSArray<NSWindow *> *windows = [pauseOverlayWindows copy];
    [pauseOverlayWindows release];
    pauseOverlayWindows = nil;
    pauseOverlayVisible = NO;
    pauseOverlayAllowSkip = NO;

    for (NSWindow *window in windows) {
        [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
            [context setDuration:pauseOverlayFadeDuration];
            [[window animator] setAlphaValue:0.0];
        } completionHandler:^{
            [window orderOut:nil];
        }];
    }

    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)((pauseOverlayFadeDuration + 0.06) * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        [windows release];
    });
}

void PauseBreakOverlayInit(void) {
    PauseOverlayRunOnMain(^{
        PauseOverlayEnsureHandler();
    });
}

int PauseBreakOverlayShow(int allowSkip, const char *skipButtonTitle, const char *countdownText, const char *theme) {
    NSString *skipTitle = skipButtonTitle ? [NSString stringWithUTF8String:skipButtonTitle] : @"Emergency Skip";
    NSString *countdown = countdownText ? [NSString stringWithUTF8String:countdownText] : @"";
    NSString *overlayTheme = theme ? [NSString stringWithUTF8String:theme] : @"dark";
    BOOL shouldAllowSkip = allowSkip != 0;
    __block BOOL didShow = NO;

    PauseOverlayRunOnMain(^{
        NSArray<NSScreen *> *screens = [NSScreen screens];
        BOOL sameScreenCount = pauseOverlayVisible && pauseOverlayWindows != nil && screens != nil && [pauseOverlayWindows count] == [screens count];
        BOOL sameSkipSetting = pauseOverlayVisible && (pauseOverlayAllowSkip == shouldAllowSkip);
        BOOL sameTitle = (pauseOverlaySkipButtonTitle == nil && skipTitle == nil) || [pauseOverlaySkipButtonTitle isEqualToString:skipTitle];
        BOOL sameTheme = (pauseOverlayTheme == nil && overlayTheme == nil) || [pauseOverlayTheme isEqualToString:overlayTheme];
        if (sameScreenCount && sameSkipSetting && sameTitle && sameTheme) {
            PauseOverlayUpdateCountdownTextOnMain(countdown);
            didShow = pauseOverlayVisible;
            return;
        }

        PauseOverlayHideOnMain();
        PauseOverlayEnsureHandler();

        if (screens == nil || [screens count] == 0) {
            didShow = NO;
            return;
        }

        pauseOverlayWindows = [[NSMutableArray alloc] init];
        pauseOverlayCountdownLabels = [[NSMutableArray alloc] init];
        pauseOverlayAllowSkip = shouldAllowSkip;
        pauseOverlaySkipButtonTitle = [skipTitle copy];
        pauseOverlayCountdownText = [countdown copy];
        pauseOverlayTheme = [overlayTheme copy];

        for (NSUInteger i = 0; i < [screens count]; i++) {
            NSScreen *screen = [screens objectAtIndex:i];
            NSTextField *countdownLabel = nil;
            NSWindow *window = PauseOverlayBuildWindowForScreen(screen, shouldAllowSkip, skipTitle, countdown, overlayTheme, &countdownLabel);
            if (window == nil) {
                continue;
            }
            [pauseOverlayWindows addObject:window];
            if (countdownLabel != nil) {
                [pauseOverlayCountdownLabels addObject:countdownLabel];
            }
            [window orderFrontRegardless];
            [window release];
        }

        for (NSWindow *window in pauseOverlayWindows) {
            [NSAnimationContext runAnimationGroup:^(NSAnimationContext *context) {
                [context setDuration:pauseOverlayFadeDuration];
                [[window animator] setAlphaValue:1.0];
            } completionHandler:nil];
        }

        pauseOverlayVisible = ([pauseOverlayWindows count] > 0);
        didShow = pauseOverlayVisible;
    });

    return didShow ? 1 : 0;
}

void PauseBreakOverlayHide(void) {
    PauseOverlayRunOnMain(^{
        PauseOverlayHideOnMain();
    });
}

void PauseBreakOverlayDestroy(void) {
    PauseOverlayRunOnMain(^{
        PauseOverlayHideOnMain();
        if (pauseOverlayHandler != nil) {
            [pauseOverlayHandler release];
            pauseOverlayHandler = nil;
        }
    });
}
*/
import "C"
