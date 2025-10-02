/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/allbin/go-serial"
	"github.com/spf13/cobra"
)

// signalsCmd represents the signals command
var signalsCmd = &cobra.Command{
	Use:   "signals <port>",
	Short: "Display current modem signal states",
	Long: `Display the current state of all modem control signals.

Shows the state of CTS, DSR, RI, DCD, RTS, and DTR signals for the specified port.

Examples:
  serial signals /dev/ttyUSB0
  serial signals /dev/ttyACM0

Signal meanings:
  CTS - Clear To Send (input)
  DSR - Data Set Ready (input)
  RI  - Ring Indicator (input)
  DCD - Data Carrier Detect (input)
  RTS - Request To Send (output)
  DTR - Data Terminal Ready (output)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		port, err := serial.Open(portPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening port: %v\n", err)
			os.Exit(1)
		}
		defer port.Close()

		signals, err := port.GetModemSignals()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading modem signals: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Modem Signals for %s:\n\n", portPath)
		fmt.Printf("  CTS (Clear To Send):       %s\n", formatSignalState(signals.CTS))
		fmt.Printf("  DSR (Data Set Ready):      %s\n", formatSignalState(signals.DSR))
		fmt.Printf("  RI  (Ring Indicator):      %s\n", formatSignalState(signals.RI))
		fmt.Printf("  DCD (Data Carrier Detect): %s\n", formatSignalState(signals.DCD))
		fmt.Printf("  RTS (Request To Send):     %s\n", formatSignalState(signals.RTS))
		fmt.Printf("  DTR (Data Terminal Ready): %s\n", formatSignalState(signals.DTR))
	},
}

func formatSignalState(state bool) string {
	if state {
		return "HIGH"
	}
	return "LOW"
}

func init() {
	rootCmd.AddCommand(signalsCmd)
}
