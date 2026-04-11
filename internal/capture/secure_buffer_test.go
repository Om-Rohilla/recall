package capture

import (
	"strings"
	"testing"
)

func TestSecureBuffer_NewAndOpen(t *testing.T) {
	secret := "super-secret-command --token abc123"
	buf := NewSecureBuffer(secret)
	defer buf.Destroy()

	lb, err := buf.Open()
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	if lb == nil {
		t.Fatal("Open() returned nil LockedBuffer")
	}
	defer lb.Destroy()

	if string(lb.Bytes()) != secret {
		t.Errorf("expected %q, got %q", secret, string(lb.Bytes()))
	}
}

func TestSecureBuffer_Destroy_NilsEnclave(t *testing.T) {
	buf := NewSecureBuffer("sensitive data")
	buf.Destroy()

	// After Destroy, enclave must be nil
	if buf.enclave != nil {
		t.Error("enclave should be nil after Destroy()")
	}

	// Open on a destroyed buffer should safely return nil, not panic
	lb, _ := buf.Open()
	if lb != nil {
		t.Error("Open() on destroyed buffer should return nil")
	}
}

func TestSecureBuffer_Destroy_Idempotent(t *testing.T) {
	buf := NewSecureBuffer("data")
	// Must not panic on double-destroy
	buf.Destroy()
	buf.Destroy()
}

func TestWithSecureString_CallsFn(t *testing.T) {
	called := false
	var received string

	err := WithSecureString("my-secret", func(plain string) error {
		called = true
		received = plain
		return nil
	})
	if err != nil {
		t.Fatalf("WithSecureString returned error: %v", err)
	}
	if !called {
		t.Error("callback was not called")
	}
	if received != "my-secret" {
		t.Errorf("expected %q, got %q", "my-secret", received)
	}
}

func TestWithSecureString_EmptyString(t *testing.T) {
	err := WithSecureString("", func(plain string) error {
		if plain != "" {
			return nil
		}
		return nil
	})
	if err != nil {
		// Empty strings may or may not be handled differently by memguard;
		// just ensure it doesn't panic.
		if !strings.Contains(err.Error(), "unavailable") {
			t.Logf("WithSecureString('') returned: %v (acceptable)", err)
		}
	}
}

func TestWithSecureString_FnErrorPropagates(t *testing.T) {
	err := WithSecureString("data", func(plain string) error {
		return errTest
	})
	if err != errTest {
		t.Errorf("expected errTest, got %v", err)
	}
}

var errTest = &testError{"simulated fn error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
