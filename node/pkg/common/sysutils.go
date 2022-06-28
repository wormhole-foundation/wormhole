package common

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// LockMemory locks current and future pages in memory to protect secret keys from being swapped out to disk.
// It's possible (and strongly recommended) to deploy Wormhole such that keys are only ever
// stored in memory and never touch the disk. This is a privileged operation and requires CAP_IPC_LOCK.
func LockMemory() {
	err := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
	if err != nil {
		fmt.Printf("Failed to lock memory: %v (CAP_IPC_LOCK missing?)\n", err)
		os.Exit(1)
	}
}

// SetRestrictiveUmask masks the group and world bits. This ensures that key material
// and sockets we create aren't accidentally group- or world-readable.
func SetRestrictiveUmask() {
	syscall.Umask(0077) // cannot fail
}
