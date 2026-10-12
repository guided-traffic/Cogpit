//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly || windows)

package engine

import "os"

// Platforms without flock or LockFileEx get no lock between processes (H-22): the journals then
// behave as before owners existed, and every owner but this process counts as ended. Cogpit is
// not built for such a platform; this keeps the package compiling there.

func tryLockFile(*os.File) error { return nil }

func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) error { return nil }
