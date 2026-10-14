// A process of its own for deeplink_darwin.m (TestDeepLinkAppleEvents): installs Cogpit's handler
// as main does, lets AppKit finish launching as GLFW does (with no Dock icon and no window), then
// sends this process "open URL" Apple events and prints what reached cogpitDeepLinkReceived. The
// events go to this process only: nothing is registered with Launch Services, no URL is opened.

#import <Cocoa/Cocoa.h>
#import <CoreServices/CoreServices.h>
#include <stdio.h>

#include "deeplink_darwin.h"

void cogpitDeepLinkReceived(char *url, int length) {
	printf("link %d %.*s\n", length, length > 120 ? 40 : length, url); // a long one: its start
	fflush(stdout);
}

static void sendLink(NSString *url) {
	NSAppleEventDescriptor *target = [NSAppleEventDescriptor currentProcessDescriptor];
	NSAppleEventDescriptor *ev = [NSAppleEventDescriptor appleEventWithEventClass:kInternetEventClass
	                                                                      eventID:kAEGetURL
	                                                             targetDescriptor:target
	                                                                     returnID:kAutoGenerateReturnID
	                                                                transactionID:kAnyTransactionID];
	[ev setParamDescriptor:[NSAppleEventDescriptor descriptorWithString:url] forKeyword:keyDirectObject];
	OSStatus err = AESendMessage([ev aeDesc], NULL, kAENoReply | kAENeverInteract, kAEDefaultTimeout);
	if (err != noErr) {
		printf("send failed %d\n", (int)err);
		fflush(stdout);
	}
}

int main(void) {
	@autoreleasepool {
		cogpitDeepLinkInstall();
		cogpitDeepLinkInstall(); // a second install changes nothing
		sendLink(@"cogpit://open?view=pods");

		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];
		[NSApp finishLaunching]; // AppKit installs its own Apple event handlers here
		sendLink(@"cogpit://open?context=kind-dev+%5Bconfig%5D&view=pods&namespace=team-a&name=web-0");
		sendLink([@"cogpit://open?view=" stringByPaddingToLength:3000 withString:@"x" startingAtIndex:0]);
		sendLink(@"cogpit://open?context=%C3%BCber");
		printf("done\n");
	}
	return 0;
}
