// A process of its own for TestSparkle* (sparkle_darwin_test.go): built with sparkle_darwin.m and
// placed inside the bundle layouts the test makes, so that [NSBundle mainBundle] is that bundle.
// It loads Sparkle as the app does and prints "load <code> <detail>". With the argument "create"
// it then makes SPUStandardUpdaterController from another thread, as a goroutine would, without
// starting the updater (no update check, no network, no window, nothing written to user
// defaults), while the main thread pumps AppKit's events as GLFW's loop does, and prints
// "state <state> <detail>" and "automatic <0|1|-1>". With "probe" it starts the updater (its
// scheduled checks off), runs the check Sparkle's scheduler runs against the bundle's feed, and
// prints what the user driver's delegate reports, "update <0|1> <version>", the Update button's
// signal (F7); Sparkle shows no window for it.

#import <Cocoa/Cocoa.h>
#include <stdio.h>
#include <string.h>

#include "sparkle_darwin.h"

static int reports;

// The Go side's export in the app (sparkle_darwin.go); here the harness prints the report.
void cogpitSparkleUpdateChanged(bool available, char *version, int length) {
	printf("update %d %.*s\n", available ? 1 : 0, length, version);
	fflush(stdout);
	reports++;
}

// pump runs AppKit's events, as GLFW's loop does, until done says so or the deadline passes.
static void pump(double seconds, BOOL (^done)(void)) {
	NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:seconds];
	while ([deadline timeIntervalSinceNow] > 0) {
		NSEvent *ev = [NSApp nextEventMatchingMask:NSEventMaskAny
		                                 untilDate:[NSDate dateWithTimeIntervalSinceNow:0.05]
		                                    inMode:NSDefaultRunLoopMode
		                                   dequeue:YES];
		if (ev != nil) {
			[NSApp sendEvent:ev];
		}
		if (done()) {
			return;
		}
	}
}

int main(int argc, char **argv) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		char detail[512] = {0};
		int code = cogpitSparkleLoad(detail, sizeof detail);
		printf("load %d %s\n", code, detail);
		if (code != COGPIT_SPARKLE_OK || argc < 2) {
			return 0;
		}
		BOOL probe = strcmp(argv[1], "probe") == 0;
		if (!probe && strcmp(argv[1], "create") != 0) {
			return 0;
		}
		dispatch_async(dispatch_get_global_queue(QOS_CLASS_DEFAULT, 0), ^{
			cogpitSparkleStart(false, probe);
		});
		__block int s = COGPIT_SPARKLE_IDLE;
		int ready = probe ? COGPIT_SPARKLE_RUNNING : COGPIT_SPARKLE_CREATED;
		pump(10, ^BOOL {
			s = cogpitSparkleState(NULL, 0);
			return s == ready || s == COGPIT_SPARKLE_FAILED;
		});
		memset(detail, 0, sizeof detail);
		s = cogpitSparkleState(detail, sizeof detail);
		printf("state %d %s\n", s, detail);
		if (!probe) {
			printf("automatic %d\n", cogpitSparkleAutomaticChecks());
			fflush(stdout);
			return 0;
		}
		fflush(stdout);
		if (s != COGPIT_SPARKLE_RUNNING) {
			return 0;
		}
		cogpitSparkleCheckInBackground();
		pump(20, ^BOOL {
			return reports > 0;
		});
		if (reports == 0) {
			printf("no report\n");
		}
		fflush(stdout);
		return 0;
	}
}
