// macOS's Reduce motion setting: the AppKit side of reducemotion_darwin.go. Compiled without ARC,
// like the rest of the package's Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>

#include "reducemotion_darwin.h"

int cogpitReduceMotion(void) {
	return [[NSWorkspace sharedWorkspace] accessibilityDisplayShouldReduceMotion] ? 1 : 0;
}
