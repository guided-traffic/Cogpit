// Package update is the desktop app's updater (D-4; docs/developer/architecture.md §11): on macOS
// Sparkle 2, which the app bundle carries in Contents/Frameworks and which checks Cogpit's signed
// appcast, shows Sparkle's own window when an update is there, verifies the disk image's EdDSA
// signature and Apple's code signature, replaces the app and relaunches it. cogpit-cli has no
// updater, and Windows and Linux come with M11.
//
// The binary does not link Sparkle. New loads Sparkle.framework at run time from the running app's
// bundle (dlopen through NSBundle) and finds SPUStandardUpdaterController with NSClassFromString,
// through cgo in sparkle_darwin.m; so every build of the module, its tests and the Linux CI work
// without the framework, and an app without it, or without the update key in its Info.plist,
// simply has no updater. Updater.Unavailable then says why, in the words the Help menu shows.
//
// The engine drives it (engine/updates.go): the preference "Check for updates automatically" is
// Cogpit's, and the engine hands it to Start and SetAutomaticChecks, which set Sparkle's
// automaticallyChecksForUpdates; Help › Check for Updates… calls CheckForUpdates. Sparkle runs on
// the main thread: every call is queued onto the main dispatch queue, which the app's event loop
// drains, so no method blocks and any goroutine may call them.
//
// Layer L4: it imports nothing of Cogpit's; cmd/cogpit wires it into the engine.
package update

import (
	"errors"
	"sync"
)

// The reasons this Cogpit cannot update itself, as Unavailable returns them.
var (
	// ErrNotMacOS: only the macOS app updates itself so far.
	ErrNotMacOS = errors.New("not macOS: Cogpit updates itself on macOS only so far (Windows and Linux with M11)")
	// ErrNoBundle: the process is not an app bundle's executable (go run, bin/cogpit).
	ErrNoBundle = errors.New("not installed from the disk image: this Cogpit does not run as an app bundle")
	// ErrNoFramework: the bundle carries no Sparkle.framework.
	ErrNoFramework = errors.New("not installed from the disk image: this app bundle carries no Sparkle.framework")
	// ErrNotInstalled: the app runs from its disk image or a translocated (quarantined) copy, which
	// Sparkle cannot replace.
	ErrNotInstalled = errors.New("not installed yet: Cogpit runs from its disk image or a quarantined folder; drag it into Applications first")
	// ErrNoFeed: the bundle's Info.plist names no appcast (SUFeedURL).
	ErrNoFeed = errors.New("this build has no update feed")
	// ErrNoKey: the bundle's Info.plist has no public update key (SUPublicEDKey): a build made
	// without SPARKLE_PUBLIC_ED_KEY, such as every local one.
	ErrNoKey = errors.New("this build has no update key")
	// ErrNotStarted: CheckForUpdates before Start.
	ErrNotStarted = errors.New("the updater has not started yet")
)

// Updater is the app's updater. Its zero value has none (ErrNotMacOS); New returns the real one.
type Updater struct {
	err error // why there is none: nil once Sparkle is loaded

	mu      sync.Mutex
	started bool
}

// New loads Sparkle from the running app's bundle when it can: a macOS app bundle with
// Contents/Frameworks/Sparkle.framework, SUFeedURL and SUPublicEDKey in its Info.plist, installed
// (not on a read-only volume, not translocated). Nothing is started and nothing reaches the
// network; Unavailable says why when it cannot.
func New() *Updater { return &Updater{err: load()} }

// Unavailable returns why this Cogpit cannot update itself, nil when it can.
func (u *Updater) Unavailable() error {
	switch {
	case u == nil:
		return ErrNotMacOS
	case u.err != nil:
		return u.err
	}
	return startFailure()
}

// Start starts Sparkle with its scheduled checks on or off, once; later calls do nothing. Sparkle
// starts when the app's event loop next runs, after the app finished launching.
func (u *Updater) Start(automaticChecks bool) {
	if u.Unavailable() != nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.started {
		u.started = true
		start(automaticChecks)
	}
}

// CheckForUpdates checks the appcast now, as the user asked: Sparkle shows its window with the
// result, and an update installs only when the user clicks Install in it.
func (u *Updater) CheckForUpdates() error {
	if err := u.Unavailable(); err != nil {
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.started {
		return ErrNotStarted
	}
	check()
	return nil
}

// SetAutomaticChecks turns Sparkle's scheduled checks on or off; before Start it does nothing
// (Start takes the value).
func (u *Updater) SetAutomaticChecks(on bool) {
	if u.Unavailable() != nil {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.started {
		setAutomaticChecks(on)
	}
}
