//go:build windows

package engine

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

// The journals' locks are LockFileEx locks on the first byte: they belong to the handle, so two
// opens of one file exclude each other even within a process, and Windows releases them when the
// process ends. Go opens files non-inheritable, so child processes do not keep them. Built and
// vetted by CI on Windows; not run there yet (D-7: Windows after the macOS beta).

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	lockfileFailImmediately = 0x1
	lockfileExclusiveLock   = 0x2
	errorLockViolation      = syscall.Errno(33)
	errorIOPending          = syscall.Errno(997)
)

// tryLockFile takes an exclusive lock on f without waiting; errLockBusy when another handle
// holds it.
func tryLockFile(f *os.File) error {
	err := lockFileEx(f, lockfileExclusiveLock|lockfileFailImmediately)
	if errors.Is(err, errorLockViolation) || errors.Is(err, errorIOPending) {
		return errLockBusy
	}
	return err
}

// lockFile takes an exclusive lock on f, waiting for it.
func lockFile(f *os.File) error { return lockFileEx(f, lockfileExclusiveLock) }

// unlockFile releases f's lock.
func unlockFile(f *os.File) error {
	var ol syscall.Overlapped
	r, _, err := procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
	if r == 0 {
		return err
	}
	return nil
}

func lockFileEx(f *os.File, flags uint32) error {
	var ol syscall.Overlapped
	r, _, err := procLockFileEx.Call(f.Fd(), uintptr(flags), 0, 1, 0, uintptr(unsafe.Pointer(&ol)))
	if r == 0 {
		return err
	}
	return nil
}
