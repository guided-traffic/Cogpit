// A native check of titlebar_darwin.m, built and run by TestTitlebarButtonsCentred (a window needs
// the main thread of its own process): it creates a window as GLFW does, installs Cogpit's title
// bar, centres the buttons in a 40-point bar, lets AppKit lay the title bar out again (a resize, a
// style change) and prints each step's button centres from the window's top and their x, one line
// per step: "<step> <close centre> <close x> <zoom centre> <zoom x>". Exit status 3: no window
// buttons (no window server).

#import <Cocoa/Cocoa.h>
#include "titlebar_darwin.h"

void cogpitTitlebarChanged(uintptr_t id) {}

static void report(NSWindow *w, const char *step) {
	NSButton *close = [w standardWindowButton:NSWindowCloseButton];
	NSButton *zoom = [w standardWindowButton:NSWindowZoomButton];
	NSRect c = [close convertRect:close.bounds toView:nil];
	NSRect z = [zoom convertRect:zoom.bounds toView:nil];
	CGFloat h = NSHeight(w.frame);
	printf("%s %.1f %.1f %.1f %.1f\n", step, h - NSMidY(c), NSMinX(c), h - NSMidY(z), NSMinX(z));
}

static void turn(void) { [[NSRunLoop mainRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.05]]; }

int main(void) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		NSWindow *w = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 800, 600)
		                                          styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
		                                                    NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable
		                                            backing:NSBackingStoreBuffered
		                                              defer:NO];
		w.releasedWhenClosed = NO;
		if ([w standardWindowButton:NSWindowCloseButton] == nil || [w standardWindowButton:NSWindowZoomButton] == nil) {
			return 3;
		}
		report(w, "appkit");
		cogpitTitlebarInstall((uintptr_t)w, 1);
		cogpitTitlebarCentreButtons((uintptr_t)w, 40);
		turn();
		report(w, "centred");
		[w setFrame:NSMakeRect(0, 0, 900, 700) display:NO];
		[w layoutIfNeeded];
		turn();
		report(w, "resized");
		w.styleMask &= ~NSWindowStyleMaskResizable;
		w.styleMask |= NSWindowStyleMaskResizable;
		[w layoutIfNeeded];
		turn();
		report(w, "restyled");
		[w close];
		turn();
		[w release];
	}
	return 0;
}
