// Cogpit's About in macOS's application menu (A60): the AppKit side of appmenu_darwin.go.

#include <stdbool.h>

// cogpitAppMenuInstallAbout makes the application menu's About item show Cogpit's own dialog: the
// item GLFW created ("About <app name>", AppKit's standard panel) now calls the Go function
// cogpitAboutChosen and is titled title; an application menu without one gets it at the top. A
// second call finds the item again. Returns false while the app has no application menu. Main
// thread only.
bool cogpitAppMenuInstallAbout(const char *title);
