// macOS's signals for the connection watch (D-58): the AppKit and Network.framework side of
// connhints_darwin.go.

// cogpitWatchConnectionHints observes NSWorkspaceDidWakeNotification (the computer woke) and starts
// an nw_path_monitor (an interface or a VPN came or went); each event calls the Go side's
// cogpitConnectionHint with 1 (wake) or 2 (network). Call it once, with the app running.
void cogpitWatchConnectionHints(void);
