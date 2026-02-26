package common

import (
	"context"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestListenSysExit(t *testing.T) {
	// Create a logger for testing
	logger := zap.NewNop()

	// Test SIGTERM signal
	t.Run("SIGTERM signal", func(t *testing.T) {
		// Create a new context for this test
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// Start ListenSysExit
		go ListenSysExit(logger, testCancel)

		// Give some time for the signal handler to be set up
		time.Sleep(100 * time.Millisecond)

		// Send SIGTERM signal to the current process
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		if err != nil {
			t.Fatalf("Failed to send SIGTERM signal: %v", err)
		}

		// Wait for the context to be cancelled
		select {
		case <-testCtx.Done():
			// Context was cancelled as expected
			t.Log("Context cancelled successfully after SIGTERM")
		case <-time.After(2 * time.Second):
			t.Fatal("Context was not cancelled within 2 seconds after SIGTERM")
		}
	})

	// Test SIGINT signal
	t.Run("SIGINT signal", func(t *testing.T) {
		// Create a new context for this test
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// Start ListenSysExit
		go ListenSysExit(logger, testCancel)

		// Give some time for the signal handler to be set up
		time.Sleep(100 * time.Millisecond)

		// Send SIGINT signal to the current process
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		if err != nil {
			t.Fatalf("Failed to send SIGINT signal: %v", err)
		}

		// Wait for the context to be cancelled
		select {
		case <-testCtx.Done():
			// Context was cancelled as expected
			t.Log("Context cancelled successfully after SIGINT")
		case <-time.After(2 * time.Second):
			t.Fatal("Context was not cancelled within 2 seconds after SIGINT")
		}
	})

	// Test that the function doesn't exit immediately
	t.Run("no signal sent", func(t *testing.T) {
		// Create a new context for this test
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		// Start ListenSysExit
		go ListenSysExit(logger, testCancel)

		// Wait a short time and verify context is not cancelled
		select {
		case <-testCtx.Done():
			t.Fatal("Context was unexpectedly cancelled without signal")
		case <-time.After(500 * time.Millisecond):
			// Context was not cancelled as expected
			t.Log("Context remained active as expected when no signal was sent")
		}
	})
}

// TestListenSysExitConcurrent tests that multiple calls to ListenSysExit work correctly
func TestListenSysExitConcurrent(t *testing.T) {
	logger := zap.NewNop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start multiple ListenSysExit goroutines
	for i := 0; i < 3; i++ {
		go ListenSysExit(logger, cancel)
	}

	// Give some time for all signal handlers to be set up
	time.Sleep(100 * time.Millisecond)

	// Send SIGTERM signal
	err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	if err != nil {
		t.Fatalf("Failed to send SIGTERM signal: %v", err)
	}

	// Wait for the context to be cancelled
	select {
	case <-ctx.Done():
		t.Log("Context cancelled successfully with multiple ListenSysExit goroutines")
	case <-time.After(2 * time.Second):
		t.Fatal("Context was not cancelled within 2 seconds")
	}
}
