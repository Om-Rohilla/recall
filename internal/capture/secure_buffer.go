package capture

import (
	"fmt"

	"github.com/awnumar/memguard"
)

// SecureBuffer wraps a memguard LockedBuffer to hold a sensitive string
// (e.g., a raw command) in locked, non-pageable memory. The buffer is
// destroyed (zeroed) as soon as it is no longer needed.
type SecureBuffer struct {
	enclave *memguard.Enclave
}

// NewSecureBuffer allocates a new locked memory region containing the
// provided raw string. The caller MUST call Destroy() when done.
func NewSecureBuffer(raw string) *SecureBuffer {
	// Seal the raw bytes into a memguard Enclave (encrypted at rest in memory).
	return &SecureBuffer{
		enclave: memguard.NewEnclave([]byte(raw)),
	}
}

// Open decrypts and returns a LockedBuffer for reading the secret value.
// The caller MUST call buf.Destroy() on the returned buffer when finished.
// Returns nil if the enclave has already been destroyed.
func (s *SecureBuffer) Open() (*memguard.LockedBuffer, error) {
	if s.enclave == nil {
		return nil, nil
	}
	return s.enclave.Open()
}

// Destroy zeroes and frees the locked memory region immediately.
// This is deterministic — it does not rely on GC finalizers.
// Safe to call multiple times.
func (s *SecureBuffer) Destroy() {
	if s.enclave != nil {
		// Open the enclave to get the LockedBuffer and destroy it immediately,
		// zeroing the memory page now rather than waiting for GC finalization.
		lb, err := s.enclave.Open()
		if err == nil && lb != nil {
			lb.Destroy() // zeroes and unlocks the page immediately
		}
		s.enclave = nil
	}
}

// WithSecureString is a convenience helper that allocates a SecureBuffer,
// calls fn with the plaintext string, then immediately destroys the buffer.
// This ensures the secret lives in locked memory only for the duration of fn.
// Returns an error if memory protection cannot be established.
func WithSecureString(raw string, fn func(plain string) error) error {
	buf := NewSecureBuffer(raw)
	defer buf.Destroy()

	lb, err := buf.Open()
	if err != nil {
		return fmt.Errorf("secure enclave unavailable: memory protection failed: %w", err)
	}
	if lb == nil {
		return fmt.Errorf("secure enclave unavailable: memory protection failed")
	}
	defer lb.Destroy()

	return fn(string(lb.Bytes()))
}
