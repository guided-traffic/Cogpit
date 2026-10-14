// A native check of appmenu_darwin.m, built and run by TestAppMenuAbout in a process of its own (the
// menu bar belongs to the process's NSApplication): it makes a menu bar as GLFW's createMenuBar does
// (an application menu whose first item, "About <program name>", opens AppKit's standard panel),
// installs Cogpit's About, rebuilds the menu bar's items the way Fyne's resetDarwinMenu does (the
// application menu's items tagged 5000 and above removed), installs again, chooses the item, and
// installs into an application menu without an About item. One line per step:
// "<step>|<first item's title>|<About items>|<items>|<first opens the standard panel>|<chosen>".

#import <Cocoa/Cocoa.h>
#include "appmenu_darwin.h"

static int chosen;

void cogpitAboutChosen(void) { chosen++; }

static NSMenu *appMenu(void) { return [[[NSApp mainMenu] itemAtIndex:0] submenu]; }

static void report(const char *step) {
	NSMenu *m = appMenu();
	NSMenuItem *first = [m itemAtIndex:0];
	int abouts = 0;
	for (NSMenuItem *item in [m itemArray]) {
		if ([[item title] hasPrefix:@"About"]) {
			abouts++;
		}
	}
	printf("%s|%s|%d|%ld|%d|%d\n", step, [[first title] UTF8String], abouts, (long)[m numberOfItems],
	       [first action] == @selector(orderFrontStandardAboutPanel:), chosen);
}

// menuBar makes a menu bar whose application menu has GLFW's items, About first when about is set.
static void menuBar(BOOL about) {
	NSMenu *bar = [[NSMenu alloc] init];
	NSMenuItem *appItem = [bar addItemWithTitle:@"" action:NULL keyEquivalent:@""];
	NSMenu *app = [[NSMenu alloc] init];
	[appItem setSubmenu:app];
	if (about) {
		[app addItemWithTitle:@"About cogpit" action:@selector(orderFrontStandardAboutPanel:) keyEquivalent:@""];
		[app addItem:[NSMenuItem separatorItem]];
	}
	[app addItemWithTitle:@"Services" action:NULL keyEquivalent:@""];
	[app addItem:[NSMenuItem separatorItem]];
	[app addItemWithTitle:@"Hide cogpit" action:@selector(hide:) keyEquivalent:@"h"];
	[app addItemWithTitle:@"Quit cogpit" action:@selector(terminate:) keyEquivalent:@"q"];
	[app release];
	[NSApp setMainMenu:bar];
	[bar release];
}

// fyneItem adds an item as Fyne's handleSpecialItems does for "Preferences…": tagged 5000 and above,
// at index 1 of the application menu.
static void fyneItem(void) {
	NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:@"Preferences…" action:@selector(hide:) keyEquivalent:@","];
	[item setTag:5000];
	[appMenu() insertItem:item atIndex:1];
	[item release];
}

// fyneReset removes the application menu's items tagged 5000 and above, as Fyne's resetDarwinMenu
// does before it builds the menu bar again.
static void fyneReset(void) {
	NSMenu *m = appMenu();
	for (NSMenuItem *item in [[m itemArray] copy]) {
		if ([item tag] >= 5000) {
			[m removeItem:item];
		}
	}
}

int main(void) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setMainMenu:nil];
		printf("nomenu|%d\n", cogpitAppMenuInstallAbout("About Cogpit"));

		menuBar(YES);
		report("glfw");
		if (!cogpitAppMenuInstallAbout("About Cogpit")) {
			return 1;
		}
		report("installed");
		fyneItem();
		fyneReset();
		fyneItem();
		report("rebuilt");
		cogpitAppMenuInstallAbout("About Cogpit");
		report("again");
		NSMenuItem *first = [appMenu() itemAtIndex:0];
		[[first target] performSelector:[first action] withObject:first];
		report("chosen");

		menuBar(NO);
		cogpitAppMenuInstallAbout("About Cogpit");
		report("inserted");
	}
	return 0;
}
