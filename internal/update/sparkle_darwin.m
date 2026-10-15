// Sparkle 2 loaded at run time: the AppKit side of sparkle_darwin.go. Cogpit's binary does not
// link Sparkle, so its headers are not used either: the protocols below declare the few methods of
// SPUStandardUpdaterController, SPUUpdater and SUAppcastItem that Cogpit calls, and the delegate
// implements those of SPUStandardUserDriverDelegate it needs, as Sparkle 2.10's headers declare
// them, so that the compiler knows their argument and return types. Compiled
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
- (void)checkForUpdatesInBackground;
@end

// SUAppcastItem, the update the user driver delegate is told about: only its version is read.
@protocol CogpitSUAppcastItem <NSObject>
@property (nonatomic, copy, readonly) NSString *displayVersionString;
@end

// The Go side's report of an update held for the user, or of the update session's end
// (sparkle_darwin.go's export; the test harness defines its own).
extern void cogpitSparkleUpdateChanged(bool available, char *version, int length);

static void reportUpdate(bool available, id item) {
	char version[COGPIT_SPARKLE_VERSION_MAX + 1] = {0};
	if (available && [item respondsToSelector:@selector(displayVersionString)]) {
		NSString *v = [(id<CogpitSUAppcastItem>)item displayVersionString];
		if ([v isKindOfClass:[NSString class]] && v.UTF8String != NULL) {
			strlcpy(version, v.UTF8String, sizeof version);
		}
	}
	cogpitSparkleUpdateChanged(available, version, (int)strlen(version));
}

// CogpitSparkleDriverDelegate is the standard user driver's delegate, as SPUStandardUserDriverDelegate
// in Sparkle 2.10's headers declares its methods: Cogpit takes over showing the updates scheduled
// checks find (Sparkle's gentle reminders): Sparkle shows no window by itself, the window shows its
// Update button while Sparkle holds the update, and the button's click (checkForUpdates:) brings
// Sparkle's window with it up. A check the user started is Sparkle's to show, as always; it is
// reported too, so the button shows while that window holds an update. Sparkle calls these on the
// main thread.
@interface CogpitSparkleDriverDelegate : NSObject
@end

@implementation CogpitSparkleDriverDelegate
- (BOOL)supportsGentleScheduledUpdateReminders {
	return YES;
}

- (BOOL)standardUserDriverShouldHandleShowingScheduledUpdate:(id)update andInImmediateFocus:(BOOL)immediateFocus {
	return NO; // never a window by itself, not even at launch: the Update button (F7)
}

- (void)standardUserDriverWillHandleShowingUpdate:(BOOL)handleShowingUpdate forUpdate:(id)update state:(id)state {
	reportUpdate(true, update);
}

- (void)standardUserDriverWillFinishUpdateSession {
	reportUpdate(false, nil);
}
@end

@protocol CogpitSPUStandardUpdaterController <NSObject>
- (instancetype)initWithStartingUpdater:(BOOL)startUpdater updaterDelegate:(id)updaterDelegate userDriverDelegate:(id)userDriverDelegate;
@property (nonatomic, readonly) id<CogpitSPUUpdater> updater;
- (void)checkForUpdates:(id)sender;
@end

static Class controllerClass;                                // set once by cogpitSparkleLoad
static id<CogpitSPUStandardUpdaterController> controller;    // main thread only
static CogpitSparkleDriverDelegate *driverDelegate;          // kept: Sparkle holds its delegates weakly
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
		if (driverDelegate == nil) {
			driverDelegate = [[CogpitSparkleDriverDelegate alloc] init]; // made with alloc: kept
		}
		id<CogpitSPUStandardUpdaterController> c =
			[(id<CogpitSPUStandardUpdaterController>)[controllerClass alloc] initWithStartingUpdater:NO updaterDelegate:nil userDriverDelegate:driverDelegate];
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

// cogpitSparkleCheckInBackground runs the check Sparkle's scheduler runs, at once: what it finds
// goes to the user driver's delegate. Any thread (the test harness; the app leaves the schedule to
// Sparkle).
void cogpitSparkleCheckInBackground(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (controller != nil && atomic_load(&state) == COGPIT_SPARKLE_RUNNING) {
			[controller.updater checkForUpdatesInBackground];
		}
	});
}

// cogpitSparkleAutomaticChecks returns Sparkle's automaticallyChecksForUpdates, -1 without a
// controller. Main thread (the test harness).
int cogpitSparkleAutomaticChecks(void) {
	if (controller == nil) {
		return -1;
	}
	return controller.updater.automaticallyChecksForUpdates ? 1 : 0;
}
