/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/allbin/go-serial"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// sendCmd represents the send command
var sendCmd = &cobra.Command{
	Use:   "send [data] <port>",
	Short: "Send data to a serial port",
	Long: `Send data to a serial port with configurable options.

This command sends data to the specified serial port. Data can be provided as:
- Command line argument: send "Hello World" /dev/ttyUSB0
- From stdin (pipe): echo "test data" | serial send /dev/ttyUSB0
- Interactive mode: serial send /dev/ttyUSB0 (prompts for input)

Features include:
- Multiple input methods (argument, stdin, interactive)
- Configurable baud rate and flow control
- Automatic line endings (--newline flag)
- Hex input support (--hex flag)
- Connection status feedback with styled output

Example usage:
  serial send "Hello World" /dev/ttyUSB0
  serial send "AT+GMR" /dev/ttyUSB0 --newline
  echo "test" | serial send /dev/ttyUSB0
  serial send /dev/ttyUSB0  # Interactive mode`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var data string
		var portPath string

		// Parse arguments: either "send data port" or "send port"
		if len(args) == 1 {
			portPath = args[0]
			// Check if we have stdin data
			stat, err := os.Stdin.Stat()
			if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
				// No pipe input, use interactive mode
				data = promptForData()
			} else {
				// Read from stdin
				stdinData, err := io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
					os.Exit(1)
				}
				data = strings.TrimRight(string(stdinData), "\r\n")
			}
		} else {
			data = args[0]
			portPath = args[1]
		}

		// Get flags
		baudRate, _ := cmd.Flags().GetInt("baud")
		flowControl, _ := cmd.Flags().GetString("flow-control")
		addNewline, _ := cmd.Flags().GetBool("newline")
		hexMode, _ := cmd.Flags().GetBool("hex")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		initialRTS, _ := cmd.Flags().GetBool("initial-rts")

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

		// Process data based on flags
		if hexMode {
			processedData, err := parseHexString(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid hex data: %v\n", err)
				os.Exit(1)
			}
			data = processedData
		}

		if addNewline && !hexMode {
			data += "\n"
		}

		// Send the data
		if err := sendData(portPath, data, timeout, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)

	// Add flags for serial configuration and send options
	sendCmd.Flags().IntP("baud", "b", 115200, "Baud rate (default: 115200)")
	sendCmd.Flags().StringP("flow-control", "f", "none", "Flow control: none, cts, rtscts (default: none)")
	sendCmd.Flags().BoolP("newline", "n", false, "Add newline character to the end of data")
	sendCmd.Flags().BoolP("hex", "x", false, "Interpret data as hexadecimal (e.g., '48656c6c6f' for 'Hello')")
	sendCmd.Flags().DurationP("timeout", "t", 5*time.Second, "Timeout for sending data (default: 5s)")
	sendCmd.Flags().Bool("initial-rts", false, "Assert RTS on port open (required for CTS flow control)")
}

func promptForData() string {
	// Styled prompt
	promptStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))

	fmt.Print(promptStyle.Render("Enter data to send: "))

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

func parseHexString(hexStr string) (string, error) {
	// Remove common hex prefixes and whitespace
	hexStr = strings.ReplaceAll(hexStr, " ", "")
	hexStr = strings.ReplaceAll(hexStr, "0x", "")
	hexStr = strings.ReplaceAll(hexStr, "0X", "")

	if len(hexStr)%2 != 0 {
		return "", fmt.Errorf("hex string must have even length")
	}

	var result strings.Builder
	for i := 0; i < len(hexStr); i += 2 {
		hexByte := hexStr[i : i+2]
		var b byte
		if _, err := fmt.Sscanf(hexByte, "%x", &b); err != nil {
			return "", fmt.Errorf("invalid hex byte '%s': %v", hexByte, err)
		}
		result.WriteByte(b)
	}

	return result.String(), nil
}

func sendData(portPath, data string, timeout time.Duration, opts ...serial.Option) error {
	// Styled output
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("40")).
		Bold(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	// Show connection attempt
	fmt.Printf("%s Opening %s...\n", infoStyle.Render("⚡"), portPath)

	// Open serial port
	port, err := serial.Open(portPath, opts...)
	if err != nil {
		return fmt.Errorf("%s %v", errorStyle.Render("✗"), err)
	}
	defer port.Close()

	fmt.Printf("%s Connected successfully\n", successStyle.Render("✓"))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Send data
	fmt.Printf("%s Sending %d bytes...\n", infoStyle.Render("📤"), len(data))

	n, err := port.WriteContext(ctx, []byte(data))
	if err != nil {
		return fmt.Errorf("%s failed to send data: %v", errorStyle.Render("✗"), err)
	}

	fmt.Printf("%s Successfully sent %d bytes\n", successStyle.Render("✓"), n)

	// Show data preview (first 50 chars)
	preview := data
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}
	// Replace non-printable characters for display
	preview = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return '·'
		}
		return r
	}, preview)

	fmt.Printf("%s Data: %s\n", infoStyle.Render("📋"), preview)

	return nil
}
