// Sparkle 2 loaded at run time: the AppKit side of sparkle_darwin.go. Cogpit's binary does not
// link Sparkle, so its headers are not used either: the two protocols below declare the few
// methods of SPUStandardUpdaterController and SPUUpdater that Cogpit calls, as Sparkle 2.10's
// headers declare them, so that the compiler knows their argument and return types. Compiled
// without ARC, like Fyne's Objective-C: the controller is made once and kept for the process's
// life.

#import <Cocoa/Cocoa.h>
#include <stdatomic.h>
#include <string.h>

#include "sparkle_darwin.h"

@protocol CogpitSPUUpdater <NSObject>
@property (nonatomic) BOOL automaticallyChecksForUpdates;
@property (nonatomic, readonly) BOOL canCheckForUpdates;
- (BOOL)startUpdater:(NSError **)error;
@end

@protocol CogpitSPUStandardUpdaterController <NSObject>
- (instancetype)initWithStartingUpdater:(BOOL)startUpdater updaterDelegate:(id)updaterDelegate userDriverDelegate:(id)userDriverDelegate;
@property (nonatomic, readonly) id<CogpitSPUUpdater> updater;
- (void)checkForUpdates:(id)sender;
@end

static Class controllerClass;                                // set once by cogpitSparkleLoad
static id<CogpitSPUStandardUpdaterController> controller;    // main thread only
static _Atomic int state = COGPIT_SPARKLE_IDLE;
static char failure[512];                                    // written before state is FAILED

static void copyText(NSString *text, char *buf, size_t size) {
	if (buf == NULL || size == 0) {
		return;
	}
	const char *s = text.UTF8String;
	strlcpy(buf, s != NULL ? s : "", size);
}

// cogpitSparkleLoad checks the running app's bundle and loads Sparkle.framework from it: an app
// bundle, installed (not on a read-only volume such as its disk image, not translocated), with the
// framework in Contents/Frameworks and SUFeedURL and SUPublicEDKey in its Info.plist; only then is
// the framework loaded (NSBundle dlopens it) and SPUStandardUpdaterController looked up. Nothing
// is started. Any thread.
int cogpitSparkleLoad(char *detail, size_t detailSize) {
	@autoreleasepool {
		NSBundle *main = [NSBundle mainBundle];
		NSString *path = main.bundlePath;
		if (![path.pathExtension isEqualToString:@"app"] || main.bundleIdentifier == nil) {
			copyText(path, detail, detailSize);
			return COGPIT_SPARKLE_NO_BUNDLE;
		}
		if ([path rangeOfString:@"/AppTranslocation/"].location != NSNotFound) {
			copyText(path, detail, detailSize);
			return COGPIT_SPARKLE_NOT_INSTALLED;
		}
		NSNumber *readOnly = nil;
		if ([[NSURL fileURLWithPath:path] getResourceValue:&readOnly forKey:NSURLVolumeIsReadOnlyKey error:NULL] &&
			readOnly.boolValue) {
			copyText(path, detail, detailSize);
			return COGPIT_SPARKLE_NOT_INSTALLED;
		}
		NSString *framework = [main.privateFrameworksPath stringByAppendingPathComponent:@"Sparkle.framework"];
		if (framework == nil || ![[NSFileManager defaultManager] fileExistsAtPath:framework]) {
			copyText(framework, detail, detailSize);
			return COGPIT_SPARKLE_NO_FRAMEWORK;
		}
		id feed = [main objectForInfoDictionaryKey:@"SUFeedURL"];
		if (![feed isKindOfClass:[NSString class]] || [(NSString *)feed length] == 0) {
			return COGPIT_SPARKLE_NO_FEED;
		}
		id key = [main objectForInfoDictionaryKey:@"SUPublicEDKey"];
		if (![key isKindOfClass:[NSString class]] || [(NSString *)key length] == 0) {
			return COGPIT_SPARKLE_NO_KEY;
		}
		NSBundle *sparkle = [NSBundle bundleWithPath:framework];
		NSError *err = nil;
		if (sparkle == nil || ![sparkle loadAndReturnError:&err]) {
			copyText(err != nil ? err.localizedDescription : @"not a bundle", detail, detailSize);
			return COGPIT_SPARKLE_LOAD_FAILED;
		}
		Class c = NSClassFromString(@"SPUStandardUpdaterController");
		if (c == Nil) {
			copyText(@"Sparkle.framework has no SPUStandardUpdaterController", detail, detailSize);
			return COGPIT_SPARKLE_LOAD_FAILED;
		}
		controllerClass = c;
		return COGPIT_SPARKLE_OK;
	}
}

static void fail(NSString *why) {
	copyText(why, failure, sizeof failure);
	atomic_store(&state, COGPIT_SPARKLE_FAILED);
}

// startOnMain makes the controller with its updater and Sparkle's standard window, sets the
// scheduled checks as Cogpit's preference says (only when they differ: Sparkle keeps the value in
// the app's user defaults) and starts the updater, whose error it keeps instead of the alert
// SPUStandardUpdaterController's own startUpdater would show. Main thread.
static void startOnMain(bool automaticChecks, bool startUpdater) {
	@autoreleasepool {
		if (controllerClass == Nil || controller != nil) {
			return;
		}
		id<CogpitSPUStandardUpdaterController> c =
			[(id<CogpitSPUStandardUpdaterController>)[controllerClass alloc] initWithStartingUpdater:NO updaterDelegate:nil userDriverDelegate:nil];
		if (c == nil) {
			fail(@"SPUStandardUpdaterController could not be made");
			return;
		}
		controller = c; // made with alloc: kept
		id<CogpitSPUUpdater> updater = c.updater;
		if (!startUpdater) {
			atomic_store(&state, COGPIT_SPARKLE_CREATED);
			return;
		}
		if (updater.automaticallyChecksForUpdates != automaticChecks) {
			updater.automaticallyChecksForUpdates = automaticChecks;
		}
		NSError *err = nil;
		if (![updater startUpdater:&err]) {
			fail(err != nil ? err.localizedDescription : @"unknown error");
			return;
		}
		atomic_store(&state, COGPIT_SPARKLE_RUNNING);
	}
}

// cogpitSparkleStart queues the start onto the main queue, which the app's event loop drains once
// it runs: Sparkle starts after the app finished launching. startUpdater false makes the
// controller only (the test harness). Any thread.
void cogpitSparkleStart(bool automaticChecks, bool startUpdater) {
	atomic_store(&state, COGPIT_SPARKLE_STARTING);
	dispatch_async(dispatch_get_main_queue(), ^{
		startOnMain(automaticChecks, startUpdater);
	});
}

// cogpitSparkleCheck checks for updates with Sparkle's window, as its "Check for Updates…" menu
// item does; while a check or an update is under way Sparkle's window is already up and nothing
// more happens. Any thread.
void cogpitSparkleCheck(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (controller != nil && atomic_load(&state) == COGPIT_SPARKLE_RUNNING && controller.updater.canCheckForUpdates) {
			[controller checkForUpdates:nil];
		}
	});
}

// cogpitSparkleSetAutomaticChecks turns the scheduled checks on or off. Any thread.
void cogpitSparkleSetAutomaticChecks(bool on) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (controller != nil && atomic_load(&state) == COGPIT_SPARKLE_RUNNING &&
			controller.updater.automaticallyChecksForUpdates != on) {
			controller.updater.automaticallyChecksForUpdates = on;
		}
	});
}

// cogpitSparkleState returns the updater's state and, once it failed to start, why. Any thread.
int cogpitSparkleState(char *detail, size_t detailSize) {
	int s = atomic_load(&state);
	if (s == COGPIT_SPARKLE_FAILED && detail != NULL && detailSize > 0) {
		strlcpy(detail, failure, detailSize);
	}
	return s;
}

// cogpitSparkleAutomaticChecks returns Sparkle's automaticallyChecksForUpdates, -1 without a
// controller. Main thread (the test harness).
int cogpitSparkleAutomaticChecks(void) {
	if (controller == nil) {
		return -1;
	}
	return controller.updater.automaticallyChecksForUpdates ? 1 : 0;
}
