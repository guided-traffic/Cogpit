// cogpit:// links on macOS: the AppKit side of deeplink_darwin.go. macOS delivers a link opened
// anywhere (a browser, Mail, `open cogpit://…`) as an Apple event, whether the app was running or
// was started for it. Compiled without ARC, like the rest of Cogpit's Objective-C.

#import <Cocoa/Cocoa.h>
#import <CoreServices/CoreServices.h>
#include <string.h>

#include "deeplink_darwin.h"

extern void cogpitDeepLinkReceived(char *url, int length);

// CogpitDeepLinkHandler receives the "open URL" Apple events. It is independent of GLFW's app
// delegate: the handler is installed with NSAppleEventManager and installed again from
// NSApplicationWillFinishLaunchingNotification, the moment Apple recommends for Apple event
// handlers, before AppKit dispatches the event that launched the app.
@interface CogpitDeepLinkHandler : NSObject
@end

@implementation CogpitDeepLinkHandler
- (void)install {
	[[NSAppleEventManager sharedAppleEventManager] setEventHandler:self
	                                                   andSelector:@selector(handleGetURL:withReplyEvent:)
	                                                 forEventClass:kInternetEventClass
	                                                    andEventID:kAEGetURL];
}
- (void)willFinishLaunching:(NSNotification *)note {
	[self install];
}
- (void)handleGetURL:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply {
	NSString *url = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
	const char *s = url == nil ? NULL : [url UTF8String];
	if (s == NULL) {
		return;
	}
	size_t n = strlen(s);
	if (n > COGPIT_MAX_DEEPLINK) {
		n = COGPIT_MAX_DEEPLINK;
	}
	cogpitDeepLinkReceived((char *)s, (int)n);
}
@end

// The handler lives as long as the process: installed once, never released.
static CogpitDeepLinkHandler *handler;

void cogpitDeepLinkInstall(void) {
	@autoreleasepool {
		if (handler != nil) {
			return;
		}
		handler = [[CogpitDeepLinkHandler alloc] init];
		[handler install];
		[[NSNotificationCenter defaultCenter] addObserver:handler
		                                         selector:@selector(willFinishLaunching:)
		                                             name:NSApplicationWillFinishLaunchingNotification
		                                           object:nil];
	}
}
