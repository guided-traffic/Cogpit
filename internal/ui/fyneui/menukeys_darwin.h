// The window's own keys in macOS's main menu (A55): the AppKit side of menukeys_darwin.go.

#include <stdbool.h>

// cogpitMenuKeyEvent reports whether AppKit is dispatching a key event: a menu item's action then
// runs for its key equivalent, not for a click. False off the main thread and without an app.
bool cogpitMenuKeyEvent(void);
