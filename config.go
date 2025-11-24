package serial

import "time"

// WriteMode represents the write synchronization mode
type WriteMode int

const (
	WriteModeBuffered WriteMode = iota // Default: kernel buffers writes
	WriteModeSynced                    // O_SYNC: writes block until hardware transmission
)

// Config holds the configuration for a serial port
type Config struct {
	BaudRate    int
	DataBits    int
	StopBits    int
	Parity      Parity
	FlowControl FlowControl
	CTSTimeout  time.Duration
	ReadTimeout time.Duration // VTIME setting (max 25.5 seconds, rounded to deciseconds)
	WriteMode   WriteMode     // Controls write synchronization behavior
	InitialRTS  *bool         // Initial RTS state (nil = hardware default)
	InitialDTR  *bool         // Initial DTR state (nil = hardware default)
}

// Option is a functional option for configuring a serial port
type Option func(*Config) error

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		BaudRate:    115200,
		DataBits:    8,
		StopBits:    1,
		Parity:      ParityNone,
		FlowControl: FlowControlNone,
		CTSTimeout:  60 * time.Second,        // Neocortec reference default (matches NcConstants.DefaultCtsTimeOutMs)
		ReadTimeout: 2500 * time.Millisecond, // 2.5 seconds - match reference 250ms * 10
		WriteMode:   WriteModeBuffered,
	}
}

// WithBaudRate sets the baud rate
func WithBaudRate(rate int) Option {
	return func(c *Config) error {
		if _, err := getBaudRate(rate); err != nil {
			return err
		}
		c.BaudRate = rate
		return nil
	}
}

// WithDataBits sets the number of data bits (5, 6, 7, or 8)
func WithDataBits(bits int) Option {
	return func(c *Config) error {
		if bits < 5 || bits > 8 {
			return ErrInvalidConfig
		}
		c.DataBits = bits
		return nil
	}
}

// WithStopBits sets the number of stop bits (1 or 2)
func WithStopBits(bits int) Option {
	return func(c *Config) error {
		if bits != 1 && bits != 2 {
			return ErrInvalidConfig
		}
		c.StopBits = bits
		return nil
	}
}

// WithParity sets the parity mode
func WithParity(parity Parity) Option {
	return func(c *Config) error {
		c.Parity = parity
		return nil
	}
}

// WithFlowControl sets the flow control mode
func WithFlowControl(fc FlowControl) Option {
	return func(c *Config) error {
		c.FlowControl = fc
		return nil
	}
}

// WithCTSTimeout sets the CTS timeout for flow control
func WithCTSTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout < 0 {
			return ErrInvalidConfig
		}
		c.CTSTimeout = timeout
		return nil
	}
}

// WithReadTimeout sets the read timeout (VTIME)
// Maximum is 25.5 seconds (255 deciseconds).
// Must be a multiple of 100ms (1 decisecond).
func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout < 0 || timeout > 255*100*time.Millisecond {
			return ErrInvalidConfig
		}
		if timeout%(100*time.Millisecond) != 0 {
			return ErrInvalidConfig
		}
		c.ReadTimeout = timeout
		return nil
	}
}

// WithWriteMode sets the write synchronization mode
func WithWriteMode(mode WriteMode) Option {
	return func(c *Config) error {
		c.WriteMode = mode
		return nil
	}
}

// WithSyncWrite enables synchronous writes (O_SYNC) for guaranteed transmission
func WithSyncWrite() Option {
	return func(c *Config) error {
		c.WriteMode = WriteModeSynced
		return nil
	}
}

// WithInitialRTS sets initial RTS state when opening port
func WithInitialRTS(state bool) Option {
	return func(c *Config) error {
		c.InitialRTS = &state
		return nil
	}
}

// WithInitialDTR sets initial DTR state when opening port
func WithInitialDTR(state bool) Option {
	return func(c *Config) error {
		c.InitialDTR = &state
		return nil
	}
}
