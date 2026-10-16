package cluster

import (
	"os"
	"syscall"
	"unsafe"
)

// Cogpit's child processes on Windows, for ending a credential plugin's run (loginrun.go): the
// Toolhelp snapshot names each process's parent and program file, not its command line, so a
// plugin is found by its program's file name alone. Not tried on Windows (no runner of it runs the
// tests); the build compiles it.

// childProcesses returns Cogpit's child processes with their programs' file names.
func childProcesses() []childProcess {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer func() { _ = syscall.CloseHandle(snap) }()
	self := uint32(os.Getpid()) // #nosec G115 -- a process number fits a DWORD
	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	var out []childProcess
	for err = syscall.Process32First(snap, &entry); err == nil; err = syscall.Process32Next(snap, &entry) {
		if entry.ParentProcessID == self && entry.ProcessID != 0 {
			out = append(out, childProcess{pid: int(entry.ProcessID), exe: syscall.UTF16ToString(entry.ExeFile[:])})
		}
	}
	return out
}

// signalPlugin ends a credential plugin's process; Windows has no SIGTERM, so both steps terminate
// it.
func signalPlugin(pid int, _ bool) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, uint32(pid)) // #nosec G115 -- a process number fits a DWORD
	if err != nil {
		return err
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	return syscall.TerminateProcess(h, 1)
}
