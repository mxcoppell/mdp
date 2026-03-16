//go:build darwin

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework UniformTypeIdentifiers

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

static void applyFrameless(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;

    nsWindow.styleMask |= NSWindowStyleMaskFullSizeContentView;
    nsWindow.titlebarAppearsTransparent = YES;
    nsWindow.titleVisibility = NSWindowTitleHidden;
    nsWindow.title = @"";

    [[nsWindow standardWindowButton:NSWindowCloseButton] setHidden:YES];
    [[nsWindow standardWindowButton:NSWindowMiniaturizeButton] setHidden:YES];
    [[nsWindow standardWindowButton:NSWindowZoomButton] setHidden:YES];

    [nsWindow setHasShadow:YES];
    [nsWindow setBackgroundColor:[NSColor clearColor]];
    nsWindow.contentView.wantsLayer = YES;
    nsWindow.contentView.layer.cornerRadius = 10;
    nsWindow.contentView.layer.masksToBounds = YES;
    [nsWindow setMovableByWindowBackground:NO];
}

// Minimal delegate for legacy single-window accessory mode (gui.Run path).
@interface AccessoryDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AccessoryDelegate
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return NO;
}
@end

void guiInitAccessoryMode(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    [NSApp setDelegate:[[AccessoryDelegate alloc] init]];
}

void guiHideWindowOffscreen(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow setAlphaValue:0];
}

void guiApplyFramelessDirect(void *window) {
    applyFrameless(window);
}

// Track cascade point across windows so each new one offsets from the last.
static NSPoint _cascadePoint = {0, 0};

void guiShowWindow(void *window, int width, int height) {
    NSWindow *nsWindow = (NSWindow *)window;

    applyFrameless(window);

    if (width > 0 && height > 0) {
        NSRect frame = [nsWindow frame];
        frame.size = NSMakeSize(width, height);
        [nsWindow setFrame:frame display:NO];
    }

    // First window centers; subsequent windows cascade from the previous.
    if (_cascadePoint.x == 0 && _cascadePoint.y == 0) {
        [nsWindow center];
    }
    _cascadePoint = [nsWindow cascadeTopLeftFromPoint:_cascadePoint];

    [nsWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    [nsWindow setLevel:NSFloatingWindowLevel];
    [nsWindow setLevel:NSNormalWindowLevel];

    [NSAnimationContext runAnimationGroup:^(NSAnimationContext *ctx) {
        ctx.duration = 0.15;
        ctx.timingFunction = [CAMediaTimingFunction functionWithName:kCAMediaTimingFunctionEaseOut];
        [[nsWindow animator] setAlphaValue:1.0];
    }];
}

void guiCenterWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow center];
}

void guiMoveWindowBy(void *window, int dx, int dy) {
    NSWindow *nsWindow = (NSWindow *)window;
    NSRect frame = nsWindow.frame;
    frame.origin.x += dx;
    frame.origin.y -= dy;
    [nsWindow setFrameOrigin:frame.origin];
}

void guiResizeWindowBy(void *window, int dw, int dh, int shiftX) {
    NSWindow *nsWindow = (NSWindow *)window;

    NSAppearanceName best = [nsWindow.effectiveAppearance
        bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
    BOOL isDark = [best isEqualToString:NSAppearanceNameDarkAqua];
    CGColorRef bg = CGColorCreateGenericRGB(
        isDark ? 0.133 : 0.973,
        isDark ? 0.133 : 0.973,
        isDark ? 0.133 : 0.973, 1.0);
    nsWindow.contentView.layer.backgroundColor = bg;
    CGColorRelease(bg);

    NSRect frame = nsWindow.frame;
    frame.origin.x += shiftX;
    frame.size.width += dw;
    frame.size.height += dh;
    NSRect screen = [[nsWindow screen] visibleFrame];
    if (frame.origin.x < screen.origin.x) frame.origin.x = screen.origin.x;
    if (frame.size.width > screen.size.width) frame.size.width = screen.size.width;
    [nsWindow setFrame:frame display:YES animate:NO];

    dispatch_after(
        dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.1 * NSEC_PER_SEC)),
        dispatch_get_main_queue(), ^{
            nsWindow.contentView.layer.backgroundColor = NULL;
        });
}

void guiActivateWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

// Close an NSWindow directly without going through webview's destructor.
// webview.Destroy() calls deplete_run_loop_event_queue() which deadlocks
// when called from within a GCD main queue block (the probe it posts to the
// serial GCD queue can't fire while the current block is still executing).
// This bypasses that by just closing the native window.
void guiCloseWindow(void *window) {
    NSWindow *nsWindow = (NSWindow *)window;
    [nsWindow setDelegate:nil];
    [nsWindow close];
}

// Legacy: schedule frameless via timer (used by gui.Run single-window path)
static int _frameless_applied = 0;
static void *_pending_frameless_window = NULL;

static void framelessTimerCallback(CFRunLoopTimerRef timer, void *info) {
    if (!_pending_frameless_window) return;
    applyFrameless(_pending_frameless_window);
    _frameless_applied = 1;
    CFRunLoopTimerInvalidate(timer);
}

void guiScheduleFrameless(void *window) {
    _pending_frameless_window = window;
    _frameless_applied = 0;

    CFRunLoopTimerContext ctx = {0, NULL, NULL, NULL, NULL};
    CFRunLoopTimerRef timer = CFRunLoopTimerCreate(
        kCFAllocatorDefault,
        CFAbsoluteTimeGetCurrent() + 0.05,
        0,
        0, 0,
        framelessTimerCallback,
        &ctx
    );
    CFRunLoopAddTimer(CFRunLoopGetMain(), timer, kCFRunLoopCommonModes);
    CFRelease(timer);
}
*/
import "C"

import "unsafe"

func initAccessoryMode() {
	C.guiInitAccessoryMode()
}

func hideWindowOffscreen(windowHandle unsafe.Pointer) {
	C.guiHideWindowOffscreen(windowHandle)
}

func applyFramelessDirect(windowHandle unsafe.Pointer) {
	C.guiApplyFramelessDirect(windowHandle)
}

func scheduleFrameless(windowHandle unsafe.Pointer) {
	C.guiScheduleFrameless(windowHandle)
}

func showWindow(windowHandle unsafe.Pointer, width, height int) {
	C.guiShowWindow(windowHandle, C.int(width), C.int(height))
}

func centerWindow(windowHandle unsafe.Pointer) {
	C.guiCenterWindow(windowHandle)
}

func moveWindowBy(windowHandle unsafe.Pointer, dx, dy int) {
	C.guiMoveWindowBy(windowHandle, C.int(dx), C.int(dy))
}

func resizeWindowBy(windowHandle unsafe.Pointer, dw, dh, shiftX int) {
	C.guiResizeWindowBy(windowHandle, C.int(dw), C.int(dh), C.int(shiftX))
}

func activateWindow(windowHandle unsafe.Pointer) {
	C.guiActivateWindow(windowHandle)
}

func closeWindow(windowHandle unsafe.Pointer) {
	C.guiCloseWindow(windowHandle)
}
