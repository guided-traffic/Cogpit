// A process of its own for TestPanelConfiguration and TestBookmarkHarness (panel_darwin_test.go):
// built with panel_darwin.m, it makes a panel configured as cogpitPanelOpen and cogpitPanelSave
// configure it and prints its configuration, without showing it (no window, no sheet, nothing on
// the screen). With "bookmark <file>" it makes a read-only security-scoped bookmark of the file,
// resolves it as the app does at start, and starts and stops accessing it once more.
//
//   open key=value...   an open panel: title, message, prompt, dir, types (comma-separated),
//                       files, folders, multiple, hidden (0 or 1)
//   save key=value...   a save panel: the same and name
//
// A panel prints one line of tab-separated key=value words (cogpitPanelDescribe).

#import <Cocoa/Cocoa.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "panel_darwin.h"

// The Go side's exports in the app (panel_darwin.go); no panel is shown here, so they never run.
void cogpitPanelPicked(uintptr_t id, char *path, int folder, void *bookmark, size_t length, char *err) {
	printf("unexpected pick %s\n", path);
}
void cogpitPanelDone(uintptr_t id, int ok) { printf("unexpected done %d\n", ok); }

static void describe(void *panel) {
	char buf[4096] = {0};
	cogpitPanelDescribe(panel, buf, sizeof buf);
	printf("%s\n", buf);
}

static int bookmark(const char *path) {
	void *data = NULL;
	size_t length = 0;
	char err[512] = {0};
	if (!cogpitBookmarkCreate(path, COGPIT_BOOKMARK_READ, &data, &length, err, sizeof err)) {
		printf("create failed %s\n", err);
		return 1;
	}
	printf("created %d\n", length > 0 ? 1 : 0);
	char *resolved = NULL;
	int accessing = 0;
	void *fresh = NULL;
	size_t freshLength = 0;
	if (!cogpitBookmarkResolve(data, length, COGPIT_BOOKMARK_READ, &resolved, &accessing, &fresh, &freshLength, err,
	                           sizeof err)) {
		printf("resolve failed %s\n", err);
		return 1;
	}
	printf("resolved %s\n", resolved);
	printf("accessing %d\n", accessing);
	printf("stale %d\n", fresh != NULL ? 1 : 0);
	// The app keeps accessing for its life; here a second resolution starts and stops once more.
	NSData *d = [NSData dataWithBytes:data length:length];
	BOOL stale = NO;
	NSURL *url = [NSURL URLByResolvingBookmarkData:d
	                                       options:NSURLBookmarkResolutionWithSecurityScope | NSURLBookmarkResolutionWithoutUI
	                                 relativeToURL:nil
	                           bookmarkDataIsStale:&stale
	                                         error:nil];
	BOOL started = [url startAccessingSecurityScopedResource];
	NSString *text = [NSString stringWithContentsOfFile:url.path encoding:NSUTF8StringEncoding error:nil];
	if (started) {
		[url stopAccessingSecurityScopedResource];
	}
	printf("read %s\n", text != nil ? [text UTF8String] : "(nothing)");
	printf("stopped %d\n", started ? 1 : 0);
	cogpitFree(resolved);
	cogpitFree(fresh);
	cogpitFree(data);
	return 0;
}

// option reads "key=value" into o; the strings stay in argv.
static void option(cogpitPanelOptions *o, char *arg) {
	char *eq = strchr(arg, '=');
	if (eq == NULL) {
		return;
	}
	*eq = 0;
	char *v = eq + 1;
	if (strcmp(arg, "title") == 0) {
		o->title = v;
	} else if (strcmp(arg, "message") == 0) {
		o->message = v;
	} else if (strcmp(arg, "prompt") == 0) {
		o->prompt = v;
	} else if (strcmp(arg, "dir") == 0) {
		o->directory = v;
	} else if (strcmp(arg, "name") == 0) {
		o->name = v;
	} else if (strcmp(arg, "types") == 0) {
		for (char *c = v; *c; c++) {
			if (*c == ',') {
				*c = '\n';
			}
		}
		o->types = v;
	} else if (strcmp(arg, "files") == 0) {
		o->files = atoi(v);
	} else if (strcmp(arg, "folders") == 0) {
		o->folders = atoi(v);
	} else if (strcmp(arg, "multiple") == 0) {
		o->multiple = atoi(v);
	} else if (strcmp(arg, "hidden") == 0) {
		o->hidden = atoi(v);
	}
}

int main(int argc, char **argv) {
	@autoreleasepool {
		if (argc < 2) {
			fprintf(stderr, "usage: harness open|save key=value... | bookmark <file>\n");
			return 2;
		}
		if (strcmp(argv[1], "bookmark") == 0) {
			return argc > 2 ? bookmark(argv[2]) : 2;
		}
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited]; // no Dock icon, no menu bar
		cogpitPanelOptions o = {0};
		for (int i = 2; i < argc; i++) {
			option(&o, argv[i]);
		}
		if (strcmp(argv[1], "open") == 0) {
			describe(cogpitPanelMakeOpen(&o));
		} else if (strcmp(argv[1], "save") == 0) {
			describe(cogpitPanelMakeSave(&o));
		} else {
			fprintf(stderr, "unknown %s\n", argv[1]);
			return 2;
		}
	}
	return 0;
}
