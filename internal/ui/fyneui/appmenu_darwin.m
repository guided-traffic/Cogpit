// Cogpit's About in macOS's application menu: the AppKit side of appmenu_darwin.go. Compiled
// without ARC, like the rest of the package's Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>

#include "appmenu_darwin.h"

extern void cogpitAboutChosen(void);

// CogpitAboutTarget receives the About item's action. A class, not an instance: nothing to keep
// alive, as Fyne's own menu handler.
@interface CogpitAboutTarget : NSObject
@end

@implementation CogpitAboutTarget
+ (void)aboutChosen:(id)sender {
	cogpitAboutChosen();
}
@end

// aboutItem returns the application menu's About item: GLFW's, which shows AppKit's standard panel,
// or this file's after an earlier call.
static NSMenuItem *aboutItem(NSMenu *appMenu) {
	for (NSMenuItem *item in [appMenu itemArray]) {
		if ([item action] == @selector(orderFrontStandardAboutPanel:) ||
		    ([item action] == @selector(aboutChosen:) && [item target] == (id)[CogpitAboutTarget class])) {
			return item;
		}
	}
	return nil;
}

bool cogpitAppMenuInstallAbout(const char *title) {
	NSMenu *bar = [NSApp mainMenu];
	if (bar == nil || [bar numberOfItems] == 0) {
		return false;
	}
	NSMenu *appMenu = [[bar itemAtIndex:0] submenu];
	if (appMenu == nil) {
		return false;
	}
	NSString *label = [NSString stringWithUTF8String:title];
	NSMenuItem *item = aboutItem(appMenu);
	if (item == nil) {
		item = [[NSMenuItem alloc] initWithTitle:label action:@selector(aboutChosen:) keyEquivalent:@""];
		[appMenu insertItem:item atIndex:0];
		[item release]; // the menu holds it
	}
	[item setTitle:label];
	[item setTarget:[CogpitAboutTarget class]];
	[item setAction:@selector(aboutChosen:)];
	// Fyne removes the application menu's items tagged 5000 and above, its own, whenever it builds the
	// menu bar again (menu_darwin.m, resetDarwinMenu). Below that, the item keeps its place.
	[item setTag:0];
	return true;
}
