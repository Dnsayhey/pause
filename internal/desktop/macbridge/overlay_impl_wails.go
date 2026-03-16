//go:build darwin && wails

package macbridge

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void overlaySkipCallbackGo(void);

static NSMutableArray<NSWindow *> *pauseOverlayWindows;
static NSMutableArray<NSTextField *> *pauseOverlayCountdownLabels;
static NSMutableArray<NSButton *> *pauseOverlaySkipButtons;
static BOOL pauseOverlayVisible;
static BOOL pauseOverlayAllowSkip;
static BOOL pauseOverlayEmergencySkipUnlocked;
static id pauseOverlayKeyMonitor;
static NSString *pauseOverlaySkipButtonTitle;
static NSString *pauseOverlayCountdownText;
static NSString *pauseOverlayTheme;
static const NSTimeInterval pauseOverlayFadeDuration = 1.0;
static const NSTimeInterval pauseOverlayCmdQDoublePressWindow = 1.0;
static NSTimeInterval pauseOverlayLastCmdQPress;
static void PauseOverlayHideOnMain(void);
static void PauseOverlayUpdateSkipButtonStyleOnMain(NSButton *button);

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

@interface PauseOverlaySkipButton : NSButton {
@private
    NSTrackingArea *_pauseOverlayTrackingArea;
    NSString *_pauseOverlayTheme;
    NSString *_pauseOverlayTitle;
    BOOL _pauseOverlayHovered;
}
@property(nonatomic, assign) BOOL pauseOverlayHovered;
@property(nonatomic, copy) NSString *pauseOverlayTheme;
@property(nonatomic, copy) NSString *pauseOverlayTitle;
- (void)pauseOverlayResetHighlight;
@end

@implementation PauseOverlaySkipButton
@synthesize pauseOverlayHovered = _pauseOverlayHovered;
@synthesize pauseOverlayTheme = _pauseOverlayTheme;
@synthesize pauseOverlayTitle = _pauseOverlayTitle;

- (void)dealloc {
    [NSObject cancelPreviousPerformRequestsWithTarget:self selector:@selector(pauseOverlayResetHighlight) object:nil];
    if (_pauseOverlayTrackingArea != nil) {
        [self removeTrackingArea:_pauseOverlayTrackingArea];
        [_pauseOverlayTrackingArea release];
        _pauseOverlayTrackingArea = nil;
    }
    [_pauseOverlayTheme release];
    _pauseOverlayTheme = nil;
    [_pauseOverlayTitle release];
    _pauseOverlayTitle = nil;
    [super dealloc];
}

- (void)updateTrackingAreas {
    if (_pauseOverlayTrackingArea != nil) {
        [self removeTrackingArea:_pauseOverlayTrackingArea];
        [_pauseOverlayTrackingArea release];
        _pauseOverlayTrackingArea = nil;
    }
    NSTrackingAreaOptions options = NSTrackingMouseEnteredAndExited | NSTrackingActiveAlways | NSTrackingInVisibleRect;
    _pauseOverlayTrackingArea = [[NSTrackingArea alloc] initWithRect:NSZeroRect options:options owner:self userInfo:nil];
    [self addTrackingArea:_pauseOverlayTrackingArea];
    [super updateTrackingAreas];
}

- (void)resetCursorRects {
    [super resetCursorRects];
    [self addCursorRect:[self bounds] cursor:[NSCursor pointingHandCursor]];
}

- (void)mouseDown:(NSEvent *)event {
    (void)event;
    if (![self isEnabled]) {
        return;
    }
    [self setHighlighted:YES];
    if ([self target] != nil && [self action] != NULL) {
        [NSApp sendAction:[self action] to:[self target] from:self];
    }
    [NSObject cancelPreviousPerformRequestsWithTarget:self selector:@selector(pauseOverlayResetHighlight) object:nil];
    [self performSelector:@selector(pauseOverlayResetHighlight) withObject:nil afterDelay:0.12];
}

- (void)mouseEntered:(NSEvent *)event {
    [super mouseEntered:event];
    self.pauseOverlayHovered = YES;
    PauseOverlayUpdateSkipButtonStyleOnMain(self);
}

- (void)mouseExited:(NSEvent *)event {
    [super mouseExited:event];
    self.pauseOverlayHovered = NO;
    PauseOverlayUpdateSkipButtonStyleOnMain(self);
}

- (void)setHighlighted:(BOOL)flag {
    BOOL changed = ([self isHighlighted] != flag);
    [super setHighlighted:flag];
    if (changed) {
        PauseOverlayUpdateSkipButtonStyleOnMain(self);
    }
}

- (void)pauseOverlayResetHighlight {
    [self setHighlighted:NO];
}
@end

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

static void PauseOverlayRemoveKeyMonitor(void) {
	if (pauseOverlayKeyMonitor != nil) {
		[NSEvent removeMonitor:pauseOverlayKeyMonitor];
		pauseOverlayKeyMonitor = nil;
	}
	pauseOverlayLastCmdQPress = 0;
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

static NSColor *PauseOverlayButtonBackgroundColorForTheme(NSString *theme, BOOL hovered, BOOL pressed) {
    if (PauseOverlayThemeIsDark(theme)) {
        if (pressed) {
            return [NSColor colorWithSRGBRed:0.09 green:0.10 blue:0.12 alpha:1.0];
        }
        if (hovered) {
            return [NSColor colorWithSRGBRed:0.17 green:0.18 blue:0.21 alpha:0.98];
        }
        return [NSColor colorWithSRGBRed:0.13 green:0.14 blue:0.16 alpha:0.98];
    }
    if (pressed) {
        return [NSColor colorWithSRGBRed:0.84 green:0.85 blue:0.88 alpha:1.0];
    }
    if (hovered) {
        return [NSColor colorWithSRGBRed:0.92 green:0.92 blue:0.92 alpha:0.98];
    }
    return [NSColor colorWithSRGBRed:0.96 green:0.96 blue:0.96 alpha:0.98];
}

static NSColor *PauseOverlayButtonBorderColorForTheme(NSString *theme, BOOL hovered, BOOL pressed) {
    if (PauseOverlayThemeIsDark(theme)) {
        if (pressed) {
            return [NSColor colorWithSRGBRed:0.58 green:0.60 blue:0.64 alpha:0.98];
        }
        if (hovered) {
            return [NSColor colorWithSRGBRed:0.46 green:0.47 blue:0.50 alpha:0.94];
        }
        return [NSColor colorWithSRGBRed:0.36 green:0.37 blue:0.40 alpha:0.92];
    }
    if (pressed) {
        return [NSColor colorWithSRGBRed:0.20 green:0.22 blue:0.26 alpha:0.42];
    }
    if (hovered) {
        return [NSColor colorWithSRGBRed:0.20 green:0.20 blue:0.20 alpha:0.30];
    }
    return [NSColor colorWithSRGBRed:0.20 green:0.20 blue:0.20 alpha:0.22];
}

static NSColor *PauseOverlayButtonTextColorForTheme(NSString *theme, BOOL hovered, BOOL pressed) {
    if (PauseOverlayThemeIsDark(theme)) {
        if (pressed) {
            return [NSColor colorWithSRGBRed:0.98 green:0.98 blue:0.99 alpha:1.0];
        }
        if (hovered) {
            return [NSColor colorWithSRGBRed:0.95 green:0.96 blue:0.98 alpha:1.0];
        }
        return [NSColor colorWithSRGBRed:0.92 green:0.93 blue:0.95 alpha:1.0];
    }
    if (pressed) {
        return [NSColor colorWithSRGBRed:0.05 green:0.05 blue:0.06 alpha:1.0];
    }
    if (hovered) {
        return [NSColor colorWithSRGBRed:0.06 green:0.06 blue:0.06 alpha:1.0];
    }
    return [NSColor colorWithSRGBRed:0.08 green:0.08 blue:0.08 alpha:1.0];
}

static NSButton *PauseOverlayBuildSkipButton(NSString *title, NSString *theme) {
    NSString *resolvedTitle = (title != nil ? title : @"Emergency Skip");
    PauseOverlaySkipButton *button = [[[PauseOverlaySkipButton alloc] initWithFrame:NSMakeRect(0, 0, 170, 36)] autorelease];
    [button setPauseOverlayTitle:resolvedTitle];
    [button setPauseOverlayTheme:(theme != nil ? theme : @"dark")];
    [button setTitle:resolvedTitle];
    [button setTarget:pauseOverlayHandler];
    [button setAction:@selector(onSkipButtonClick:)];
    [button setButtonType:NSButtonTypeMomentaryChange];
    [button setBezelStyle:NSBezelStyleRegularSquare];
    [button setBordered:NO];
    [button setControlSize:NSControlSizeRegular];
    [button setFont:[NSFont systemFontOfSize:14 weight:NSFontWeightSemibold]];
    [button setTranslatesAutoresizingMaskIntoConstraints:NO];
    [button setWantsLayer:YES];
    [button.layer setCornerRadius:10.0];
    [button.layer setBorderWidth:1.0];
    [button.layer setMasksToBounds:YES];
    [button.layer setShadowOffset:CGSizeMake(0.0, 2.0)];
    [button.layer setShadowRadius:6.0];
    [button.layer setShadowOpacity:0.18];
    [button.layer setOpacity:1.0];
    PauseOverlayUpdateSkipButtonStyleOnMain(button);
    return button;
}

static void PauseOverlayUpdateSkipButtonStyleOnMain(NSButton *button) {
    if (button == nil || ![button isKindOfClass:[PauseOverlaySkipButton class]]) {
        return;
    }
    PauseOverlaySkipButton *skipButton = (PauseOverlaySkipButton *)button;
    NSString *title = skipButton.pauseOverlayTitle;
    if (title == nil || [title length] == 0) {
        title = @"Skip";
    }
    NSString *theme = skipButton.pauseOverlayTheme;
    if (theme == nil) {
        theme = @"dark";
    }

    BOOL hovered = skipButton.pauseOverlayHovered;
    BOOL pressed = skipButton.isHighlighted;
    NSDictionary *attrs = @{
        NSForegroundColorAttributeName: PauseOverlayButtonTextColorForTheme(theme, hovered, pressed),
        NSFontAttributeName: [NSFont systemFontOfSize:14 weight:NSFontWeightSemibold]
    };
    [skipButton setTitle:title];
    NSAttributedString *styledTitle = [[[NSAttributedString alloc] initWithString:title attributes:attrs] autorelease];
    [skipButton setAttributedTitle:styledTitle];
    [skipButton.layer setBackgroundColor:[PauseOverlayButtonBackgroundColorForTheme(theme, hovered, pressed) CGColor]];
    [skipButton.layer setBorderColor:[PauseOverlayButtonBorderColorForTheme(theme, hovered, pressed) CGColor]];
    [skipButton.layer setBorderWidth:(pressed ? 1.35 : 1.0)];
    [skipButton.layer setOpacity:(pressed ? 0.93 : 1.0)];
    [skipButton.layer setShadowOffset:(pressed ? CGSizeMake(0.0, 1.0) : CGSizeMake(0.0, 2.0))];
    [skipButton.layer setShadowOpacity:(pressed ? 0.08 : (hovered ? 0.26 : 0.18))];
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

static NSWindow *PauseOverlayBuildWindowForScreen(
    NSScreen *screen,
    BOOL allowSkip,
    NSString *skipButtonTitle,
    NSString *countdownText,
    NSString *theme,
    NSTextField **outCountdownLabel,
    NSButton **outSkipButton
) {
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
    NSButton *skipButton = nil;
    if (contentView != nil) {
        countdownLabel = [NSTextField labelWithString:(countdownText != nil ? countdownText : @"")];
        [countdownLabel setFont:[NSFont monospacedDigitSystemFontOfSize:30 weight:NSFontWeightMedium]];
        [countdownLabel setTextColor:PauseOverlayCountdownColorForTheme(theme)];
        [countdownLabel setAlignment:NSTextAlignmentCenter];
        [countdownLabel setLineBreakMode:NSLineBreakByTruncatingTail];
        [countdownLabel setTranslatesAutoresizingMaskIntoConstraints:NO];
        skipButton = PauseOverlayBuildSkipButton(skipButtonTitle, theme);
        [skipButton setHidden:!allowSkip];

        NSStackView *stack = [NSStackView stackViewWithViews:@[countdownLabel, skipButton]];
        [stack setOrientation:NSUserInterfaceLayoutOrientationVertical];
        [stack setAlignment:NSLayoutAttributeCenterX];
        [stack setSpacing:16.0];
        [stack setTranslatesAutoresizingMaskIntoConstraints:NO];
        [stack setDetachesHiddenViews:YES];
        [contentView addSubview:stack];

        [NSLayoutConstraint activateConstraints:@[
            [stack.centerXAnchor constraintEqualToAnchor:contentView.centerXAnchor],
            [stack.centerYAnchor constraintEqualToAnchor:contentView.centerYAnchor],
            [skipButton.heightAnchor constraintEqualToConstant:36],
            [skipButton.widthAnchor constraintGreaterThanOrEqualToConstant:170],
        ]];
    }

    if (outCountdownLabel != NULL) {
        *outCountdownLabel = countdownLabel;
    }
    if (outSkipButton != NULL) {
        *outSkipButton = skipButton;
    }
    return window;
}

static void PauseOverlaySetAllowSkipOnMain(BOOL allowSkip) {
    pauseOverlayAllowSkip = allowSkip;
    if (pauseOverlaySkipButtons == nil) {
        return;
    }
    for (NSButton *button in pauseOverlaySkipButtons) {
        [button setHidden:!allowSkip];
        if ([button isKindOfClass:[PauseOverlaySkipButton class]]) {
            PauseOverlaySkipButton *skipButton = (PauseOverlaySkipButton *)button;
            [skipButton setPauseOverlayTitle:pauseOverlaySkipButtonTitle];
            [skipButton setPauseOverlayTheme:pauseOverlayTheme];
            if (!allowSkip) {
                skipButton.pauseOverlayHovered = NO;
                [NSObject cancelPreviousPerformRequestsWithTarget:skipButton selector:@selector(pauseOverlayResetHighlight) object:nil];
                [skipButton setHighlighted:NO];
            }
        }
        PauseOverlayUpdateSkipButtonStyleOnMain(button);
    }
}

static void PauseOverlayUpdateThemeOnMain(NSString *theme) {
    if (theme == nil) {
        theme = @"dark";
    }
    if (pauseOverlayTheme == nil || ![pauseOverlayTheme isEqualToString:theme]) {
        if (pauseOverlayTheme != nil) {
            [pauseOverlayTheme release];
        }
        pauseOverlayTheme = [theme copy];
    }
    if (pauseOverlayWindows != nil) {
        for (NSWindow *window in pauseOverlayWindows) {
            [window setBackgroundColor:PauseOverlayBackgroundColorForTheme(pauseOverlayTheme)];
        }
    }
    if (pauseOverlayCountdownLabels != nil) {
        NSColor *countdownColor = PauseOverlayCountdownColorForTheme(pauseOverlayTheme);
        for (NSTextField *label in pauseOverlayCountdownLabels) {
            [label setTextColor:countdownColor];
        }
    }
    if (pauseOverlaySkipButtons != nil) {
        for (NSButton *button in pauseOverlaySkipButtons) {
            if ([button isKindOfClass:[PauseOverlaySkipButton class]]) {
                PauseOverlaySkipButton *skipButton = (PauseOverlaySkipButton *)button;
                [skipButton setPauseOverlayTheme:pauseOverlayTheme];
                [skipButton setPauseOverlayTitle:pauseOverlaySkipButtonTitle];
            }
            PauseOverlayUpdateSkipButtonStyleOnMain(button);
        }
    }
}

static void PauseOverlayUpdateSkipButtonTitleOnMain(NSString *title) {
    if (title == nil) {
        title = @"Emergency Skip";
    }
    if (pauseOverlaySkipButtonTitle == nil || ![pauseOverlaySkipButtonTitle isEqualToString:title]) {
        if (pauseOverlaySkipButtonTitle != nil) {
            [pauseOverlaySkipButtonTitle release];
        }
        pauseOverlaySkipButtonTitle = [title copy];
    }
    if (pauseOverlaySkipButtons == nil) {
        return;
    }
    for (NSButton *button in pauseOverlaySkipButtons) {
        if ([button isKindOfClass:[PauseOverlaySkipButton class]]) {
            PauseOverlaySkipButton *skipButton = (PauseOverlaySkipButton *)button;
            [skipButton setPauseOverlayTitle:pauseOverlaySkipButtonTitle];
            [skipButton setPauseOverlayTheme:pauseOverlayTheme];
        }
        PauseOverlayUpdateSkipButtonStyleOnMain(button);
    }
}

static void PauseOverlayRebuildWindowsOnMain(BOOL allowSkip, NSString *skipTitle, NSString *countdown, NSString *overlayTheme) {
    if (skipTitle == nil) {
        skipTitle = @"Emergency Skip";
    }
    if (countdown == nil) {
        countdown = @"";
    }
    if (overlayTheme == nil) {
        overlayTheme = @"dark";
    }

    BOOL keepEmergencySkipUnlocked = pauseOverlayEmergencySkipUnlocked;
    PauseOverlayHideOnMain();
    pauseOverlayEmergencySkipUnlocked = keepEmergencySkipUnlocked;
    PauseOverlayEnsureHandler();

    NSArray<NSScreen *> *screens = [NSScreen screens];
    if (screens == nil || [screens count] == 0) {
        pauseOverlayVisible = NO;
        pauseOverlayAllowSkip = NO;
        return;
    }

    pauseOverlayWindows = [[NSMutableArray alloc] init];
    pauseOverlayCountdownLabels = [[NSMutableArray alloc] init];
    pauseOverlaySkipButtons = [[NSMutableArray alloc] init];
    pauseOverlayAllowSkip = allowSkip;
    pauseOverlaySkipButtonTitle = [skipTitle copy];
    pauseOverlayCountdownText = [countdown copy];
    pauseOverlayTheme = [overlayTheme copy];

    for (NSUInteger i = 0; i < [screens count]; i++) {
        NSScreen *screen = [screens objectAtIndex:i];
        NSTextField *countdownLabel = nil;
        NSButton *skipButton = nil;
        NSWindow *window = PauseOverlayBuildWindowForScreen(screen, allowSkip, skipTitle, countdown, overlayTheme, &countdownLabel, &skipButton);
        if (window == nil) {
            continue;
        }
        [pauseOverlayWindows addObject:window];
        if (countdownLabel != nil) {
            [pauseOverlayCountdownLabels addObject:countdownLabel];
        }
        if (skipButton != nil) {
            [pauseOverlaySkipButtons addObject:skipButton];
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
}

static void PauseOverlayUnlockEmergencySkipOnMain(void) {
    if (pauseOverlayVisible == NO || pauseOverlayAllowSkip == YES) {
        return;
    }
    pauseOverlayEmergencySkipUnlocked = YES;
    PauseOverlaySetAllowSkipOnMain(YES);
}

static void PauseOverlayInstallKeyMonitor(void) {
    if (pauseOverlayKeyMonitor != nil) {
        return;
    }
    pauseOverlayKeyMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyDown handler:^NSEvent * _Nullable(NSEvent * _Nonnull event) {
        if (pauseOverlayVisible == NO) {
            return event;
        }
        NSEventModifierFlags flags = [event modifierFlags] & NSEventModifierFlagDeviceIndependentFlagsMask;
        if ((flags & NSEventModifierFlagCommand) == 0) {
            return event;
        }
        if ((flags & (NSEventModifierFlagControl | NSEventModifierFlagOption)) != 0) {
            return event;
        }
        NSString *key = [[event charactersIgnoringModifiers] lowercaseString];
        if (key == nil || [key length] == 0) {
            return event;
        }

        // Always block close/hide shortcuts while break overlay is active.
        if ([key isEqualToString:@"w"] || [key isEqualToString:@"h"]) {
            return nil;
        }
        if (![key isEqualToString:@"q"]) {
            return event;
        }

        // Require quick double Cmd+Q to reveal emergency skip when skip is disabled.
        NSTimeInterval now = [NSDate timeIntervalSinceReferenceDate];
        if (!pauseOverlayAllowSkip) {
            if (pauseOverlayLastCmdQPress > 0 && (now - pauseOverlayLastCmdQPress) <= pauseOverlayCmdQDoublePressWindow) {
                pauseOverlayLastCmdQPress = 0;
                PauseOverlayUnlockEmergencySkipOnMain();
            } else {
                pauseOverlayLastCmdQPress = now;
            }
            return nil;
        }

        // When skip button is visible, still block Cmd+Q to avoid accidental app quit.
        return nil;
    }];
}

static void PauseOverlayHideOnMain(void) {
    PauseOverlayRemoveKeyMonitor();
    pauseOverlayEmergencySkipUnlocked = NO;
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
    if (pauseOverlaySkipButtons != nil) {
        for (NSButton *button in pauseOverlaySkipButtons) {
            if ([button isKindOfClass:[PauseOverlaySkipButton class]]) {
                PauseOverlaySkipButton *skipButton = (PauseOverlaySkipButton *)button;
                [NSObject cancelPreviousPerformRequestsWithTarget:skipButton selector:@selector(pauseOverlayResetHighlight) object:nil];
                [skipButton setHighlighted:NO];
            }
        }
        [pauseOverlaySkipButtons release];
        pauseOverlaySkipButtons = nil;
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
        BOOL effectiveAllowSkip = shouldAllowSkip || pauseOverlayEmergencySkipUnlocked;

        if (screens == nil || [screens count] == 0) {
            PauseOverlayHideOnMain();
            didShow = NO;
            return;
        }

        BOOL needsRebuild = (pauseOverlayVisible == NO || pauseOverlayWindows == nil || [pauseOverlayWindows count] != [screens count]);
        if (needsRebuild) {
            PauseOverlayRebuildWindowsOnMain(effectiveAllowSkip, skipTitle, countdown, overlayTheme);
        } else {
            PauseOverlaySetAllowSkipOnMain(effectiveAllowSkip);
            PauseOverlayUpdateSkipButtonTitleOnMain(skipTitle);
            PauseOverlayUpdateThemeOnMain(overlayTheme);
            PauseOverlayUpdateCountdownTextOnMain(countdown);
        }
        didShow = pauseOverlayVisible;
        if (didShow) {
            PauseOverlayInstallKeyMonitor();
        }
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
