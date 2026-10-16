// The system's file panels and security-scoped bookmarks on macOS: the AppKit side of
// panel_darwin.go. Compiled without ARC (-fno-objc-arc), as the rest of Cogpit's Objective-C.

#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#include <stdlib.h>
#include <string.h>

#include "panel_darwin.h"

// The Go side's exports (panel_darwin.go); the harness test defines its own
// (testdata/panel_harness.m).
extern void cogpitPanelPicked(uintptr_t id, char *path, int folder, void *bookmark, size_t length, char *err);
extern void cogpitPanelDone(uintptr_t id, int ok);

// str is a C string as an NSString (autoreleased); nil for NULL, "" and text that is not UTF-8.
static NSString *str(const char *s) {
	if (s == NULL || s[0] == 0) {
		return nil;
	}
	return [NSString stringWithUTF8String:s];
}

// copyText writes s into buf, cut to size and terminated; nothing for a nil s.
static void copyText(NSString *s, char *buf, size_t size) {
	if (buf == NULL || size == 0) {
		return;
	}
	buf[0] = 0;
	const char *u = s != nil ? [s UTF8String] : NULL;
	if (u != NULL) {
		strlcpy(buf, u, size);
	}
}

// contentTypes are the types of the extensions in types, one per line, each once; nil for none.
// An extension no app declares still gets a dynamic type that matches it.
static NSArray<UTType *> *contentTypes(const char *types) {
	NSString *all = str(types);
	if (all == nil) {
		return nil;
	}
	NSMutableArray<UTType *> *out = [NSMutableArray array];
	for (NSString *ext in [all componentsSeparatedByString:@"\n"]) {
		if (ext.length == 0) {
			continue;
		}
		UTType *t = [UTType typeWithFilenameExtension:ext];
		if (t != nil && ![out containsObject:t]) {
			[out addObject:t];
		}
	}
	return out.count > 0 ? out : nil;
}

// configure sets what open and save panels share.
static void configure(NSSavePanel *p, const cogpitPanelOptions *o) {
	NSString *s;
	if ((s = str(o->title)) != nil) {
		p.title = s;
	}
	if ((s = str(o->message)) != nil) {
		p.message = s;
	}
	if ((s = str(o->prompt)) != nil) {
		p.prompt = s;
	}
	if ((s = str(o->directory)) != nil) {
		p.directoryURL = [NSURL fileURLWithPath:s isDirectory:YES];
	}
	p.showsHiddenFiles = o->hidden != 0;
	NSArray<UTType *> *types = contentTypes(o->types);
	if (types != nil) {
		p.allowedContentTypes = types;
	}
}

void *cogpitPanelMakeOpen(const cogpitPanelOptions *o) {
	NSOpenPanel *p = [NSOpenPanel openPanel];
	configure(p, o);
	p.canChooseFiles = o->files != 0;
	p.canChooseDirectories = o->folders != 0;
	p.allowsMultipleSelection = o->multiple != 0;
	p.canCreateDirectories = NO; // choosing what to read, not making room
	p.resolvesAliases = YES;
	p.treatsFilePackagesAsDirectories = NO;
	return p;
}

void *cogpitPanelMakeSave(const cogpitPanelOptions *o) {
	NSSavePanel *p = [NSSavePanel savePanel];
	configure(p, o);
	NSString *name = str(o->name);
	if (name != nil) {
		p.nameFieldStringValue = name;
	}
	p.canCreateDirectories = YES;
	p.extensionHidden = NO; // the name says it is a .yaml, a .csv or a .log
	return p;
}

void cogpitPanelDescribe(void *panel, char *buf, size_t size) {
	@autoreleasepool {
		NSSavePanel *p = (NSSavePanel *)panel;
		NSMutableArray<NSString *> *ids = [NSMutableArray array];
		for (UTType *t in p.allowedContentTypes) {
			[ids addObject:t.identifier];
		}
		NSMutableArray<NSString *> *f = [NSMutableArray array];
		if ([p isKindOfClass:[NSOpenPanel class]]) {
			NSOpenPanel *op = (NSOpenPanel *)p;
			[f addObject:@"kind=open"];
			[f addObject:[NSString stringWithFormat:@"files=%d", op.canChooseFiles ? 1 : 0]];
			[f addObject:[NSString stringWithFormat:@"folders=%d", op.canChooseDirectories ? 1 : 0]];
			[f addObject:[NSString stringWithFormat:@"multiple=%d", op.allowsMultipleSelection ? 1 : 0]];
		} else {
			[f addObject:@"kind=save"];
			[f addObject:[NSString stringWithFormat:@"name=%@", p.nameFieldStringValue ?: @""]];
			[f addObject:[NSString stringWithFormat:@"extensionHidden=%d", p.extensionHidden ? 1 : 0]];
		}
		[f addObject:[NSString stringWithFormat:@"hidden=%d", p.showsHiddenFiles ? 1 : 0]];
		[f addObject:[NSString stringWithFormat:@"create=%d", p.canCreateDirectories ? 1 : 0]];
		[f addObject:[NSString stringWithFormat:@"types=%@", [ids componentsJoinedByString:@","]]];
		[f addObject:[NSString stringWithFormat:@"dir=%@", p.directoryURL.path ?: @""]];
		[f addObject:[NSString stringWithFormat:@"prompt=%@", p.prompt ?: @""]];
		[f addObject:[NSString stringWithFormat:@"title=%@", p.title ?: @""]];
		[f addObject:[NSString stringWithFormat:@"message=%@", p.message ?: @""]];
		copyText([f componentsJoinedByString:@"\t"], buf, size);
	}
}

// createOptions are the bookmark options of an access: always security-scoped, read-only when
// nothing more is needed.
static NSURLBookmarkCreationOptions createOptions(int access) {
	NSURLBookmarkCreationOptions opts = NSURLBookmarkCreationWithSecurityScope;
	if (access == COGPIT_BOOKMARK_READ) {
		opts |= NSURLBookmarkCreationSecurityScopeAllowOnlyReadAccess;
	}
	return opts;
}

static NSData *makeBookmark(NSURL *url, int access, NSError **err) {
	return [url bookmarkDataWithOptions:createOptions(access) includingResourceValuesForKeys:nil relativeToURL:nil error:err];
}

static void *copyBytes(NSData *d) {
	void *p = malloc(d.length > 0 ? d.length : 1);
	if (p != NULL && d.length > 0) {
		memcpy(p, d.bytes, d.length);
	}
	return p;
}

int cogpitBookmarkCreate(const char *path, int access, void **data, size_t *length, char *err, size_t errSize) {
	*data = NULL;
	*length = 0;
	@autoreleasepool {
		NSString *s = str(path);
		if (s == nil) {
			copyText(@"no path", err, errSize);
			return 0;
		}
		NSError *e = nil;
		NSData *d = makeBookmark([NSURL fileURLWithPath:s], access, &e);
		if (d == nil) {
			copyText(e.localizedDescription ?: @"the system made no bookmark", err, errSize);
			return 0;
		}
		*data = copyBytes(d);
		*length = d.length;
		return 1;
	}
}

// held are the items being accessed: kept for the process's life, since Cogpit reads them again
// (a sync folder on every poll) and access ends with stopAccessing or the process.
static NSMutableArray<NSURL *> *held;

int cogpitBookmarkResolve(const void *data, size_t length, int access, char **path, int *accessing, void **fresh,
                          size_t *freshLength, char *err, size_t errSize) {
	*path = NULL;
	*accessing = 0;
	*fresh = NULL;
	*freshLength = 0;
	@autoreleasepool {
		NSData *d = [NSData dataWithBytes:data length:length];
		BOOL stale = NO;
		NSError *e = nil;
		NSURLBookmarkResolutionOptions opts = NSURLBookmarkResolutionWithSecurityScope | NSURLBookmarkResolutionWithoutUI |
		                                      NSURLBookmarkResolutionWithoutMounting;
		NSURL *url = [NSURL URLByResolvingBookmarkData:d options:opts relativeToURL:nil bookmarkDataIsStale:&stale error:&e];
		if (url == nil) {
			copyText(e.localizedDescription ?: @"the bookmark does not resolve", err, errSize);
			return 0;
		}
		if ([url startAccessingSecurityScopedResource]) {
			*accessing = 1;
			@synchronized([NSURL class]) {
				if (held == nil) {
					held = [[NSMutableArray alloc] init];
				}
				[held addObject:url];
			}
		}
		if (stale) {
			NSData *n = makeBookmark(url, access, nil);
			if (n != nil) {
				*fresh = copyBytes(n);
				*freshLength = n.length;
			}
		}
		const char *fs = url.fileSystemRepresentation;
		*path = fs != NULL ? strdup(fs) : NULL;
		return 1;
	}
}

void cogpitFree(void *p) { free(p); }

// report hands one chosen item to Go: its path, whether it is a folder and, if asked for, its
// bookmark, made while the panel's grant is fresh.
static void report(uintptr_t id, NSURL *u, int bookmark) {
	const char *fs = u.fileSystemRepresentation;
	if (fs == NULL) {
		return;
	}
	NSNumber *dir = nil;
	[u getResourceValue:&dir forKey:NSURLIsDirectoryKey error:nil];
	void *bytes = NULL;
	size_t n = 0;
	char err[512] = {0};
	if (bookmark != COGPIT_BOOKMARK_NONE) {
		NSError *e = nil;
		NSData *d = makeBookmark(u, bookmark, &e);
		if (d != nil) {
			bytes = (void *)d.bytes; // copied by Go before the call returns
			n = d.length;
		} else {
			copyText(e.localizedDescription ?: @"the system made no bookmark", err, sizeof err);
		}
	}
	cogpitPanelPicked(id, (char *)fs, dir.boolValue ? 1 : 0, bytes, n, err);
}

// begin shows p as a sheet on window. The completion handler runs on the main thread; the block
// holds the panel until it ran.
static int begin(uintptr_t window, NSSavePanel *p, uintptr_t id, int bookmark, BOOL open) {
	NSWindow *w = (NSWindow *)(void *)window;
	if (w == nil) {
		return COGPIT_PANEL_NO_WINDOW;
	}
	if (w.attachedSheet != nil) {
		return COGPIT_PANEL_BUSY;
	}
	[p beginSheetModalForWindow:w
	          completionHandler:^(NSModalResponse r) {
		            BOOL ok = r == NSModalResponseOK;
		            if (ok) {
			            NSArray<NSURL *> *urls = open ? ((NSOpenPanel *)p).URLs : (p.URL != nil ? @[ p.URL ] : @[]);
			            for (NSURL *u in urls) {
				            report(id, u, bookmark);
			            }
		            }
		            cogpitPanelDone(id, ok ? 1 : 0);
	            }];
	return COGPIT_PANEL_SHOWN;
}

int cogpitPanelOpen(uintptr_t window, const cogpitPanelOptions *o, uintptr_t id) {
	if (![NSThread isMainThread]) {
		return COGPIT_PANEL_NOT_MAIN;
	}
	@autoreleasepool {
		return begin(window, cogpitPanelMakeOpen(o), id, o->bookmark, YES);
	}
}

int cogpitPanelSave(uintptr_t window, const cogpitPanelOptions *o, uintptr_t id) {
	if (![NSThread isMainThread]) {
		return COGPIT_PANEL_NOT_MAIN;
	}
	@autoreleasepool {
		return begin(window, cogpitPanelMakeSave(o), id, COGPIT_BOOKMARK_NONE, NO);
	}
}
