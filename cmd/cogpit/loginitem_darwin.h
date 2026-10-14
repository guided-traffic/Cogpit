// Start at login on macOS: the ServiceManagement side of loginitem_darwin.go (macOS 13 or later).

// cogpitLoginItemInAppBundle reports whether this process runs from an app bundle with a bundle
// identifier (Cogpit.app), the only kind of process SMAppService.mainAppService describes.
int cogpitLoginItemInAppBundle(void);
// cogpitLoginItemStatus returns SMAppService.mainAppService.status: 0 not registered, 1 enabled,
// 2 requires approval, 3 not found.
long cogpitLoginItemStatus(void);
// cogpitLoginItemRegister and cogpitLoginItemUnregister return NULL on success, else the error's
// description, which the caller frees.
char *cogpitLoginItemRegister(void);
char *cogpitLoginItemUnregister(void);
// cogpitLoginItemOpenSettings opens System Settings › General › Login Items.
void cogpitLoginItemOpenSettings(void);
