package cluster

import (
	"bytes"
	"encoding/binary"
	"os"
	"syscall"
	"unsafe"
)

// Cogpit's child processes on macOS, for ending a credential plugin's run (loginrun.go): libproc's
// proc_listchildpids and the command line from sysctl kern.procargs2, called as the kernel's own
// system calls so that the package stays free of cgo (cogpit-cli is pure Go).

const (
	procInfoCallListPIDs = 1  // PROC_INFO_CALL_LISTPIDS (sys/proc_info.h)
	procPPIDOnly         = 6  // PROC_PPID_ONLY: the processes whose parent is the given one
	ctlKern              = 1  // CTL_KERN
	kernProcArgs2        = 49 // KERN_PROCARGS2: argc, the executable's path, argv, the environment
	maxChildren          = 1024
)

// childProcesses returns Cogpit's child processes with their command lines.
func childProcesses() []childProcess {
	pids := make([]int32, maxChildren)
	// #nosec G103 -- proc_info(PROC_INFO_CALL_LISTPIDS, PROC_PPID_ONLY, ppid) fills the slice's array, bounded by its size.
	n, _, errno := syscall.Syscall6(syscall.SYS_PROC_INFO, procInfoCallListPIDs, procPPIDOnly, uintptr(os.Getpid()), 0,
		uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)*4))
	if errno != 0 {
		return nil
	}
	var out []childProcess
	for _, pid := range pids[:min(int(n)/4, len(pids))] {
		if pid <= 0 {
			continue
		}
		if argv := procArgv(int(pid)); argv != nil {
			out = append(out, childProcess{pid: int(pid), argv: argv})
		}
	}
	return out
}

// procArgv is a process's command line, from kern.procargs2; nil when it cannot be read.
func procArgv(pid int) []string {
	mib := []int32{ctlKern, kernProcArgs2, int32(pid)} // #nosec G115 -- a process number fits an int32
	size := uintptr(0)
	// #nosec G103 -- sysctl's size query: the MIB and the size's address, no buffer.
	if _, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)), 0,
		uintptr(unsafe.Pointer(&size)), 0, 0); errno != 0 || size == 0 {
		return nil
	}
	buf := make([]byte, size)
	// #nosec G103 -- sysctl fills buf, bounded by size, which it sets to the length written.
	if _, _, errno := syscall.Syscall6(syscall.SYS___SYSCTL, uintptr(unsafe.Pointer(&mib[0])), uintptr(len(mib)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)), 0, 0); errno != 0 {
		return nil
	}
	return parseProcArgs(buf[:min(size, uintptr(len(buf)))])
}

// parseProcArgs reads kern.procargs2's layout: argc (a 32-bit integer), the executable's path, NUL
// padding, then argc NUL-terminated arguments (the environment follows; it is not read).
func parseProcArgs(b []byte) []string {
	if len(b) < 4 {
		return nil
	}
	argc := int(binary.LittleEndian.Uint32(b))
	rest := b[4:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return nil
	}
	rest = rest[i:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	argv := make([]string, 0, min(argc, 64))
	for len(argv) < argc && len(rest) > 0 {
		j := bytes.IndexByte(rest, 0)
		if j < 0 {
			j = len(rest)
		}
		argv = append(argv, string(rest[:j]))
		rest = rest[min(j+1, len(rest)):]
	}
	if len(argv) != argc {
		return nil
	}
	return argv
}
