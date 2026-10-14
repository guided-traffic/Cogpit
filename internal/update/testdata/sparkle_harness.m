// A process of its own for TestSparkle* (sparkle_darwin_test.go): built with sparkle_darwin.m and
// placed inside the bundle layouts the test makes, so that [NSBundle mainBundle] is that bundle.
// It loads Sparkle as the app does and prints "load <code> <detail>". With the argument "create"
// it then makes SPUStandardUpdaterController from another thread, as a goroutine would, without
// starting the updater (no update check, no network, no window, nothing written to user
// defaults), while the main thread pumps AppKit's events as GLFW's loop does, and prints
// "state <state> <detail>" and "automatic <0|1|-1>".

#import <Cocoa/Cocoa.h>
#include <stdio.h>
#include <string.h>

#include "sparkle_darwin.h"

int main(int argc, char **argv) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		char detail[512] = {0};
		int code = cogpitSparkleLoad(detail, sizeof detail);
		printf("load %d %s\n", code, detail);
		if (code != COGPIT_SPARKLE_OK || argc < 2 || strcmp(argv[1], "create") != 0) {
			return 0;
		}
		dispatch_async(dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0), ^{
			cogpitSparkleStart(false, false);
		});
		NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:10];
		int s = COGPIT_SPARKLE_IDLE;
		while ([deadline timeIntervalSinceNow] > 0) {
			NSEvent *ev = [NSApp nextEventMatchingMask:NSEventMaskAny
			                                 untilDate:[NSDate dateWithTimeIntervalSinceNow:0.05]
			                                    inMode:NSDefaultRunLoopMode
			                                   dequeue:YES];
			if (ev != nil) {
				[NSApp sendEvent:ev];
			}
			memset(detail, 0, sizeof detail);
			s = cogpitSparkleState(detail, sizeof detail);
			if (s == COGPIT_SPARKLE_CREATED || s == COGPIT_SPARKLE_FAILED) {
				break;
			}
		}
		printf("state %d %s\n", s, detail);
		printf("automatic %d\n", cogpitSparkleAutomaticChecks());
		fflush(stdout);
		return 0;
	}
}
