// macOS's signals for the connection watch (D-58): the computer's wake and changes of its network
// path. Compiled without ARC, like the rest of the package's Objective-C (Fyne's).

#import <Cocoa/Cocoa.h>
#import <Network/Network.h>

#include "connhints_darwin.h"

extern void cogpitConnectionHint(int kind);

static nw_path_monitor_t cogpitPathMonitor;

void cogpitWatchConnectionHints(void) {
	// The wake: AppKit posts it on the workspace's own notification centre, on the main thread.
	dispatch_async(dispatch_get_main_queue(), ^{
		[[[NSWorkspace sharedWorkspace] notificationCenter] addObserverForName:NSWorkspaceDidWakeNotification
		                                                                  object:nil
		                                                                   queue:nil
		                                                              usingBlock:^(NSNotification *note) {
			(void)note;
			cogpitConnectionHint(1);
		}];
	});
	// The network: every change of the default path after the first update, which is the path as it
	// is when the monitor starts.
	cogpitPathMonitor = nw_path_monitor_create();
	__block bool first = true;
	nw_path_monitor_set_update_handler(cogpitPathMonitor, ^(nw_path_t path) {
		(void)path;
		if (first) {
			first = false;
			return;
		}
		cogpitConnectionHint(2);
	});
	nw_path_monitor_set_queue(cogpitPathMonitor, dispatch_get_global_queue(QOS_CLASS_UTILITY, 0));
	nw_path_monitor_start(cogpitPathMonitor);
}
