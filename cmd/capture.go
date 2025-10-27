/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/allbin/go-serial"
	"github.com/spf13/cobra"
)

// captureCmd represents the capture command
var captureCmd = &cobra.Command{
	Use:   "capture <port> <output-file>",
	Short: "Capture serial data to a file",
	Long: `Capture incoming serial data to a file for later parsing.

Reads ASCII data from the specified serial port and writes it directly to
the output file. Runs continuously until interrupted (Ctrl+C).

The output file is opened in append mode, allowing you to resume captures
without overwriting existing data.

Example usage:
  serial capture /dev/ttyUSB0 data.log
  serial capture /dev/ttyUSB0 output.txt --baud 9600
  serial capture /dev/ttyUSB0 capture.log --console
  serial capture /dev/ttyUSB0 capture.log --flow-control cts --initial-rts -c`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]
		outputPath := args[1]

		// Get flags
		baudRate, _ := cmd.Flags().GetInt("baud")
		flowControl, _ := cmd.Flags().GetString("flow-control")
		initialRTS, _ := cmd.Flags().GetBool("initial-rts")
		bufferSize, _ := cmd.Flags().GetInt("buffer")
		showConsole, _ := cmd.Flags().GetBool("console")

		// Configure port options
		opts := []serial.Option{
			serial.WithBaudRate(baudRate),
		}

		switch strings.ToLower(flowControl) {
		case "cts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlCTS))
			if initialRTS {
				opts = append(opts, serial.WithInitialRTS(true))
			}
		case "rtscts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlRTSCTS))
			if initialRTS {
				opts = append(opts, serial.WithInitialRTS(true))
			}
		}

		if err := runCapture(portPath, outputPath, bufferSize, showConsole, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(captureCmd)

	captureCmd.Flags().IntP("baud", "b", 115200, "Baud rate")
	captureCmd.Flags().StringP("flow-control", "f", "none", "Flow control: none, cts, rtscts")
	captureCmd.Flags().Bool("initial-rts", false, "Assert RTS on port open")
	captureCmd.Flags().Int("buffer", 4096, "Read buffer size")
	captureCmd.Flags().BoolP("console", "c", false, "Display incoming data on console while capturing")
}

func runCapture(portPath, outputPath string, bufferSize int, showConsole bool, opts ...serial.Option) error {
	// Open serial port
	port, err := serial.Open(portPath, opts...)
	if err != nil {
		return fmt.Errorf("failed to open port: %w", err)
	}
	defer port.Close()

	// Open output file in append mode
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// Setup signal handling for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\nReceived interrupt signal, shutting down...\n")
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "Capturing data from %s to %s\n", portPath, outputPath)
	if showConsole {
		fmt.Fprintf(os.Stderr, "Console display enabled\n")
	}
	fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

	// Read and write loop
	buffer := make([]byte, bufferSize)
	bytesWritten := int64(0)
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			duration := time.Since(startTime)
			fmt.Fprintf(os.Stderr, "\nCapture complete: %d bytes written in %v\n", bytesWritten, duration.Round(time.Millisecond))
			return nil
		default:
			n, err := port.ReadContext(ctx, buffer)
			if err != nil {
				if ctx.Err() != nil {
					// Context cancelled, clean shutdown
					return nil
				}
				return fmt.Errorf("read error: %w", err)
			}

			if n > 0 {
				written, err := file.Write(buffer[:n])
				if err != nil {
					return fmt.Errorf("write error: %w", err)
				}
				bytesWritten += int64(written)

				// Display on console if enabled
				if showConsole {
					os.Stdout.Write(buffer[:n])
				}
			}
		}
	}
}
