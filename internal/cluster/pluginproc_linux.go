package cluster

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Cogpit's child processes on Linux, for ending a credential plugin's run (loginrun.go): /proc's
// stat for the parent and cmdline for the command line.

// childProcesses returns Cogpit's child processes with their command lines.
func childProcesses() []childProcess {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	var out []childProcess
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		dir := filepath.Join("/proc", e.Name())
		stat, err := os.ReadFile(filepath.Join(dir, "stat")) // #nosec G304 -- a /proc entry named by a process number
		if err != nil || procParent(stat) != self {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline")) // #nosec G304 -- a /proc entry named by a process number
		if err != nil || len(cmdline) == 0 {
			continue
		}
		out = append(out, childProcess{pid: pid, argv: strings.Split(strings.TrimSuffix(string(cmdline), "\x00"), "\x00")})
	}
	return out
}

// procParent is the parent's number in /proc/<pid>/stat ("pid (comm) state ppid …"; comm may hold
// spaces and parentheses, so the fields are read after its last ')'); 0 when it cannot be read.
func procParent(stat []byte) int {
	i := bytes.LastIndexByte(stat, ')')
	if i < 0 {
		return 0
	}
	fields := strings.Fields(string(stat[i+1:]))
	if len(fields) < 2 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return ppid
}
