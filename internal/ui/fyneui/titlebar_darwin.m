// Cogpit's title bar on macOS: the AppKit side of titlebar_darwin.go. Compiled without ARC, like
// the rest of the package's Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <string.h>

#include "titlebar_darwin.h"

extern void cogpitTitlebarChanged(uintptr_t id);

// CogpitTitlebarObserver forwards a window's layout notifications to Go.
@interface CogpitTitlebarObserver : NSObject
@property(nonatomic) uintptr_t titlebarID;
@end

@implementation CogpitTitlebarObserver
- (void)changed:(NSNotification *)note {
	cogpitTitlebarChanged(self.titlebarID);
}
@end

// The observer lives as long as its window: the window holds it as an associated object.
static char observerKey;

static NSWindow *windowOf(uintptr_t window) { return (NSWindow *)(void *)window; }

void cogpitTitlebarInstall(uintptr_t window, uintptr_t id) {
	NSWindow *w = windowOf(window);
	w.styleMask |= NSWindowStyleMaskFullSizeContentView;
	w.titlebarAppearsTransparent = YES;
	w.titleVisibility = NSWindowTitleHidden;

	CogpitTitlebarObserver *o = [[CogpitTitlebarObserver alloc] init];
	o.titlebarID = id;
	NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
	NSArray<NSNotificationName> *names = @[
		NSWindowDidEnterFullScreenNotification, NSWindowDidExitFullScreenNotification,
		NSWindowDidResizeNotification, NSWindowDidChangeBackingPropertiesNotification,
		NSWindowDidChangeScreenNotification,
	];
	for (NSNotificationName name in names) {
		[nc addObserver:o selector:@selector(changed:) name:name object:w];
	}
	objc_setAssociatedObject(w, &observerKey, o, OBJC_ASSOCIATION_RETAIN);
	[o release];

	// The content view now covers the title bar's area too, without a resize event: announce one,
	// so that GLFW's window delegate (which observes the window's notifications) reads the larger
	// content size and Fyne lays the canvas out to it, and the observer above reads the geometry.
	[nc postNotificationName:NSWindowDidResizeNotification object:w];
}

void cogpitTitlebarRemove(uintptr_t window) {
	NSWindow *w = windowOf(window);
	id o = objc_getAssociatedObject(w, &observerKey);
	if (o != nil) {
		[[NSNotificationCenter defaultCenter] removeObserver:o];
		objc_setAssociatedObject(w, &observerKey, nil, OBJC_ASSOCIATION_RETAIN);
	}
}

void cogpitTitlebarGeometry(uintptr_t window, cogpitTitlebarGeom *geom) {
	NSWindow *w = windowOf(window);
	memset(geom, 0, sizeof(*geom));
	geom->fullScreen = (w.styleMask & NSWindowStyleMaskFullScreen) != 0;
	NSButton *close = [w standardWindowButton:NSWindowCloseButton];
	NSButton *zoom = [w standardWindowButton:NSWindowZoomButton];
	if (close == nil || zoom == nil) {
		return;
	}
	// The buttons' frames in the window's coordinates, whose origin is at the bottom left; x is the
	// same from the top left.
	NSRect c = [close convertRect:close.bounds toView:nil];
	NSRect z = [zoom convertRect:zoom.bounds toView:nil];
	geom->buttonsMinX = NSMinX(c);
	geom->buttonsMaxX = NSMaxX(z);
	geom->height = NSHeight(w.frame) - NSHeight(w.contentLayoutRect);
	geom->ok = 1;
}

static NSEvent *leftMouseDownIn(NSWindow *w) {
	NSEvent *e = [NSApp currentEvent];
	if (e == nil || e.type != NSEventTypeLeftMouseDown || e.window != w) {
		return nil;
	}
	return e;
}

int cogpitTitlebarClickCount(uintptr_t window) {
	NSEvent *e = leftMouseDownIn(windowOf(window));
	return e == nil ? 0 : (int)e.clickCount;
}

void cogpitTitlebarDrag(uintptr_t window) {
	NSWindow *w = windowOf(window);
	NSEvent *e = leftMouseDownIn(w);
	if (e == nil) {
		return;
	}
	[w performWindowDragWithEvent:e];
	// The window server moves the window and keeps the mouse-up that ends the drag: post one, so
	// that GLFW and Fyne, which saw the mouse-down, do not keep the button pressed and take the
	// next move for a drag.
	NSEvent *up = [NSEvent mouseEventWithType:NSEventTypeLeftMouseUp
	                                 location:e.locationInWindow
	                            modifierFlags:e.modifierFlags
	                                timestamp:e.timestamp
	                             windowNumber:e.windowNumber
	                                  context:nil
	                              eventNumber:e.eventNumber
	                               clickCount:e.clickCount
	                                 pressure:0];
	[NSApp postEvent:up atStart:NO];
}

void cogpitTitlebarDoubleClickSetting(char *action, int size, int *legacySet, int *legacyMinimize) {
	NSUserDefaults *d = [NSUserDefaults standardUserDefaults];
	NSString *a = [d stringForKey:@"AppleActionOnDoubleClick"];
	if (size > 0) {
		action[0] = 0;
		if (a != nil) {
			strlcpy(action, a.UTF8String, (size_t)size);
		}
	}
	id legacy = [d objectForKey:@"AppleMiniaturizeOnDoubleClick"];
	*legacySet = legacy != nil;
	*legacyMinimize = legacy != nil && [d boolForKey:@"AppleMiniaturizeOnDoubleClick"];
}

void cogpitTitlebarZoom(uintptr_t window) { [windowOf(window) zoom:nil]; }

void cogpitTitlebarMiniaturize(uintptr_t window) { [windowOf(window) miniaturize:nil]; }
