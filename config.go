package serial

import "time"

// Config holds the configuration for a serial port
type Config struct {
	BaudRate        int
	DataBits        int
	StopBits        int
	Parity          Parity
	FlowControl     FlowControl
	CTSTimeout      time.Duration
	ReadBufferSize  int
	WriteBufferSize int
}

// Option is a functional option for configuring a serial port
type Option func(*Config) error

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		BaudRate:        115200,
		DataBits:        8,
		StopBits:        1,
		Parity:          ParityNone,
		FlowControl:     FlowControlNone,
		CTSTimeout:      500 * time.Millisecond,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
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

// WithReadBufferSize sets the read buffer size
func WithReadBufferSize(size int) Option {
	return func(c *Config) error {
		if size <= 0 {
			return ErrInvalidConfig
		}
		c.ReadBufferSize = size
		return nil
	}
}

// WithWriteBufferSize sets the write buffer size
func WithWriteBufferSize(size int) Option {
	return func(c *Config) error {
		if size <= 0 {
			return ErrInvalidConfig
		}
		c.WriteBufferSize = size
		return nil
	}
}
