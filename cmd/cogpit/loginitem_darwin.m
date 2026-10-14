// Start at login on macOS: the ServiceManagement side of loginitem_darwin.go. The app's own login
// item (SMAppService.mainAppService, macOS 13 or later, Cogpit's minimum), which System Settings ›
// General › Login Items lists and the user can switch there too. Compiled without ARC, like the
// rest of Cogpit's Objective-C.

#import <Foundation/Foundation.h>
#import <ServiceManagement/ServiceManagement.h>
#include <stdlib.h>
#include <string.h>

#include "loginitem_darwin.h"

int cogpitLoginItemInAppBundle(void) {
	@autoreleasepool {
		NSBundle *b = [NSBundle mainBundle];
		return [[b bundlePath] hasSuffix:@".app"] && [b bundleIdentifier] != nil;
	}
}

long cogpitLoginItemStatus(void) {
	@autoreleasepool {
		return (long)[SMAppService mainAppService].status;
	}
}

// describe returns err's description as a C string the caller frees.
static char *describe(NSError *err) {
	NSString *s = err == nil ? @"macOS gave no reason"
	                         : [NSString stringWithFormat:@"%@ (%@ %ld)", err.localizedDescription, err.domain, (long)err.code];
	const char *c = [s UTF8String];
	return strdup(c == NULL ? "macOS gave no reason" : c);
}

char *cogpitLoginItemRegister(void) {
	@autoreleasepool {
		NSError *err = nil;
		if ([[SMAppService mainAppService] registerAndReturnError:&err]) {
			return NULL;
		}
		return describe(err);
	}
}

char *cogpitLoginItemUnregister(void) {
	@autoreleasepool {
		NSError *err = nil;
		if ([[SMAppService mainAppService] unregisterAndReturnError:&err]) {
			return NULL;
		}
		return describe(err);
	}
}

void cogpitLoginItemOpenSettings(void) {
	@autoreleasepool {
		[SMAppService openSystemSettingsLoginItems];
	}
}
