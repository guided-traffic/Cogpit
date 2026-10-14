// cogpit:// links on macOS: the AppKit side of deeplink_darwin.go.

// COGPIT_MAX_DEEPLINK is the most of a link handed to Go: one byte more than engine.MaxDeepLink,
// so that a longer link is still seen as too long and dropped there.
#define COGPIT_MAX_DEEPLINK 2049

// cogpitDeepLinkInstall installs the handler of the "open URL" Apple event (kInternetEventClass,
// kAEGetURL) at once and again when the app is about to finish launching, so that AppKit's own
// handlers do not replace it and a link that started the app is not missed. Each link goes to the
// Go function cogpitDeepLinkReceived. Main thread, before the app runs.
void cogpitDeepLinkInstall(void);
