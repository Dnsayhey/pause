//go:build darwin && wails

package macbridge

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework Foundation

#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void statusBarMenuCallbackGo(int callbackID);
extern void statusBarPopoverVisibilityCallbackGo(int visible);

enum {
    PauseStatusBarActionBreakNow = 1,
    PauseStatusBarActionPause = 2,
    PauseStatusBarActionResume = 4,
    PauseStatusBarActionOpenWindow = 5,
    PauseStatusBarActionQuit = 6,
    PauseStatusBarActionPauseReminderBase = 1000,
    PauseStatusBarActionResumeReminderBase = 2000
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
static void PauseClearReminderRows(void);
static void PauseRebuildReminderRows(NSString *remindersPayload);
static void PauseSetLatestRemindersPayload(NSString *remindersPayload);
static void PauseRefreshReminderRowsIfNeeded(void);
static NSString *PauseNoRemindersText(void);
static void PauseAppendNoReminderRow(NSString *titleText);
static void PauseUpdatePopoverSize(void);

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
    statusBarPopoverVisibilityCallbackGo(1);
    PauseRefreshReminderRowsIfNeeded();
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
NSView *pausePopoverContentView;
NSTextField *pausePopoverTitleLabel;
NSTextField *pauseStatusLabel;
NSStackView *pauseRemindersStack;
NSMutableArray *pauseReminderPauseButtons;
NSMutableArray *pauseReminderResumeButtons;
NSMutableArray *pauseReminderProgressBars;
NSButton *pauseGlobalToggleButton;
NSButton *pauseBreakNowButton;
NSButton *pauseOpenButton;
NSButton *pauseMoreButton;
NSString *pauseAboutMenuTitle;
NSString *pauseQuitMenuTitle;
NSString *pauseMoreButtonTip;
NSString *pauseStatusTooltipText;
NSString *pausePauseButtonTooltip;
NSString *pauseResumeButtonTooltip;
NSString *pauseGlobalPauseButtonTitle;
NSString *pauseGlobalResumeButtonTitle;
NSString *pauseLatestRemindersPayload;
NSString *pauseRenderedRemindersPayload;
NSString *pauseStatusCurrentTitle;
id pauseLocalMonitor;
id pauseGlobalMonitor;
PauseStatusBarHandler *pauseStatusHandler;
NSImage *pauseStatusIcon;
NSImage *pauseStatusIconAlt;
NSTimer *pauseStatusBlinkTimer;
BOOL pauseStatusBlinkVisible;
BOOL pauseStatusPaused;
BOOL pauseStatusLayoutHasText;
BOOL pauseStatusLayoutHasTextKnown;

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
    NSImage *target = pauseStatusIcon;
    if (pauseStatusPaused && !blinkingOnPhase && pauseStatusIconAlt != nil) {
        target = pauseStatusIconAlt;
    }
    if ([pauseStatusItem.button image] == target) {
        return;
    }
    [pauseStatusItem.button setImage:target];
}

static void PauseStatusBarSetTitleText(NSString *text) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    NSString *safeText = text ? text : @"";
    if (pauseStatusCurrentTitle != nil) {
        [pauseStatusCurrentTitle release];
        pauseStatusCurrentTitle = nil;
    }
    pauseStatusCurrentTitle = [safeText copy];
    PauseStatusBarAdjustLengthForTitle(safeText);
    NSDictionary *attrs = @{
        NSFontAttributeName: [NSFont monospacedDigitSystemFontOfSize:13 weight:NSFontWeightSemibold],
        NSBaselineOffsetAttributeName: @(-1.0),
    };
    NSAttributedString *title = [[[NSAttributedString alloc] initWithString:safeText attributes:attrs] autorelease];
    NSAttributedString *currentTitle = [pauseStatusItem.button attributedTitle];
    if (currentTitle != nil && [currentTitle isEqualToAttributedString:title]) {
        return;
    }
    [pauseStatusItem.button setAttributedTitle:title];
}

static void PauseStatusBarAdjustLengthForTitle(NSString *text) {
    if (pauseStatusItem == nil || pauseStatusItem.button == nil) {
        return;
    }
    BOOL hasText = text != nil && [text length] > 0;
    if (pausePopover != nil && [pausePopover isShown]) {
        return;
    }
    if (pauseStatusLayoutHasTextKnown && pauseStatusLayoutHasText == hasText) {
        return;
    }
    pauseStatusLayoutHasText = hasText;
    pauseStatusLayoutHasTextKnown = YES;
    [pauseStatusItem setLength:(hasText ? pauseStatusItemWidthWithTime : pauseStatusItemWidthIconOnly)];
    [pauseStatusItem.button setImagePosition:(hasText ? NSImageLeft : NSImageOnly)];
}

static void PauseStatusBarSetPausedBlinking(BOOL paused) {
    BOOL wasPaused = pauseStatusPaused;
    BOOL hadBlinkTimer = (pauseStatusBlinkTimer != nil);
    pauseStatusPaused = paused;
    if (!paused) {
        if (hadBlinkTimer) {
            [pauseStatusBlinkTimer invalidate];
            pauseStatusBlinkTimer = nil;
        }
        if (!wasPaused && !hadBlinkTimer && pauseStatusBlinkVisible) {
            return;
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
    if (pauseGlobalToggleButton != nil) {
        NSString *pauseTitle = pauseGlobalPauseButtonTitle ? pauseGlobalPauseButtonTitle : @"Pause";
        NSString *resumeTitle = pauseGlobalResumeButtonTitle ? pauseGlobalResumeButtonTitle : @"Resume";
        if (paused) {
            [pauseGlobalToggleButton setTitle:resumeTitle];
            [pauseGlobalToggleButton setTag:PauseStatusBarActionResume];
            [pauseGlobalToggleButton setToolTip:resumeTitle];
        } else {
            [pauseGlobalToggleButton setTitle:pauseTitle];
            [pauseGlobalToggleButton setTag:PauseStatusBarActionPause];
            [pauseGlobalToggleButton setToolTip:pauseTitle];
        }
    }
    if (pauseBreakNowButton != nil) {
        [pauseBreakNowButton setHidden:NO];
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
    BOOL wasShown = NO;
    if (pausePopover != nil && [pausePopover isShown]) {
        wasShown = YES;
        [pausePopover close];
    }
    PauseRemovePopoverAutoClose();
    PauseUpdateStatusItemTooltipVisibility();
    if (wasShown) {
        PauseStatusBarAdjustLengthForTitle(pauseStatusCurrentTitle);
        statusBarPopoverVisibilityCallbackGo(0);
    }
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

static NSString *PauseNoRemindersText(void) {
    return @"暂无提醒";
}

static void PauseUpdatePopoverSize(void) {
    if (pausePopover == nil || pausePopoverContentView == nil) {
        return;
    }
    [pausePopoverContentView layoutSubtreeIfNeeded];
    NSSize fitting = [pausePopoverContentView fittingSize];
    CGFloat height = fitting.height;
    if (height < 190.0) {
        height = 190.0;
    }
    if (height > 380.0) {
        height = 380.0;
    }
    [pausePopover setContentSize:NSMakeSize(300.0, height)];
}

static void PauseClearReminderRows(void) {
    if (pauseRemindersStack != nil) {
        NSArray *existingRows = [[pauseRemindersStack arrangedSubviews] copy];
        for (NSView *row in existingRows) {
            [pauseRemindersStack removeArrangedSubview:row];
            [row removeFromSuperview];
        }
        [existingRows release];
    }
    [pauseReminderPauseButtons removeAllObjects];
    [pauseReminderResumeButtons removeAllObjects];
    [pauseReminderProgressBars removeAllObjects];
}

static int PauseActionForReminder(NSInteger rowIndex, BOOL paused) {
    if (rowIndex < 0) {
        return (paused ? PauseStatusBarActionResume : PauseStatusBarActionPause);
    }
    if (paused) {
        return (int)(PauseStatusBarActionResumeReminderBase + rowIndex);
    }
    return (int)(PauseStatusBarActionPauseReminderBase + rowIndex);
}

static void PauseAppendReminderRow(NSInteger rowIndex, BOOL paused, NSString *titleText, double progress) {
    if (pauseRemindersStack == nil) {
        return;
    }

    NSString *safeTitle = titleText ? titleText : @"";
    NSStackView *reminderStack = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [reminderStack setOrientation:NSUserInterfaceLayoutOrientationVertical];
    [reminderStack setAlignment:NSLayoutAttributeCenterX];
    [reminderStack setSpacing:6.0];
    [reminderStack setTranslatesAutoresizingMaskIntoConstraints:NO];

    NSTextField *titleLabel = MakeLabel(safeTitle, [NSFont monospacedDigitSystemFontOfSize:13 weight:NSFontWeightMedium], [NSColor labelColor]);
    [titleLabel setAlignment:NSTextAlignmentCenter];
    [titleLabel setLineBreakMode:NSLineBreakByTruncatingTail];

    NSStackView *rowControls = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [rowControls setOrientation:NSUserInterfaceLayoutOrientationHorizontal];
    [rowControls setDistribution:NSStackViewDistributionFill];
    [rowControls setAlignment:NSLayoutAttributeCenterY];
    [rowControls setSpacing:8.0];
    [rowControls setTranslatesAutoresizingMaskIntoConstraints:NO];

    NSString *pauseTip = pausePauseButtonTooltip ? pausePauseButtonTooltip : @"Pause";
    NSString *resumeTip = pauseResumeButtonTooltip ? pauseResumeButtonTooltip : @"Resume";
    int pauseActionID = PauseActionForReminder(rowIndex, NO);
    int resumeActionID = PauseActionForReminder(rowIndex, YES);
    NSButton *pauseButton = MakeIconActionButton(@"pause.fill", pauseTip, pauseActionID);
    NSButton *resumeButton = MakeIconActionButton(@"play.fill", resumeTip, resumeActionID);
    NSProgressIndicator *progressView = [[NSProgressIndicator alloc] initWithFrame:NSZeroRect];
    [progressView setIndeterminate:NO];
    [progressView setMinValue:0];
    [progressView setMaxValue:100];
    [progressView setControlSize:NSControlSizeSmall];
    [progressView setStyle:NSProgressIndicatorStyleBar];
    [progressView setTranslatesAutoresizingMaskIntoConstraints:NO];

    double clamped = progress;
    if (clamped < 0) {
        clamped = 0;
    }
    if (clamped > 1) {
        clamped = 1;
    }
    [progressView setDoubleValue:(clamped * 100.0)];

    [rowControls addArrangedSubview:pauseButton];
    [rowControls addArrangedSubview:resumeButton];
    [rowControls addArrangedSubview:progressView];

    [reminderStack addArrangedSubview:titleLabel];
    [reminderStack addArrangedSubview:rowControls];
    [pauseRemindersStack addArrangedSubview:reminderStack];

    [pauseReminderPauseButtons addObject:pauseButton];
    [pauseReminderResumeButtons addObject:resumeButton];
    [pauseReminderProgressBars addObject:progressView];

    [NSLayoutConstraint activateConstraints:@[
        [reminderStack.leadingAnchor constraintEqualToAnchor:pauseRemindersStack.leadingAnchor],
        [reminderStack.trailingAnchor constraintEqualToAnchor:pauseRemindersStack.trailingAnchor],
        [titleLabel.leadingAnchor constraintEqualToAnchor:reminderStack.leadingAnchor],
        [titleLabel.trailingAnchor constraintEqualToAnchor:reminderStack.trailingAnchor],
        [rowControls.leadingAnchor constraintEqualToAnchor:reminderStack.leadingAnchor],
        [rowControls.trailingAnchor constraintEqualToAnchor:reminderStack.trailingAnchor],
        [pauseButton.widthAnchor constraintEqualToConstant:32],
        [resumeButton.widthAnchor constraintEqualToConstant:32],
        [progressView.heightAnchor constraintEqualToConstant:8],
        [rowControls.heightAnchor constraintEqualToConstant:28],
    ]];

    [pauseButton setHidden:paused];
    [resumeButton setHidden:!paused];

    [progressView release];
    [rowControls release];
    [reminderStack release];
}

static void PauseAppendNoReminderRow(NSString *titleText) {
    if (pauseRemindersStack == nil) {
        return;
    }

    NSString *safeTitle = titleText ? titleText : @"";
    NSStackView *reminderStack = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [reminderStack setOrientation:NSUserInterfaceLayoutOrientationVertical];
    [reminderStack setAlignment:NSLayoutAttributeCenterX];
    [reminderStack setSpacing:0.0];
    [reminderStack setTranslatesAutoresizingMaskIntoConstraints:NO];

    NSTextField *titleLabel = MakeLabel(safeTitle, [NSFont systemFontOfSize:13 weight:NSFontWeightRegular], [NSColor secondaryLabelColor]);
    [titleLabel setAlignment:NSTextAlignmentCenter];
    [titleLabel setLineBreakMode:NSLineBreakByTruncatingTail];

    [reminderStack addArrangedSubview:titleLabel];
    [pauseRemindersStack addArrangedSubview:reminderStack];

    [NSLayoutConstraint activateConstraints:@[
        [reminderStack.leadingAnchor constraintEqualToAnchor:pauseRemindersStack.leadingAnchor],
        [reminderStack.trailingAnchor constraintEqualToAnchor:pauseRemindersStack.trailingAnchor],
        [titleLabel.leadingAnchor constraintEqualToAnchor:reminderStack.leadingAnchor],
        [titleLabel.trailingAnchor constraintEqualToAnchor:reminderStack.trailingAnchor],
    ]];

    [reminderStack release];
}

static void PauseRebuildReminderRows(NSString *remindersPayload) {
    PauseClearReminderRows();

    NSData *payloadData = nil;
    if (remindersPayload != nil && [remindersPayload length] > 0) {
        payloadData = [remindersPayload dataUsingEncoding:NSUTF8StringEncoding];
    }
    NSArray *items = nil;
    if (payloadData != nil) {
        NSError *error = nil;
        id parsed = [NSJSONSerialization JSONObjectWithData:payloadData options:0 error:&error];
        if (error == nil && [parsed isKindOfClass:[NSArray class]]) {
            items = (NSArray *)parsed;
        }
    }

    if (items == nil || [items count] == 0) {
        PauseAppendNoReminderRow(PauseNoRemindersText());
        PauseUpdatePopoverSize();
        return;
    }

    NSInteger appended = 0;
    for (id item in items) {
        if (![item isKindOfClass:[NSDictionary class]]) {
            continue;
        }
        NSDictionary *entry = (NSDictionary *)item;
        NSString *title = @"";
        id titleValue = [entry objectForKey:@"title"];
        if ([titleValue isKindOfClass:[NSString class]]) {
            title = (NSString *)titleValue;
        }

        double progress = 0;
        id progressValue = [entry objectForKey:@"progress"];
        if ([progressValue respondsToSelector:@selector(doubleValue)]) {
            progress = [progressValue doubleValue];
        }
        BOOL paused = NO;
        id pausedValue = [entry objectForKey:@"paused"];
        if ([pausedValue respondsToSelector:@selector(boolValue)]) {
            paused = [pausedValue boolValue];
        }
        PauseAppendReminderRow(appended, paused, title, progress);
        appended += 1;
    }
    if (appended == 0) {
        PauseAppendNoReminderRow(PauseNoRemindersText());
    }
    PauseUpdatePopoverSize();
}

static void PauseSetLatestRemindersPayload(NSString *remindersPayload) {
    NSString *safePayload = remindersPayload ? remindersPayload : @"";
    if (pauseLatestRemindersPayload != nil && [pauseLatestRemindersPayload isEqualToString:safePayload]) {
        return;
    }
    if (pauseLatestRemindersPayload != nil) {
        [pauseLatestRemindersPayload release];
        pauseLatestRemindersPayload = nil;
    }
    pauseLatestRemindersPayload = [safePayload copy];
}

static void PauseRefreshReminderRowsIfNeeded(void) {
    if (pauseRemindersStack == nil) {
        return;
    }
    NSString *latestPayload = pauseLatestRemindersPayload ? pauseLatestRemindersPayload : @"";
    if (pauseRenderedRemindersPayload != nil && [pauseRenderedRemindersPayload isEqualToString:latestPayload]) {
        return;
    }
    PauseRebuildReminderRows(latestPayload);
    if (pauseRenderedRemindersPayload != nil) {
        [pauseRenderedRemindersPayload release];
        pauseRenderedRemindersPayload = nil;
    }
    pauseRenderedRemindersPayload = [latestPayload copy];
}

static void BuildPopoverContent(void) {
    NSView *contentView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 300, 250)];
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
    pauseRemindersStack = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [pauseRemindersStack setOrientation:NSUserInterfaceLayoutOrientationVertical];
    [pauseRemindersStack setAlignment:NSLayoutAttributeLeading];
    [pauseRemindersStack setSpacing:10.0];
    [pauseRemindersStack setTranslatesAutoresizingMaskIntoConstraints:NO];
    pauseReminderPauseButtons = [[NSMutableArray alloc] init];
    pauseReminderResumeButtons = [[NSMutableArray alloc] init];
    pauseReminderProgressBars = [[NSMutableArray alloc] init];
    PauseSetLatestRemindersPayload(@"");
    PauseRefreshReminderRowsIfNeeded();

    pauseGlobalToggleButton = MakeActionButton(@"Pause", PauseStatusBarActionPause);
    pauseBreakNowButton = MakeActionButton(@"zzZ", PauseStatusBarActionBreakNow);
    NSStackView *breakNowButtonsRow = [[NSStackView alloc] initWithFrame:NSZeroRect];
    [breakNowButtonsRow setOrientation:NSUserInterfaceLayoutOrientationHorizontal];
    [breakNowButtonsRow setAlignment:NSLayoutAttributeCenterY];
    [breakNowButtonsRow setSpacing:8.0];
    [breakNowButtonsRow setTranslatesAutoresizingMaskIntoConstraints:NO];
    [breakNowButtonsRow addArrangedSubview:pauseGlobalToggleButton];
    [breakNowButtonsRow addArrangedSubview:pauseBreakNowButton];

    NSView *breakNowContainer = [[NSView alloc] initWithFrame:NSZeroRect];
    [breakNowContainer setTranslatesAutoresizingMaskIntoConstraints:NO];
    [breakNowContainer addSubview:breakNowButtonsRow];
    [NSLayoutConstraint activateConstraints:@[
        [breakNowButtonsRow.centerXAnchor constraintEqualToAnchor:breakNowContainer.centerXAnchor],
        [breakNowButtonsRow.topAnchor constraintEqualToAnchor:breakNowContainer.topAnchor],
        [breakNowButtonsRow.bottomAnchor constraintEqualToAnchor:breakNowContainer.bottomAnchor],
        [breakNowButtonsRow.leadingAnchor constraintGreaterThanOrEqualToAnchor:breakNowContainer.leadingAnchor],
        [breakNowButtonsRow.trailingAnchor constraintLessThanOrEqualToAnchor:breakNowContainer.trailingAnchor],
        [pauseGlobalToggleButton.heightAnchor constraintEqualToConstant:28],
        [pauseBreakNowButton.heightAnchor constraintEqualToConstant:28],
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
    [stack addArrangedSubview:pauseRemindersStack];
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
        [pauseRemindersStack.leadingAnchor constraintEqualToAnchor:stack.leadingAnchor],
        [pauseRemindersStack.trailingAnchor constraintEqualToAnchor:stack.trailingAnchor],
        [breakNowContainer.heightAnchor constraintEqualToConstant:28],
        [openButtonContainer.widthAnchor constraintEqualToAnchor:stack.widthAnchor],
        [openButtonContainer.heightAnchor constraintEqualToConstant:28],
    ]];

    pausePopoverController = [[NSViewController alloc] init];
    [pausePopoverController setView:contentView];
    pausePopoverContentView = [contentView retain];
    [contentView release];

    pausePopover = [[NSPopover alloc] init];
    [pausePopover setBehavior:NSPopoverBehaviorTransient];
    [pausePopover setContentSize:NSMakeSize(300, 250)];
    [pausePopover setContentViewController:pausePopoverController];
    PauseSetActionButtonsPausedState(NO);
    PauseUpdatePopoverSize();

    [breakNowButtonsRow release];
    [openButtonContainer release];
    [breakNowContainer release];
    [headerView release];
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
        [pauseStatusItem.button setFont:[NSFont monospacedDigitSystemFontOfSize:13 weight:NSFontWeightRegular]];
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

void PauseStatusBarUpdate(const char *status, const char *countdown, const char *title, int paused, double progress, const char *remindersPayload) {
    NSString *statusText = status ? [NSString stringWithUTF8String:status] : @"Status: running";
    NSString *countdownText = countdown ? [NSString stringWithUTF8String:countdown] : @"Next break: --:--";
    NSString *titleText = title ? [NSString stringWithUTF8String:title] : @"";
    NSString *remindersText = remindersPayload ? [NSString stringWithUTF8String:remindersPayload] : @"";

    PauseRunOnMain(^{
        if (pauseStatusItem == nil) {
            return;
        }
        PauseStatusBarSetTitleText(titleText);
        PauseStatusBarSetPausedBlinking(paused != 0);
        if (pauseStatusLabel != nil) {
            [pauseStatusLabel setStringValue:statusText];
        }
        (void)countdownText;
        (void)progress;
        PauseSetLatestRemindersPayload(remindersText);
        if (pausePopover != nil && [pausePopover isShown]) {
            PauseRefreshReminderRowsIfNeeded();
        }
        PauseSetActionButtonsPausedState(paused != 0);
    });
}

void PauseStatusBarSetLocaleStrings(
    const char *popoverTitle,
    const char *breakNowButton,
    const char *pauseButton,
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
        if (pauseBreakNowButton != nil) {
            [pauseBreakNowButton setTitle:breakNowButtonText];
        }
        if (pauseGlobalPauseButtonTitle != nil) {
            [pauseGlobalPauseButtonTitle release];
            pauseGlobalPauseButtonTitle = nil;
        }
        pauseGlobalPauseButtonTitle = [pauseButtonText copy];
        if (pauseGlobalResumeButtonTitle != nil) {
            [pauseGlobalResumeButtonTitle release];
            pauseGlobalResumeButtonTitle = nil;
        }
        pauseGlobalResumeButtonTitle = [resumeButtonText copy];
        PauseSetActionButtonsPausedState(pauseStatusPaused);
        if (pausePauseButtonTooltip != nil) {
            [pausePauseButtonTooltip release];
            pausePauseButtonTooltip = nil;
        }
        pausePauseButtonTooltip = [pauseButtonText copy];
        if (pauseResumeButtonTooltip != nil) {
            [pauseResumeButtonTooltip release];
            pauseResumeButtonTooltip = nil;
        }
        pauseResumeButtonTooltip = [resumeButtonText copy];
        for (NSButton *pauseButton in pauseReminderPauseButtons) {
            [pauseButton setTitle:@""];
            [pauseButton setToolTip:pauseButtonText];
        }
        for (NSButton *resumeButton in pauseReminderResumeButtons) {
            [resumeButton setTitle:@""];
            [resumeButton setToolTip:resumeButtonText];
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
        if (pausePopoverContentView != nil) {
            [pausePopoverContentView release];
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
        if (pausePauseButtonTooltip != nil) {
            [pausePauseButtonTooltip release];
        }
        if (pauseResumeButtonTooltip != nil) {
            [pauseResumeButtonTooltip release];
        }
        if (pauseGlobalPauseButtonTitle != nil) {
            [pauseGlobalPauseButtonTitle release];
        }
        if (pauseGlobalResumeButtonTitle != nil) {
            [pauseGlobalResumeButtonTitle release];
        }
        if (pauseLatestRemindersPayload != nil) {
            [pauseLatestRemindersPayload release];
        }
        if (pauseRenderedRemindersPayload != nil) {
            [pauseRenderedRemindersPayload release];
        }
        if (pauseStatusCurrentTitle != nil) {
            [pauseStatusCurrentTitle release];
        }
        if (pauseReminderPauseButtons != nil) {
            [pauseReminderPauseButtons release];
        }
        if (pauseReminderResumeButtons != nil) {
            [pauseReminderResumeButtons release];
        }
        if (pauseReminderProgressBars != nil) {
            [pauseReminderProgressBars release];
        }
        if (pauseRemindersStack != nil) {
            [pauseRemindersStack release];
        }

        pauseStatusItem = nil;
        pausePopover = nil;
        pausePopoverController = nil;
        pausePopoverContentView = nil;
        pausePopoverTitleLabel = nil;
        pauseStatusLabel = nil;
        pauseRemindersStack = nil;
        pauseReminderPauseButtons = nil;
        pauseReminderResumeButtons = nil;
        pauseReminderProgressBars = nil;
        pauseGlobalToggleButton = nil;
        pauseBreakNowButton = nil;
        pauseOpenButton = nil;
        pauseMoreButton = nil;
        pauseAboutMenuTitle = nil;
        pauseQuitMenuTitle = nil;
        pauseMoreButtonTip = nil;
        pauseStatusTooltipText = nil;
        pausePauseButtonTooltip = nil;
        pauseResumeButtonTooltip = nil;
        pauseGlobalPauseButtonTitle = nil;
        pauseGlobalResumeButtonTitle = nil;
        pauseLatestRemindersPayload = nil;
        pauseRenderedRemindersPayload = nil;
        pauseStatusCurrentTitle = nil;
        pauseLocalMonitor = nil;
        pauseGlobalMonitor = nil;
        pauseStatusHandler = nil;
        pauseStatusIcon = nil;
        pauseStatusIconAlt = nil;
        pauseStatusBlinkTimer = nil;
        pauseStatusBlinkVisible = YES;
        pauseStatusPaused = NO;
        pauseStatusLayoutHasText = NO;
        pauseStatusLayoutHasTextKnown = NO;
    });
}
*/
import "C"
