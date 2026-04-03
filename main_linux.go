package main

import (
	"log"
	"golang.org/x/sys/unix"
)

func init() {
	// Process Isolation: Prevent other processes from attaching via ptrace or dumping core memory.
	// This helps protect the decrypted SQLCipher key and plaintext buffers in memory.
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		log.Printf("Warning: failed to set PR_SET_DUMPABLE: %v", err)
	}
}
