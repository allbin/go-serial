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

// dtrCmd represents the dtr command
var dtrCmd = &cobra.Command{
	Use:   "dtr <port> <state>",
	Short: "Control DTR (Data Terminal Ready) signal",
	Long: `Manually set the DTR (Data Terminal Ready) signal state.

The DTR signal indicates that the terminal is ready for communication.

Examples:
  serial dtr /dev/ttyUSB0 high
  serial dtr /dev/ttyUSB0 low
  serial dtr /dev/ttyUSB0 on
  serial dtr /dev/ttyUSB0 off

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

		err = port.SetDTR(state)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting DTR: %v\n", err)
			os.Exit(1)
		}

		// Verify the state was set
		currentState, err := port.GetDTR()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not verify DTR state: %v\n", err)
		}

		fmt.Printf("DTR set to %s on %s\n", formatSignalState(currentState), portPath)
	},
}

func init() {
	rootCmd.AddCommand(dtrCmd)
}
