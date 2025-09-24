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
	BaudRate          int
	DataBits          int
	StopBits          int
	Parity            Parity
	FlowControl       FlowControl
	CTSTimeout        time.Duration
	ReadTimeoutTenths int       // VTIME setting in tenths of seconds (0-255)
	WriteMode         WriteMode // Controls write synchronization behavior
}

// Option is a functional option for configuring a serial port
type Option func(*Config) error

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		BaudRate:          115200,
		DataBits:          8,
		StopBits:          1,
		Parity:            ParityNone,
		FlowControl:       FlowControlNone,
		CTSTimeout:        500 * time.Millisecond,
		ReadTimeoutTenths: 25, // 2.5 seconds - match reference 250ms * 10
		WriteMode:         WriteModeBuffered,
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

// WithReadTimeout sets the read timeout in tenths of seconds (VTIME)
func WithReadTimeout(tenths int) Option {
	return func(c *Config) error {
		if tenths < 0 || tenths > 255 {
			return ErrInvalidConfig
		}
		c.ReadTimeoutTenths = tenths
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

