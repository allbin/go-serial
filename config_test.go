package serial

import (
	"testing"
	"time"
)

func TestWithReadTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"0ms (non-blocking)", 0, false},
		{"100ms (valid)", 100 * time.Millisecond, false},
		{"500ms (valid)", 500 * time.Millisecond, false},
		{"2500ms (valid)", 2500 * time.Millisecond, false},
		{"25500ms (max)", 25500 * time.Millisecond, false},
		{"150ms (not multiple of 100ms)", 150 * time.Millisecond, true},
		{"250ns (not multiple of 100ms)", 250 * time.Nanosecond, true},
		{"25600ms (exceeds max)", 25600 * time.Millisecond, true},
		{"-100ms (negative)", -100 * time.Millisecond, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			opt := WithReadTimeout(tt.timeout)
			err := opt(&config)
			if (err != nil) != tt.wantErr {
				t.Errorf("WithReadTimeout(%v) error = %v, wantErr %v", tt.timeout, err, tt.wantErr)
			}
			if err == nil && config.ReadTimeout != tt.timeout {
				t.Errorf("ReadTimeout = %v, want %v", config.ReadTimeout, tt.timeout)
			}
		})
	}
}
