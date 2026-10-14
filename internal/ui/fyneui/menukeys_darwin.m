// The window's own keys in macOS's main menu (A55): the AppKit side of menukeys_darwin.go. Compiled
// without ARC, like the rest of the package's Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>

#include "menukeys_darwin.h"

bool cogpitMenuKeyEvent(void) {
	if (![NSThread isMainThread] || NSApp == nil) {
		return false;
	}
	NSEvent *ev = [NSApp currentEvent];
	return ev != nil && [ev type] == NSEventTypeKeyDown;
}
