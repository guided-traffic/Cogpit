// A Secret's value on the clipboard, marked for clipboard managers and kept to this Mac: the
// AppKit side of clipboard_darwin.go. Compiled without ARC, like the rest of the package's
// Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>

#include "clipboard_darwin.h"

// cogpitSetConcealedString replaces the general pasteboard's contents with text as a string,
// marked with nspasteboard.org's concealed and transient types, which clipboard managers honour
// by not showing or recording it, and for the current host only, which keeps it out of Universal
// Clipboard. It returns false when the string could not be set.
bool cogpitSetConcealedString(const char *text) {
	@autoreleasepool {
		NSString *s = [NSString stringWithUTF8String:text];
		if (s == nil) {
			return false;
		}
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		[pb prepareForNewContentsWithOptions:NSPasteboardContentsCurrentHostOnly];
		if (![pb setString:s forType:NSPasteboardTypeString]) {
			return false;
		}
		[pb setString:@"" forType:@"org.nspasteboard.ConcealedType"];
		[pb setString:@"" forType:@"org.nspasteboard.TransientType"];
		return true;
	}
}
