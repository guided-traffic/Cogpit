//go:build darwin

package main

import "syscall"

// mntReadOnly is MNT_RDONLY of <sys/mount.h>, which package syscall does not export for darwin.
const mntReadOnly = 0x1

// readOnlyVolume reports whether path lies on a volume mounted read-only, such as a disk image.
// A path it cannot check counts as read-only: then nothing is registered that may not last.
func readOnlyVolume(path string) bool {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return true
	}
	return st.Flags&mntReadOnly != 0
}
