// macOS's Reduce motion setting: the AppKit side of reducemotion_darwin.go.

// cogpitReduceMotion returns 1 while System Settings › Accessibility › Display › Reduce motion is
// on (NSWorkspace's accessibilityDisplayShouldReduceMotion), else 0.
int cogpitReduceMotion(void);
