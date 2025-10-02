/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/allbin/go-serial"
	"github.com/spf13/cobra"
)

var rtsState string

// rtsCmd represents the rts command
var rtsCmd = &cobra.Command{
	Use:   "rts <port> <state>",
	Short: "Control RTS (Request To Send) signal",
	Long: `Manually set the RTS (Request To Send) signal state.

The RTS signal can be used for software flow control or custom signaling.

Examples:
  serial rts /dev/ttyUSB0 high
  serial rts /dev/ttyUSB0 low
  serial rts /dev/ttyUSB0 on
  serial rts /dev/ttyUSB0 off

Valid states: high, low, on, off, true, false, 1, 0`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]
		stateArg := args[1]

		state, err := parseSignalState(stateArg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		port, err := serial.Open(portPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening port: %v\n", err)
			os.Exit(1)
		}
		defer port.Close()

		err = port.SetRTS(state)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting RTS: %v\n", err)
			os.Exit(1)
		}

		// Verify the state was set
		currentState, err := port.GetRTS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not verify RTS state: %v\n", err)
		}

		fmt.Printf("RTS set to %s on %s\n", formatSignalState(currentState), portPath)
	},
}

func parseSignalState(state string) (bool, error) {
	switch strings.ToLower(state) {
	case "high", "on", "true", "1":
		return true, nil
	case "low", "off", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid state: %s (valid: high, low, on, off, true, false, 1, 0)", state)
	}
}

func init() {
	rootCmd.AddCommand(rtsCmd)
}
