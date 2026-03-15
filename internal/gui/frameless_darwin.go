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

static int _frameless_applied = 0;
static void *_pending_frameless_window = NULL;

static void framelessTimerCallback(CFRunLoopTimerRef timer, void *info) {
    if (!_pending_frameless_window) return;
    applyFrameless(_pending_frameless_window);
    _frameless_applied = 1;
    CFRunLoopTimerInvalidate(timer);
}

// Minimal delegate that keeps the app as an accessory (no dock icon).
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
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
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

void guiShowWindow(void *window, int width, int height) {
    NSWindow *nsWindow = (NSWindow *)window;

    if (!_frameless_applied) {
        applyFrameless(window);
        _frameless_applied = 1;
    }
    applyFrameless(window);

    if (width > 0 && height > 0) {
        NSRect frame = [nsWindow frame];
        frame.size = NSMakeSize(width, height);
        [nsWindow setFrame:frame display:NO];
    }

    [nsWindow center];

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

    // Fill new area with theme-matching background to prevent flash.
    // Without this, [NSColor clearColor] window background shows the desktop
    // through the newly exposed area before the webview re-renders.
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
    // Clamp to screen bounds
    NSRect screen = [[nsWindow screen] visibleFrame];
    if (frame.origin.x < screen.origin.x) frame.origin.x = screen.origin.x;
    if (frame.size.width > screen.size.width) frame.size.width = screen.size.width;
    [nsWindow setFrame:frame display:YES animate:NO];

    // Clear layer background after webview has rendered
    dispatch_after(
        dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.1 * NSEC_PER_SEC)),
        dispatch_get_main_queue(), ^{
            nsWindow.contentView.layer.backgroundColor = NULL;
        });
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
