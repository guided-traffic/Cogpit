// Package nativedialog shows the operating system's own file panels and keeps the access macOS
// grants to what the user picks in them (D-72): on macOS NSOpenPanel and NSSavePanel, shown as a
// sheet on Cogpit's window, and security-scoped bookmarks of the files and folders Cogpit reads
// again later. Elsewhere (Windows and Linux until M11) and without cgo it has no panels: Open and
// Save return ErrUnsupported and the caller shows its own dialog.
//
// Why the system's panels: a file the user chooses in them is granted to Cogpit by macOS without
// a privacy prompt (Apple's "user intent"), and Cogpit itself never lists a folder to draw a
// chooser, so it never touches the Desktop, Documents or Downloads folders on its own. Fyne's
// file dialog opened and read those three folders for its sidebar each time it was shown, which
// raised macOS's "would like to access files in your … folder" prompts.
//
// Layer L4 (internal/ui): it imports no Fyne and nothing that talks to a cluster; the shell
// (internal/ui/fyneui) and the app's start-up (cmd/cogpit) use it.
package nativedialog

import "errors"

// Errors of Open and Save. A caller that gets any of them shows its own dialog, except ErrBusy:
// a panel is already up on the window, and the request is dropped.
var (
	// ErrUnsupported: this build has no system panels (not macOS, or no cgo).
	ErrUnsupported = errors.New("the system's file panels are not available in this build")
	// ErrNoWindow: there is no window to attach the panel to (Fyne's test driver, a closed window).
	ErrNoWindow = errors.New("no window for the system's file panel")
	// ErrBusy: the window already shows a panel or another sheet.
	ErrBusy = errors.New("the window already shows a file panel")
	// ErrNotMain: the panel was asked for off the main thread, where AppKit cannot show it.
	ErrNotMain = errors.New("the system's file panel can only be shown from the main thread")
)

// Access is what a bookmark of a picked item lets Cogpit do with it in a later run.
type Access int

const (
	// NoBookmark keeps no bookmark: the item is read or written once, while the panel's grant
	// lasts (an image copied into Cogpit's own folder, a file saved).
	NoBookmark Access = iota
	// ReadOnly keeps a read-only security-scoped bookmark (a kubeconfig, a CA file).
	ReadOnly
	// ReadWrite keeps a read-write one (a folder Cogpit writes into or starts a shell in).
	ReadWrite
)

// OpenOptions configure the open panel.
type OpenOptions struct {
	// Title names the panel for accessibility; a sheet shows Message instead.
	Title string
	// Message is the line above the file list.
	Message string
	// Prompt is the default button's text; empty: the system's "Open".
	Prompt string
	// Files and Folders say what may be chosen; both may be true (Sync Files and Folders).
	Files, Folders bool
	// Multiple lets the user choose several items at once.
	Multiple bool
	// ShowHidden lists hidden files and folders (~/.kube, a .kubeconfig).
	ShowHidden bool
	// Types are the file name extensions that may be chosen, without the dot ("png"); empty:
	// any file.
	Types []string
	// Dir is the folder the panel opens in; empty: the folder the system remembers.
	Dir string
	// Bookmark is the bookmark kept of each chosen item.
	Bookmark Access
}

// SaveOptions configure the save panel.
type SaveOptions struct {
	// Title names the panel for accessibility; a sheet shows Message instead.
	Title string
	// Message is the line above the name field; empty: none.
	Message string
	// Prompt is the default button's text; empty: the system's "Save".
	Prompt string
	// Name is the suggested file name.
	Name string
	// Types are the extensions the file may have, without the dot; the first is added to a name
	// without one. Empty: any name.
	Types []string
	// Dir is the folder the panel opens in; empty: the folder the system remembers.
	Dir string
}

// Pick is one item chosen in an open panel.
type Pick struct {
	// Path is the item's absolute path.
	Path string
	// Folder says that it is a folder.
	Folder bool
	// Bookmark is its security-scoped bookmark, when OpenOptions.Bookmark asked for one and the
	// system made it; nil otherwise.
	Bookmark []byte
	// BookmarkErr says why no bookmark was made, when one was asked for; empty otherwise.
	BookmarkErr string
}
