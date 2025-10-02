package serial

import (
	"context"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// TestSignalMaskToTIOCM tests the signal mask conversion
func TestSignalMaskToTIOCM(t *testing.T) {
	tests := []struct {
		name     string
		mask     SignalMask
		expected int
	}{
		{
			name:     "CTS only",
			mask:     SignalCTS,
			expected: unix.TIOCM_CTS,
		},
		{
			name:     "DSR only",
			mask:     SignalDSR,
			expected: unix.TIOCM_DSR,
		},
		{
			name:     "RI only",
			mask:     SignalRI,
			expected: unix.TIOCM_RI,
		},
		{
			name:     "DCD only",
			mask:     SignalDCD,
			expected: unix.TIOCM_CAR,
		},
		{
			name:     "Multiple signals",
			mask:     SignalCTS | SignalDSR,
			expected: unix.TIOCM_CTS | unix.TIOCM_DSR,
		},
		{
			name:     "All signals",
			mask:     SignalCTS | SignalDSR | SignalRI | SignalDCD,
			expected: unix.TIOCM_CTS | unix.TIOCM_DSR | unix.TIOCM_RI | unix.TIOCM_CAR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := signalMaskToTIOCM(tt.mask)
			if result != tt.expected {
				t.Errorf("signalMaskToTIOCM(%v) = %v, want %v", tt.mask, result, tt.expected)
			}
		})
	}
}

// TestDetectSignalChanges tests signal change detection
func TestDetectSignalChanges(t *testing.T) {
	tests := []struct {
		name      string
		oldStatus int
		newStatus int
		expected  SignalMask
	}{
		{
			name:      "No change",
			oldStatus: unix.TIOCM_CTS | unix.TIOCM_DSR,
			newStatus: unix.TIOCM_CTS | unix.TIOCM_DSR,
			expected:  0,
		},
		{
			name:      "CTS changed",
			oldStatus: 0,
			newStatus: unix.TIOCM_CTS,
			expected:  SignalCTS,
		},
		{
			name:      "DSR changed",
			oldStatus: 0,
			newStatus: unix.TIOCM_DSR,
			expected:  SignalDSR,
		},
		{
			name:      "RI changed",
			oldStatus: 0,
			newStatus: unix.TIOCM_RI,
			expected:  SignalRI,
		},
		{
			name:      "DCD changed",
			oldStatus: 0,
			newStatus: unix.TIOCM_CAR,
			expected:  SignalDCD,
		},
		{
			name:      "Multiple signals changed",
			oldStatus: 0,
			newStatus: unix.TIOCM_CTS | unix.TIOCM_DSR,
			expected:  SignalCTS | SignalDSR,
		},
		{
			name:      "Signal went low",
			oldStatus: unix.TIOCM_CTS,
			newStatus: 0,
			expected:  SignalCTS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectSignalChanges(tt.oldStatus, tt.newStatus)
			if result != tt.expected {
				t.Errorf("detectSignalChanges(%v, %v) = %v, want %v", tt.oldStatus, tt.newStatus, result, tt.expected)
			}
		})
	}
}

// TestWithInitialRTS tests the initial RTS configuration
func TestWithInitialRTS(t *testing.T) {
	tests := []struct {
		name     string
		state    bool
		expected bool
	}{
		{
			name:     "Initial RTS true",
			state:    true,
			expected: true,
		},
		{
			name:     "Initial RTS false",
			state:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			opt := WithInitialRTS(tt.state)
			err := opt(&config)
			if err != nil {
				t.Errorf("WithInitialRTS(%v) returned error: %v", tt.state, err)
			}
			if config.InitialRTS == nil {
				t.Errorf("WithInitialRTS(%v) did not set InitialRTS", tt.state)
			} else if *config.InitialRTS != tt.expected {
				t.Errorf("WithInitialRTS(%v) set InitialRTS to %v, want %v", tt.state, *config.InitialRTS, tt.expected)
			}
		})
	}
}

// TestWithInitialDTR tests the initial DTR configuration
func TestWithInitialDTR(t *testing.T) {
	tests := []struct {
		name     string
		state    bool
		expected bool
	}{
		{
			name:     "Initial DTR true",
			state:    true,
			expected: true,
		},
		{
			name:     "Initial DTR false",
			state:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			opt := WithInitialDTR(tt.state)
			err := opt(&config)
			if err != nil {
				t.Errorf("WithInitialDTR(%v) returned error: %v", tt.state, err)
			}
			if config.InitialDTR == nil {
				t.Errorf("WithInitialDTR(%v) did not set InitialDTR", tt.state)
			} else if *config.InitialDTR != tt.expected {
				t.Errorf("WithInitialDTR(%v) set InitialDTR to %v, want %v", tt.state, *config.InitialDTR, tt.expected)
			}
		})
	}
}

// TestWaitForSignalChangeInvalidMask tests error handling for invalid signal masks
func TestWaitForSignalChangeInvalidMask(t *testing.T) {
	// This test doesn't require actual port, just tests the validation logic
	// We can't test without a real port, but we can verify the error type matches

	// Create a mock closed port to test error handling
	p := &port{closed: true}

	_, _, err := p.WaitForSignalChange(0, time.Second)
	if err != ErrInvalidSignalMask {
		t.Errorf("WaitForSignalChange(0, ...) error = %v, want %v", err, ErrInvalidSignalMask)
	}

	ctx := context.Background()
	_, _, err = p.WaitForSignalChangeContext(ctx, 0)
	if err != ErrInvalidSignalMask {
		t.Errorf("WaitForSignalChangeContext(ctx, 0) error = %v, want %v", err, ErrInvalidSignalMask)
	}
}

// TestModemSignalsOnClosedPort tests that methods return appropriate errors on closed ports
func TestModemSignalsOnClosedPort(t *testing.T) {
	p := &port{closed: true}

	t.Run("GetModemSignals", func(t *testing.T) {
		_, err := p.GetModemSignals()
		if err != ErrPortClosed {
			t.Errorf("GetModemSignals() on closed port error = %v, want %v", err, ErrPortClosed)
		}
	})

	t.Run("SetRTS", func(t *testing.T) {
		err := p.SetRTS(true)
		if err != ErrPortClosed {
			t.Errorf("SetRTS() on closed port error = %v, want %v", err, ErrPortClosed)
		}
	})

	t.Run("GetRTS", func(t *testing.T) {
		_, err := p.GetRTS()
		if err != ErrPortClosed {
			t.Errorf("GetRTS() on closed port error = %v, want %v", err, ErrPortClosed)
		}
	})

	t.Run("WaitForSignalChange", func(t *testing.T) {
		_, _, err := p.WaitForSignalChange(SignalCTS, time.Second)
		if err != ErrPortClosed {
			t.Errorf("WaitForSignalChange() on closed port error = %v, want %v", err, ErrPortClosed)
		}
	})

	t.Run("WaitForSignalChangeContext", func(t *testing.T) {
		ctx := context.Background()
		_, _, err := p.WaitForSignalChangeContext(ctx, SignalCTS)
		if err != ErrPortClosed {
			t.Errorf("WaitForSignalChangeContext() on closed port error = %v, want %v", err, ErrPortClosed)
		}
	})
}

// TestWaitForSignalChangeContextCancellation tests context cancellation
func TestWaitForSignalChangeContextCancellation(t *testing.T) {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &port{closed: true}
	_, _, err := p.WaitForSignalChangeContext(ctx, SignalCTS)

	// Should return either ErrPortClosed or context.Canceled
	// Both are acceptable since we're checking a closed port first
	if err != ErrPortClosed && err != context.Canceled {
		t.Errorf("WaitForSignalChangeContext() with cancelled context error = %v, want %v or %v",
			err, ErrPortClosed, context.Canceled)
	}
}
