//go:build darwin && wails

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void statusBarMenuCallbackGo(int callbackID);

enum {
    PauseStatusBarActionBreakNow = 1,
    PauseStatusBarActionPause = 2,
    PauseStatusBarActionPause30 = 3,
    PauseStatusBarActionResume = 4,
    PauseStatusBarActionOpenWindow = 5,
    PauseStatusBarActionQuit = 6
};

static void PauseClosePopover(void);
static void PauseInstallPopoverAutoClose(void);
static void PauseRemovePopoverAutoClose(void);
static void PauseSetActionButtonsPausedState(BOOL paused);
static void PauseStatusBarApplyIcon(BOOL blinkingOnPhase);
static void PauseStatusBarSetPausedBlinking(BOOL paused);
static NSImage *PauseStatusBarBuildIconByName(NSString *symbolName, CGFloat pointSize);
static void PauseStatusBarSetTitleText(NSString *text);
static void PauseStatusBarAdjustLengthForTitle(NSString *text);
static void PauseUpdateStatusItemTooltipVisibility(void);

static const CGFloat pauseStatusItemWidthWithTime = 68.0;
static const CGFloat pauseStatusItemWidthIconOnly = 26.0;

@interface PauseStatusBarHandler : NSObject
- (void)onStatusItemClick:(id)sender;
- (void)onActionClick:(id)sender;
- (void)onMoreClick:(id)sender;
- (void)onMenuAbout:(id)sender;
- (void)onMenuQuit:(id)sender;
- (void)onAppDidResignActive:(NSNotification *)notification;
- (void)onBlinkTick:(NSTimer *)timer;
@end

@implementation PauseStatusBarHandler
- (void)onStatusItemClick:(id)sender {
    (void)sender;
    extern NSPopover *pausePopover;
    extern NSStatusItem *pauseStatusItem;

    if (pausePopover == nil || pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    if ([pausePopover isShown]) {
        PauseClosePopover();
        return;
    }
    NSArray<NSWindow *> *windowsBefore = [[NSApp windows] copy];
    NSMutableSet<NSWindow *> *visibleBefore = [NSMutableSet set];
    for (NSWindow *window in windowsBefore) {
        if (window != nil && [window isVisible]) {
            [visibleBefore addObject:window];
        }
    }

    if (![NSApp isActive]) {
        [[NSApplication sharedApplication] activateIgnoringOtherApps:YES];
    }
    [pausePopover showRelativeToRect:pauseStatusItem.button.bounds ofView:pauseStatusItem.button preferredEdge:NSRectEdgeMinY];
    PauseUpdateStatusItemTooltipVisibility();
    NSWindow *popoverWindow = pausePopover.contentViewController.view.window;
    if (popoverWindow != nil) {
        [popoverWindow makeKeyAndOrderFront:nil];
    }

    for (NSWindow *window in windowsBefore) {
        if (window == nil || window == popoverWindow) {
            continue;
        }
        if (![visibleBefore containsObject:window] && [window isVisible]) {
            [window orderOut:nil];
        }
    }
    [windowsBefore release];

    PauseInstallPopoverAutoClose();
}

- (void)onActionClick:(id)sender {
    NSInteger callbackID = [sender tag];
    if (callbackID == PauseStatusBarActionOpenWindow) {
        PauseClosePopover();
    }
    statusBarMenuCallbackGo((int)callbackID);
}

- (void)onMoreClick:(id)sender {
    (void)sender;
    extern NSButton *pauseMoreButton;
    if (pauseMoreButton == nil) {
        return;
    }

    extern NSString *pauseAboutMenuTitle;
    extern NSString *pauseQuitMenuTitle;
    NSString *aboutTitle = pauseAboutMenuTitle ? pauseAboutMenuTitle : @"About";
    NSString *quitTitle = pauseQuitMenuTitle ? pauseQuitMenuTitle : @"Quit";

    NSMenu *menu = [[[NSMenu alloc] initWithTitle:@""] autorelease];
    NSMenuItem *aboutItem = [[[NSMenuItem alloc] initWithTitle:aboutTitle action:@selector(onMenuAbout:) keyEquivalent:@""] autorelease];
    [aboutItem setTarget:self];
    [menu addItem:aboutItem];
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quitItem = [[[NSMenuItem alloc] initWithTitle:quitTitle action:@selector(onMenuQuit:) keyEquivalent:@""] autorelease];
    [quitItem setTarget:self];
    [menu addItem:quitItem];

    [menu popUpMenuPositioningItem:nil atLocation:NSMakePoint(NSMinX(pauseMoreButton.bounds) + 1.0, NSMaxY(pauseMoreButton.bounds) - 1.0) inView:pauseMoreButton];
}

- (void)onMenuAbout:(id)sender {
    (void)sender;
    PauseClosePopover();
    [[NSApplication sharedApplication] activateIgnoringOtherApps:YES];
    [NSApp orderFrontStandardAboutPanel:nil];
}

- (void)onMenuQuit:(id)sender {
    (void)sender;
    PauseClosePopover();
    statusBarMenuCallbackGo(PauseStatusBarActionQuit);
}

- (void)onAppDidResignActive:(NSNotification *)notification {
    (void)notification;
    PauseClosePopover();
}

- (void)onBlinkTick:(NSTimer *)timer {
    (void)timer;
    extern BOOL pauseStatusBlinkVisible;
    pauseStatusBlinkVisible = !pauseStatusBlinkVisible;
    PauseStatusBarApplyIcon(pauseStatusBlinkVisible);
}
@end

NSStatusItem *pauseStatusItem;
NSPopover *pausePopover;
NSViewController *pausePopoverController;
NSTextField *pausePopoverTitleLabel;
NSTextField *pauseStatusLabel;
NSTextField *pauseCountdownLabel;
NSProgressIndicator *pauseCountdownProgress;
NSButton *pausePauseButton;
NSButton *pausePause30Button;
NSButton *pauseResumeButton;
NSButton *pauseBreakNowButton;
NSButton *pauseOpenButton;
NSButton *pauseMoreButton;
NSString *pauseAboutMenuTitle;
NSString *pauseQuitMenuTitle;
NSString *pauseMoreButtonTip;
NSString *pauseStatusTooltipText;
id pauseLocalMonitor;
id pauseGlobalMonitor;
PauseStatusBarHandler *pauseStatusHandler;
NSImage *pauseStatusIcon;
NSImage *pauseStatusIconAlt;
NSTimer *pauseStatusBlinkTimer;
BOOL pauseStatusBlinkVisible;
BOOL pauseStatusPaused;

static void PauseRunOnMain(void (^block)(void)) {
    if ([NSThread isMainThread]) {
        block();
        return;
    }
    dispatch_sync(dispatch_get_main_queue(), block);
}

static NSTextField *MakeLabel(NSString *text, NSFont *font, NSColor *color) {
    NSTextField *label = [NSTextField labelWithString:text];
    [label setFont:font];
    [label setTextColor:color];
    [label setAlignment:NSTextAlignmentLeft];
    [label setLineBreakMode:NSLineBreakByTruncatingTail];
    [label setTranslatesAutoresizingMaskIntoConstraints:NO];
    return label;
}

static NSButton *MakeActionButton(NSString *title, int callbackID) {
    NSButton *button = [NSButton buttonWithTitle:title target:pauseStatusHandler action:@selector(onActionClick:)];
    [button setBezelStyle:NSBezelStyleRounded];
    [button setFont:[NSFont systemFontOfSize:12 weight:NSFontWeightRegular]];
    [button setTag:callbackID];
    [button setTranslatesAutoresizingMaskIntoConstraints:NO];
    return button;
}

static NSButton *MakeIconActionButton(NSString *symbolName, NSString *fallbackTitle, int callbackID) {
    NSButton *button = [NSButton buttonWithTitle:@"" target:pauseStatusHandler action:@selector(onActionClick:)];
    [button setBezelStyle:NSBezelStyleTexturedRounded];
    [button setTag:callbackID];
    [button setTranslatesAutoresizingMaskIntoConstraints:NO];
    [button setToolTip:fallbackTitle];
    NSImage *icon = PauseStatusBarBuildIconByName(symbolName, 13.0);
    if (icon != nil) {
        [button setImage:icon];
        [button setImagePosition:NSImageOnly];
        [icon autorelease];
    }
    return button;
}

static NSImage *PauseStatusBarBuildIconByName(NSString *symbolName, CGFloat pointSize) {
    NSImage *icon = nil;
    if ([NSImage respondsToSelector:@selector(imageWithSystemSymbolName:accessibilityDescription:)]) {
        NSImage *base = [NSImage imageWithSystemSymbolName:symbolName accessibilityDescription:@"Pause"];
        if (base != nil) {
            if ([NSImageSymbolConfiguration respondsToSelector:@selector(configurationWithPointSize:weight:)]) {
                NSImageSymbolConfiguration *cfg = [NSImageSymbolConfiguration configurationWithPointSize:pointSize weight:NSFontWeightSemibold];
                icon = [[base imageWithSymbolConfiguration:cfg] copy];
            } else {
                icon = [base copy];
            }
        }
    }
    if (icon == nil) {
        NSImage *fallback = [NSImage imageNamed:NSImageNameTouchBarPauseTemplate];
        if (fallback != nil) {
            icon = [fallback copy];
        }
    }
    if (icon != nil) {
        [icon setTemplate:YES];
        [icon setSize:NSMakeSize(pointSize, pointSize)];
    }
    return icon;
}

static void PauseStatusBarApplyIcon(BOOL blinkingOnPhase) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    if (pauseStatusPaused && !blinkingOnPhase && pauseStatusIconAlt != nil) {
        [pauseStatusItem.button setImage:pauseStatusIconAlt];
        return;
    }
    [pauseStatusItem.button setImage:pauseStatusIcon];
}

static void PauseStatusBarSetTitleText(NSString *text) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    NSString *safeText = text ? text : @"";
    PauseStatusBarAdjustLengthForTitle(safeText);
    NSDictionary *attrs = @{
        NSFontAttributeName: [NSFont monospacedDigitSystemFontOfSize:13 weight:NSFontWeightSemibold],
        NSForegroundColorAttributeName: [NSColor labelColor],
        NSBaselineOffsetAttributeName: @(-1.0),
    };
    NSAttributedString *title = [[[NSAttributedString alloc] initWithString:safeText attributes:attrs] autorelease];
    [pauseStatusItem.button setAttributedTitle:title];
}

static void PauseStatusBarAdjustLengthForTitle(NSString *text) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    BOOL hasText = text != nil && [text length] > 0;
    [pauseStatusItem setLength:(hasText ? pauseStatusItemWidthWithTime : pauseStatusItemWidthIconOnly)];
    [pauseStatusItem.button setImagePosition:(hasText ? NSImageLeft : NSImageOnly)];
}

static void PauseStatusBarSetPausedBlinking(BOOL paused) {
    pauseStatusPaused = paused;
    if (!paused) {
        if (pauseStatusBlinkTimer != nil) {
            [pauseStatusBlinkTimer invalidate];
            pauseStatusBlinkTimer = nil;
        }
        pauseStatusBlinkVisible = YES;
        PauseStatusBarApplyIcon(YES);
        return;
    }

    if (pauseStatusBlinkTimer != nil) {
        return;
    }
    pauseStatusBlinkVisible = YES;
    PauseStatusBarApplyIcon(YES);
    pauseStatusBlinkTimer = [NSTimer timerWithTimeInterval:0.6 target:pauseStatusHandler selector:@selector(onBlinkTick:) userInfo:nil repeats:YES];
    [[NSRunLoop mainRunLoop] addTimer:pauseStatusBlinkTimer forMode:NSRunLoopCommonModes];
}

static void PauseSetActionButtonsPausedState(BOOL paused) {
    if (pauseBreakNowButton != nil) {
        [pauseBreakNowButton setHidden:NO];
    }
    if (pausePauseButton != nil) {
        [pausePauseButton setHidden:paused];
    }
    if (pausePause30Button != nil) {
        [pausePause30Button setHidden:paused];
    }
    if (pauseResumeButton != nil) {
        [pauseResumeButton setHidden:!paused];
    }
}

static void PauseRemovePopoverAutoClose(void) {
    if (pauseLocalMonitor != nil) {
        [NSEvent removeMonitor:pauseLocalMonitor];
        pauseLocalMonitor = nil;
    }
    if (pauseGlobalMonitor != nil) {
        [NSEvent removeMonitor:pauseGlobalMonitor];
        pauseGlobalMonitor = nil;
    }
}

static void PauseClosePopover(void) {
    if (pausePopover != nil && [pausePopover isShown]) {
        [pausePopover close];
    }
    PauseRemovePopoverAutoClose();
    PauseUpdateStatusItemTooltipVisibility();
}

static void PauseInstallPopoverAutoClose(void) {
    PauseRemovePopoverAutoClose();

    pauseLocalMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:(NSEventMaskLeftMouseDown | NSEventMaskRightMouseDown | NSEventMaskOtherMouseDown | NSEventMaskKeyDown) handler:^NSEvent * _Nullable(NSEvent * _Nonnull event) {
        if (pausePopover == nil || ![pausePopover isShown]) {
            PauseRemovePopoverAutoClose();
            return event;
        }
        if ([event type] == NSEventTypeKeyDown && [event keyCode] == 53) {
            PauseClosePopover();
            return nil;
        }

        NSWindow *eventWindow = [event window];
        NSWindow *popoverWindow = pausePopover.contentViewController.view.window;
        NSWindow *statusWindow = pauseStatusItem.button.window;
        if (eventWindow != popoverWindow && eventWindow != statusWindow) {
            PauseClosePopover();
        }
        return event;
    }];

    pauseGlobalMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:(NSEventMaskLeftMouseDown | NSEventMaskRightMouseDown | NSEventMaskOtherMouseDown) handler:^(NSEvent * _Nonnull event) {
        (void)event;
        PauseRunOnMain(^{
            PauseClosePopover();
        });
    }];
}

static void PauseUpdateStatusItemTooltipVisibility(void) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    if (pauseStatusTooltipText != nil) {
        [pauseStatusItem.button setToolTip:pauseStatusTooltipText];
    }
}

static void BuildPopoverContent(void) {
    NSView *contentView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 300, 210)];
    [contentView setTranslatesAutoresizingMaskIntoConstraints:NO];

    NSStackView *stack = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [stack setOrientation:NSUserInterfaceLayoutOrientationVertical];
    [stack setAlignment:NSLayoutAttributeLeading];
    [stack setSpacing:10.0];
    [stack setTranslatesAutoresizingMaskIntoConstraints:NO];

    pausePopoverTitleLabel = MakeLabel(@"Pause", [NSFont systemFontOfSize:14 weight:NSFontWeightSemibold], [NSColor labelColor]);
    NSView *headerView = [[NSView alloc] initWithFrame:NSZeroRect];
    [headerView setTranslatesAutoresizingMaskIntoConstraints:NO];
    [headerView addSubview:pausePopoverTitleLabel];

    pauseMoreButton = [NSButton buttonWithTitle:@"…" target:pauseStatusHandler action:@selector(onMoreClick:)];
    [pauseMoreButton setBezelStyle:NSBezelStyleRoundRect];
    [pauseMoreButton setShowsBorderOnlyWhileMouseInside:YES];
    [pauseMoreButton setFont:[NSFont systemFontOfSize:13 weight:NSFontWeightSemibold]];
    [pauseMoreButton setFocusRingType:NSFocusRingTypeNone];
    [pauseMoreButton setTranslatesAutoresizingMaskIntoConstraints:NO];
    [pauseMoreButton setToolTip:@"More"];
    [headerView addSubview:pauseMoreButton];
    [NSLayoutConstraint activateConstraints:@[
        [pausePopoverTitleLabel.leadingAnchor constraintEqualToAnchor:headerView.leadingAnchor],
        [pausePopoverTitleLabel.centerYAnchor constraintEqualToAnchor:headerView.centerYAnchor],
        [pauseMoreButton.trailingAnchor constraintEqualToAnchor:headerView.trailingAnchor],
        [pauseMoreButton.centerYAnchor constraintEqualToAnchor:headerView.centerYAnchor constant:-1],
        [pauseMoreButton.widthAnchor constraintEqualToConstant:26],
        [pauseMoreButton.heightAnchor constraintEqualToConstant:22],
        [pausePopoverTitleLabel.trailingAnchor constraintLessThanOrEqualToAnchor:pauseMoreButton.leadingAnchor constant:-8],
    ]];
    pauseStatusLabel = MakeLabel(@"Status: running", [NSFont systemFontOfSize:12 weight:NSFontWeightRegular], [NSColor secondaryLabelColor]);
    pauseCountdownLabel = MakeLabel(@"Next break: --:--", [NSFont monospacedDigitSystemFontOfSize:13 weight:NSFontWeightMedium], [NSColor labelColor]);
    [pauseCountdownLabel setAlignment:NSTextAlignmentCenter];
    pauseCountdownProgress = [[NSProgressIndicator alloc] initWithFrame:NSZeroRect];
    [pauseCountdownProgress setIndeterminate:NO];
    [pauseCountdownProgress setMinValue:0];
    [pauseCountdownProgress setMaxValue:100];
    [pauseCountdownProgress setDoubleValue:0];
    [pauseCountdownProgress setControlSize:NSControlSizeSmall];
    [pauseCountdownProgress setStyle:NSProgressIndicatorStyleBar];
    [pauseCountdownProgress setTranslatesAutoresizingMaskIntoConstraints:NO];

    NSStackView *rowControls = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [rowControls setOrientation:NSUserInterfaceLayoutOrientationHorizontal];
    [rowControls setDistribution:NSStackViewDistributionFill];
    [rowControls setAlignment:NSLayoutAttributeCenterY];
    [rowControls setSpacing:8.0];
    pausePauseButton = MakeIconActionButton(@"pause.fill", @"Pause", PauseStatusBarActionPause);
    pauseResumeButton = MakeIconActionButton(@"play.fill", @"Resume", PauseStatusBarActionResume);
    [rowControls addArrangedSubview:pausePauseButton];
    [rowControls addArrangedSubview:pauseResumeButton];
    [rowControls addArrangedSubview:pauseCountdownProgress];

    pauseBreakNowButton = MakeActionButton(@"zzZ", PauseStatusBarActionBreakNow);
    NSView *breakNowContainer = [[NSView alloc] initWithFrame:NSZeroRect];
    [breakNowContainer setTranslatesAutoresizingMaskIntoConstraints:NO];
    [breakNowContainer addSubview:pauseBreakNowButton];
    [NSLayoutConstraint activateConstraints:@[
        [pauseBreakNowButton.centerXAnchor constraintEqualToAnchor:breakNowContainer.centerXAnchor],
        [pauseBreakNowButton.topAnchor constraintEqualToAnchor:breakNowContainer.topAnchor],
        [pauseBreakNowButton.bottomAnchor constraintEqualToAnchor:breakNowContainer.bottomAnchor],
        [pauseBreakNowButton.heightAnchor constraintEqualToConstant:28],
        [pauseBreakNowButton.widthAnchor constraintEqualToConstant:72],
    ]];

    pauseOpenButton = MakeActionButton(@"Open Pause", PauseStatusBarActionOpenWindow);
    [pauseOpenButton setAlignment:NSTextAlignmentCenter];
    NSView *openButtonContainer = [[NSView alloc] initWithFrame:NSZeroRect];
    [openButtonContainer setTranslatesAutoresizingMaskIntoConstraints:NO];
    [openButtonContainer addSubview:pauseOpenButton];
    [NSLayoutConstraint activateConstraints:@[
        [pauseOpenButton.leadingAnchor constraintEqualToAnchor:openButtonContainer.leadingAnchor],
        [pauseOpenButton.trailingAnchor constraintEqualToAnchor:openButtonContainer.trailingAnchor],
        [pauseOpenButton.topAnchor constraintEqualToAnchor:openButtonContainer.topAnchor],
        [pauseOpenButton.bottomAnchor constraintEqualToAnchor:openButtonContainer.bottomAnchor],
        [pauseOpenButton.heightAnchor constraintEqualToConstant:28],
    ]];

    NSBox *separator = [[NSBox alloc] initWithFrame:NSZeroRect];
    [separator setBoxType:NSBoxSeparator];

    [stack addArrangedSubview:headerView];
    [stack addArrangedSubview:pauseStatusLabel];
    [stack addArrangedSubview:pauseCountdownLabel];
    [stack addArrangedSubview:rowControls];
    [stack addArrangedSubview:breakNowContainer];
    [stack addArrangedSubview:separator];
    [stack addArrangedSubview:openButtonContainer];

    [contentView addSubview:stack];

    [NSLayoutConstraint activateConstraints:@[
        [stack.leadingAnchor constraintEqualToAnchor:contentView.leadingAnchor constant:12],
        [stack.trailingAnchor constraintEqualToAnchor:contentView.trailingAnchor constant:-12],
        [stack.topAnchor constraintEqualToAnchor:contentView.topAnchor constant:12],
        [stack.bottomAnchor constraintEqualToAnchor:contentView.bottomAnchor constant:-12],
        [headerView.widthAnchor constraintEqualToAnchor:stack.widthAnchor],
        [headerView.heightAnchor constraintEqualToConstant:24],
        [pauseCountdownLabel.leadingAnchor constraintEqualToAnchor:stack.leadingAnchor],
        [pauseCountdownLabel.trailingAnchor constraintEqualToAnchor:stack.trailingAnchor],
        [pausePauseButton.widthAnchor constraintEqualToConstant:32],
        [pauseResumeButton.widthAnchor constraintEqualToConstant:32],
        [pauseCountdownProgress.heightAnchor constraintEqualToConstant:8],
        [rowControls.heightAnchor constraintEqualToConstant:28],
        [breakNowContainer.heightAnchor constraintEqualToConstant:28],
        [openButtonContainer.widthAnchor constraintEqualToAnchor:stack.widthAnchor],
        [openButtonContainer.heightAnchor constraintEqualToConstant:28],
    ]];

    pausePopoverController = [[NSViewController alloc] init];
    [pausePopoverController setView:contentView];
    [contentView release];

    pausePopover = [[NSPopover alloc] init];
    [pausePopover setBehavior:NSPopoverBehaviorTransient];
    [pausePopover setContentSize:NSMakeSize(300, 220)];
    [pausePopover setContentViewController:pausePopoverController];
    PauseSetActionButtonsPausedState(NO);

    [openButtonContainer release];
    [breakNowContainer release];
    [headerView release];
    [pauseCountdownProgress release];
    [rowControls release];
    [separator release];
    [stack release];
}

void PauseStatusBarInit(void) {
    PauseRunOnMain(^{
        if (pauseStatusItem != nil) {
            return;
        }

        pauseStatusHandler = [PauseStatusBarHandler new];
        pauseStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        [pauseStatusItem retain];
        [pauseStatusItem setLength:pauseStatusItemWidthWithTime];
        pauseStatusIcon = PauseStatusBarBuildIconByName(@"pause.circle.fill", 15.0);
        pauseStatusIconAlt = PauseStatusBarBuildIconByName(@"pause.circle", 15.0);
        pauseStatusPaused = NO;
        pauseStatusBlinkVisible = YES;

        PauseStatusBarSetTitleText(@"");
        [pauseStatusItem.button setImagePosition:NSImageLeft];
        [pauseStatusItem.button setImageScaling:NSImageScaleProportionallyDown];
        [pauseStatusItem.button setImage:pauseStatusIcon];
        pauseStatusTooltipText = [@"Pause break reminder" copy];
        [pauseStatusItem.button setToolTip:pauseStatusTooltipText];
        [pauseStatusItem.button setTarget:pauseStatusHandler];
        [pauseStatusItem.button setAction:@selector(onStatusItemClick:)];
        [pauseStatusItem.button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
        [[NSNotificationCenter defaultCenter] addObserver:pauseStatusHandler selector:@selector(onAppDidResignActive:) name:NSApplicationDidResignActiveNotification object:nil];

        BuildPopoverContent();
    });
}

void PauseStatusBarUpdate(const char *status, const char *countdown, const char *title, int paused, double progress) {
    NSString *statusText = status ? [NSString stringWithUTF8String:status] : @"Status: running";
    NSString *countdownText = countdown ? [NSString stringWithUTF8String:countdown] : @"Next break: --:--";
    NSString *titleText = title ? [NSString stringWithUTF8String:title] : @"";

    PauseRunOnMain(^{
        if (pauseStatusItem == nil) {
            return;
        }
        PauseStatusBarSetTitleText(titleText);
        PauseStatusBarSetPausedBlinking(paused != 0);
        if (pauseStatusLabel != nil) {
            [pauseStatusLabel setStringValue:statusText];
        }
        if (pauseCountdownLabel != nil) {
            [pauseCountdownLabel setStringValue:countdownText];
        }
        if (pauseCountdownProgress != nil) {
            double clamped = progress;
            if (clamped < 0) {
                clamped = 0;
            }
            if (clamped > 1) {
                clamped = 1;
            }
            [pauseCountdownProgress setDoubleValue:(clamped * 100.0)];
        }
        PauseSetActionButtonsPausedState(paused != 0);
    });
}

void PauseStatusBarSetLocaleStrings(
    const char *popoverTitle,
    const char *breakNowButton,
    const char *pauseButton,
    const char *pause30Button,
    const char *resumeButton,
    const char *openButton,
    const char *aboutMenuItem,
    const char *quitMenuItem,
    const char *moreButtonTip,
    const char *tooltip
) {
    NSString *popoverTitleText = popoverTitle ? [NSString stringWithUTF8String:popoverTitle] : @"Pause";
    NSString *breakNowButtonText = breakNowButton ? [NSString stringWithUTF8String:breakNowButton] : @"zzZ";
    NSString *pauseButtonText = pauseButton ? [NSString stringWithUTF8String:pauseButton] : @"Pause";
    NSString *pause30ButtonText = pause30Button ? [NSString stringWithUTF8String:pause30Button] : @"Pause 30m";
    NSString *resumeButtonText = resumeButton ? [NSString stringWithUTF8String:resumeButton] : @"Resume";
    NSString *openButtonText = openButton ? [NSString stringWithUTF8String:openButton] : @"Open Pause";
    NSString *aboutMenuText = aboutMenuItem ? [NSString stringWithUTF8String:aboutMenuItem] : @"About";
    NSString *quitMenuText = quitMenuItem ? [NSString stringWithUTF8String:quitMenuItem] : @"Quit";
    NSString *moreButtonTipText = moreButtonTip ? [NSString stringWithUTF8String:moreButtonTip] : @"More";
    NSString *tooltipText = tooltip ? [NSString stringWithUTF8String:tooltip] : @"Pause break reminder";

    PauseRunOnMain(^{
        if (pauseStatusItem == nil) {
            return;
        }
        if (pauseStatusTooltipText != nil) {
            [pauseStatusTooltipText release];
            pauseStatusTooltipText = nil;
        }
        pauseStatusTooltipText = [tooltipText copy];
        PauseUpdateStatusItemTooltipVisibility();
        if (pausePopoverTitleLabel != nil) {
            [pausePopoverTitleLabel setStringValue:popoverTitleText];
        }
        if (pausePauseButton != nil) {
            [pausePauseButton setTitle:@""];
            [pausePauseButton setToolTip:pauseButtonText];
        }
        if (pauseBreakNowButton != nil) {
            [pauseBreakNowButton setTitle:breakNowButtonText];
        }
        if (pausePause30Button != nil) {
            [pausePause30Button setTitle:pause30ButtonText];
        }
        if (pauseResumeButton != nil) {
            [pauseResumeButton setTitle:@""];
            [pauseResumeButton setToolTip:resumeButtonText];
        }
        if (pauseOpenButton != nil) {
            [pauseOpenButton setTitle:openButtonText];
        }
        if (pauseMoreButton != nil) {
            [pauseMoreButton setToolTip:moreButtonTipText];
        }
        if (pauseAboutMenuTitle != nil) {
            [pauseAboutMenuTitle release];
            pauseAboutMenuTitle = nil;
        }
        pauseAboutMenuTitle = [aboutMenuText copy];
        if (pauseQuitMenuTitle != nil) {
            [pauseQuitMenuTitle release];
            pauseQuitMenuTitle = nil;
        }
        pauseQuitMenuTitle = [quitMenuText copy];
        if (pauseMoreButtonTip != nil) {
            [pauseMoreButtonTip release];
            pauseMoreButtonTip = nil;
        }
        pauseMoreButtonTip = [moreButtonTipText copy];
    });
}

void PauseStatusBarDestroy(void) {
    PauseRunOnMain(^{
        PauseClosePopover();
        PauseStatusBarSetPausedBlinking(NO);
        if (pauseStatusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:pauseStatusItem];
            [pauseStatusItem release];
        }
        if (pausePopover != nil) {
            [pausePopover release];
        }
        if (pausePopoverController != nil) {
            [pausePopoverController release];
        }
        if (pauseStatusHandler != nil) {
            [[NSNotificationCenter defaultCenter] removeObserver:pauseStatusHandler];
            [pauseStatusHandler release];
        }
        if (pauseStatusIcon != nil) {
            [pauseStatusIcon release];
        }
        if (pauseStatusIconAlt != nil) {
            [pauseStatusIconAlt release];
        }
        if (pauseAboutMenuTitle != nil) {
            [pauseAboutMenuTitle release];
        }
        if (pauseQuitMenuTitle != nil) {
            [pauseQuitMenuTitle release];
        }
        if (pauseMoreButtonTip != nil) {
            [pauseMoreButtonTip release];
        }
        if (pauseStatusTooltipText != nil) {
            [pauseStatusTooltipText release];
        }

        pauseStatusItem = nil;
        pausePopover = nil;
        pausePopoverController = nil;
        pausePopoverTitleLabel = nil;
        pauseStatusLabel = nil;
        pauseCountdownLabel = nil;
        pauseCountdownProgress = nil;
        pausePauseButton = nil;
        pausePause30Button = nil;
        pauseResumeButton = nil;
        pauseBreakNowButton = nil;
        pauseOpenButton = nil;
        pauseMoreButton = nil;
        pauseAboutMenuTitle = nil;
        pauseQuitMenuTitle = nil;
        pauseMoreButtonTip = nil;
        pauseStatusTooltipText = nil;
        pauseLocalMonitor = nil;
        pauseGlobalMonitor = nil;
        pauseStatusHandler = nil;
        pauseStatusIcon = nil;
        pauseStatusIconAlt = nil;
        pauseStatusBlinkTimer = nil;
        pauseStatusBlinkVisible = YES;
        pauseStatusPaused = NO;
    });
}
*/
import "C"
