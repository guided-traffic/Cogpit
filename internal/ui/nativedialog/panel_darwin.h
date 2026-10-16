// The system's file panels and security-scoped bookmarks on macOS: the AppKit side of
// panel_darwin.go. Windows are passed as the NSWindow pointer Fyne hands out
// (driver.MacWindowContext), as an integer.

#include <stddef.h>
#include <stdint.h>

// What a bookmark of a picked item allows (nativedialog.Access).
enum {
	COGPIT_BOOKMARK_NONE = 0,
	COGPIT_BOOKMARK_READ = 1,
	COGPIT_BOOKMARK_READ_WRITE = 2,
};

// What cogpitPanelOpen and cogpitPanelSave did.
enum {
	COGPIT_PANEL_SHOWN = 0,     // the sheet is up; the result follows through the Go callbacks
	COGPIT_PANEL_NO_WINDOW = 1, // no window to attach it to
	COGPIT_PANEL_BUSY = 2,      // the window already shows a sheet
	COGPIT_PANEL_NOT_MAIN = 3,  // not called on the main thread
};

// cogpitPanelOptions configure a panel. Strings are UTF-8 and copied before the call returns;
// NULL or "" means unset.
typedef struct {
	const char *title;
	const char *message;
	const char *prompt;
	const char *directory; // the folder the panel opens in
	const char *name;      // a save panel's suggested file name
	const char *types;     // allowed file name extensions without the dot, separated by '\n'
	int files;             // an open panel may choose files
	int folders;           // an open panel may choose folders
	int multiple;          // an open panel may choose several items
	int hidden;            // hidden files are listed
	int bookmark;          // an open panel's bookmark of each chosen item: COGPIT_BOOKMARK_*
} cogpitPanelOptions;

// cogpitPanelOpen shows an open panel configured by o as a sheet on window. When the user ends
// it, the Go callbacks run on the main thread: cogpitPanelPicked for each chosen item (none for a
// cancel), then cogpitPanelDone with id.
int cogpitPanelOpen(uintptr_t window, const cogpitPanelOptions *o, uintptr_t id);
// cogpitPanelSave shows a save panel as a sheet on window: cogpitPanelPicked with the chosen
// path (none for a cancel), then cogpitPanelDone.
int cogpitPanelSave(uintptr_t window, const cogpitPanelOptions *o, uintptr_t id);

// cogpitPanelMakeOpen and cogpitPanelMakeSave return the panel cogpitPanelOpen and
// cogpitPanelSave show, configured but not shown (an autoreleased NSOpenPanel or NSSavePanel), and
// cogpitPanelDescribe writes its configuration as one line of "key=value" words: what the
// harness test checks (testdata/panel_harness.m).
void *cogpitPanelMakeOpen(const cogpitPanelOptions *o);
void *cogpitPanelMakeSave(const cogpitPanelOptions *o);
void cogpitPanelDescribe(void *panel, char *buf, size_t size);

// cogpitBookmarkCreate makes a security-scoped bookmark of path, read-only or read-write
// (COGPIT_BOOKMARK_*). It returns 1 and the bookmark in *data (malloc'd; free it with
// cogpitFree), or 0 and the reason in err.
int cogpitBookmarkCreate(const char *path, int access, void **data, size_t *length, char *err, size_t errSize);
// cogpitBookmarkResolve resolves a security-scoped bookmark without showing anything and without
// mounting a volume, and starts accessing the item for the rest of the process's life. It returns
// 1 with the item's path in *path (malloc'd), *accessing set when startAccessing succeeded, and,
// when the bookmark is stale, a fresh one of the same access in *fresh (malloc'd, else NULL); or
// 0 and the reason in err.
int cogpitBookmarkResolve(const void *data, size_t length, int access, char **path, int *accessing, void **fresh,
                          size_t *freshLength, char *err, size_t errSize);
// cogpitFree frees what the functions above allocated.
void cogpitFree(void *p);
