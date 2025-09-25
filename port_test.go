package serial

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BaudRate != 115200 {
		t.Errorf("Expected BaudRate 115200, got %d", config.BaudRate)
	}

	if config.DataBits != 8 {
		t.Errorf("Expected DataBits 8, got %d", config.DataBits)
	}

	if config.StopBits != 1 {
		t.Errorf("Expected StopBits 1, got %d", config.StopBits)
	}

	if config.Parity != ParityNone {
		t.Errorf("Expected Parity None, got %v", config.Parity)
	}

	if config.FlowControl != FlowControlNone {
		t.Errorf("Expected FlowControl None, got %v", config.FlowControl)
	}

	if config.CTSTimeout != 500*time.Millisecond {
		t.Errorf("Expected CTSTimeout 500ms, got %v", config.CTSTimeout)
	}
}

func TestFunctionalOptions(t *testing.T) {
	config := DefaultConfig()

	// Test WithBaudRate
	err := WithBaudRate(9600)(&config)
	if err != nil {
		t.Errorf("WithBaudRate failed: %v", err)
	}
	if config.BaudRate != 9600 {
		t.Errorf("Expected BaudRate 9600, got %d", config.BaudRate)
	}

	// Test WithDataBits
	err = WithDataBits(7)(&config)
	if err != nil {
		t.Errorf("WithDataBits failed: %v", err)
	}
	if config.DataBits != 7 {
		t.Errorf("Expected DataBits 7, got %d", config.DataBits)
	}

	// Test WithStopBits
	err = WithStopBits(2)(&config)
	if err != nil {
		t.Errorf("WithStopBits failed: %v", err)
	}
	if config.StopBits != 2 {
		t.Errorf("Expected StopBits 2, got %d", config.StopBits)
	}

	// Test WithParity
	err = WithParity(ParityEven)(&config)
	if err != nil {
		t.Errorf("WithParity failed: %v", err)
	}
	if config.Parity != ParityEven {
		t.Errorf("Expected Parity Even, got %v", config.Parity)
	}

	// Test WithFlowControl
	err = WithFlowControl(FlowControlCTS)(&config)
	if err != nil {
		t.Errorf("WithFlowControl failed: %v", err)
	}
	if config.FlowControl != FlowControlCTS {
		t.Errorf("Expected FlowControl CTS, got %v", config.FlowControl)
	}
}

func TestInvalidBaudRate(t *testing.T) {
	config := DefaultConfig()
	err := WithBaudRate(123456)(&config)
	if err == nil {
		t.Error("Expected error for invalid baud rate")
	}
	if err != ErrInvalidBaudRate {
		t.Errorf("Expected ErrInvalidBaudRate, got %v", err)
	}
}

func TestInvalidDataBits(t *testing.T) {
	config := DefaultConfig()
	err := WithDataBits(9)(&config)
	if err == nil {
		t.Error("Expected error for invalid data bits")
	}
	if err != ErrInvalidConfig {
		t.Errorf("Expected ErrInvalidConfig, got %v", err)
	}
}

func TestInvalidStopBits(t *testing.T) {
	config := DefaultConfig()
	err := WithStopBits(3)(&config)
	if err == nil {
		t.Error("Expected error for invalid stop bits")
	}
	if err != ErrInvalidConfig {
		t.Errorf("Expected ErrInvalidConfig, got %v", err)
	}
}

func TestGetBaudRate(t *testing.T) {
	tests := []struct {
		input    int
		hasError bool
	}{
		{115200, false},
		{9600, false},
		{57600, false},
		{123456, true}, // Invalid baud rate
	}

	for _, test := range tests {
		result, err := getBaudRate(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for baud rate %d", test.input)
			}
			if err != ErrInvalidBaudRate {
				t.Errorf("Expected ErrInvalidBaudRate for %d, got %v", test.input, err)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for baud rate %d: %v", test.input, err)
			}
			if result == 0 {
				t.Errorf("Got zero result for valid baud rate %d", test.input)
			}
		}
	}
}

func TestOpenNonExistentDevice(t *testing.T) {
	_, err := Open("/dev/nonexistent")
	if err == nil {
		t.Error("Expected error when opening non-existent device")
	}
	if err != ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

// This test requires running with appropriate permissions and available device
func TestContextTimeout(t *testing.T) {
	// Test context timeout behavior without actual hardware
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Microsecond)

	// Create a mock port for testing context behavior
	port := &Port{closed: false}

	// Test ReadContext with expired context
	buf := make([]byte, 10)
	_, err := port.ReadContext(ctx, buf)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Test WriteContext with expired context
	data := []byte("test")
	_, err = port.WriteContext(ctx, data)
	if err == nil {
		t.Error("Expected timeout error")
	}
}
