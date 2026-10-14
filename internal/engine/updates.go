package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Updates (D-4, G14; docs/security/updates.md): the macOS app updates itself through Sparkle 2,
// which the binary wires in (internal/update) as an Updater; the engine does no OS calls itself.
// Two preferences are Cogpit's, kept with the others and shown in UIState.Preferences:
// CheckForUpdates ("Check for updates automatically", on by default as in Lens), which the engine
// hands to the updater at start and whenever it changes, so that Cogpit's preference and not
// Sparkle's own prompt decides whether it checks once a day; and UpdateChannel, only Stable until
// Cogpit publishes prereleases. app.checkForUpdates (Help › Check for Updates…) checks now with
// Sparkle's own window; installing stays the user's click in it.
//
// Who may change them (the lead's default, 2026-10-14, for the owner to confirm): the user turns
// the automatic checks on and off; an agent or the CLI may turn them on but not off, since off
// leaves a copy with known flaws unpatched (the reverse of Load chart icons, which an agent may
// only turn off: there off is the safer side). Any origin may run app.checkForUpdates: it changes
// nothing, and Sparkle asks the user before it installs anything.

// Update channels. Stable is the appcast's default channel; Beta (Sparkle's channel "beta") comes
// with the first prerelease.
const (
	UpdateChannelStable = "stable"
	UpdateChannelBeta   = "beta"
)

// Updater is the app's self-update (internal/update.Updater, Sparkle 2 on macOS), wired by the
// binary with SetUpdater. Its methods do not block.
type Updater interface {
	// Unavailable returns why this Cogpit cannot update itself ("this build has no update key",
	// "not installed from the disk image", "not macOS"), nil when it can.
	Unavailable() error
	// Start starts the updater once, its scheduled checks on or off.
	Start(automaticChecks bool)
	// CheckForUpdates checks now, with the updater's own window.
	CheckForUpdates() error
	// SetAutomaticChecks turns the scheduled checks on or off.
	SetAutomaticChecks(on bool)
}

// ErrNoUpdater: a session without an updater (cogpit-cli, a headless engine) does not update
// itself.
var ErrNoUpdater = errors.New("this Cogpit does not update itself: cogpit-cli and headless sessions have no updater; " +
	"download a new release or update it with your package manager")

// ErrUpdateChecksGUIOnly refuses turning the automatic update checks off from anywhere but the
// user's own action in Cogpit's window.
var ErrUpdateChecksGUIOnly = errors.New("only the user, in Cogpit's own window, can turn off the automatic update checks; " +
	"agents and the CLI may only turn them on (off leaves this copy without the fixes of newer releases)")

// ErrNoBetaChannel refuses the Beta channel while there is none.
var ErrNoBetaChannel = errors.New("update channel beta: Cogpit publishes no prereleases yet, so only the stable channel exists")

// updaterSlot holds the session's updater. Its own lock orders a start against preference changes.
type updaterSlot struct {
	mu sync.Mutex
	u  Updater
}

// SetUpdater wires the app's updater in and starts it with the preference's automatic checks,
// unless it says it cannot update this Cogpit. Called once by the desktop binary.
func (s *Session) SetUpdater(u Updater) {
	s.updates.mu.Lock() // before the preference is read: a change meanwhile waits and follows (updatePrefsChanged)
	defer s.updates.mu.Unlock()
	p, _ := s.Preferences()
	s.updates.u = u
	if u == nil {
		return
	}
	if err := u.Unavailable(); err != nil {
		s.log.Info("updates: none for this Cogpit", "reason", err)
		return
	}
	u.Start(p.CheckForUpdates)
}

// updater returns the session's updater; nil without one.
func (s *Session) updater() Updater {
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	return s.updates.u
}

// UpdatesUnavailable returns why this Cogpit cannot check for updates, nil when it can: the
// enabled state of app.checkForUpdates.
func (s *Session) UpdatesUnavailable() error {
	u := s.updater()
	if u == nil {
		return ErrNoUpdater
	}
	return u.Unavailable()
}

// CheckForUpdates checks for updates now, with the updater's own window.
func (s *Session) CheckForUpdates() error {
	if err := s.UpdatesUnavailable(); err != nil {
		return err
	}
	return s.updater().CheckForUpdates()
}

// updatePrefsChanged hands a change of the automatic checks to the updater.
func (s *Session) updatePrefsChanged(old, next Preferences) {
	if old.CheckForUpdates == next.CheckForUpdates {
		return
	}
	s.updates.mu.Lock()
	defer s.updates.mu.Unlock()
	if s.updates.u != nil && s.updates.u.Unavailable() == nil {
		s.updates.u.SetAutomaticChecks(next.CheckForUpdates)
	}
}

// RefuseUpdateChecks refuses a preferences change that turns the automatic update checks off from
// another origin than the GUI.
func RefuseUpdateChecks(ctx context.Context, on *bool) error {
	if on != nil && !*on && OriginOf(ctx) != OriginGUI {
		return ErrUpdateChecksGUIOnly
	}
	return nil
}

// CheckUpdateChannel refuses what is not an update channel Cogpit offers: stable ("" means it
// too). Beta is refused for every origin until prereleases exist; once it is offered, only the
// user is to choose it (the lead's default: a prerelease is less tested code).
func CheckUpdateChannel(ch string) error {
	switch ch {
	case "", UpdateChannelStable:
		return nil
	case UpdateChannelBeta:
		return ErrNoBetaChannel
	}
	return fmt.Errorf("update channel %q: only %s", ch, UpdateChannelStable)
}
