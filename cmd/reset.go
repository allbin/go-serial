/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/allbin/go-serial"
	"github.com/spf13/cobra"
)

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset <port|serial>",
	Short: "Reset a USB serial device",
	Long: `Perform a USB-level reset on a serial device. This can recover devices
that are hung or unresponsive without physically unplugging them.

The device will re-enumerate after reset, which may cause the port path
to change (e.g., /dev/ttyUSB0 might become /dev/ttyUSB1). Use serial
numbers to reliably identify devices after reset.

Requirements:
- usbreset utility must be installed (from usbutils package)
- Root/sudo permissions required for USB operations

Examples:
  sudo serial reset /dev/ttyUSB0          # Reset by port path
  sudo serial reset --serial NC7ILXW1    # Reset by serial number`,
	Args: func(cmd *cobra.Command, args []string) error {
		serialFlag, _ := cmd.Flags().GetString("serial")
		if serialFlag == "" && len(args) != 1 {
			return errors.New("requires either a port path argument or --serial flag")
		}
		if serialFlag != "" && len(args) > 0 {
			return errors.New("cannot specify both port path and --serial flag")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Check if usbreset is available
		if !serial.IsUSBResetAvailable() {
			fmt.Fprintln(os.Stderr, "Error: usbreset utility not available")
			fmt.Fprintln(os.Stderr, "Install with: sudo apt-get install usbutils")
			os.Exit(1)
		}

		serialFlag, _ := cmd.Flags().GetString("serial")

		var err error
		if serialFlag != "" {
			// Reset by serial number
			fmt.Printf("Resetting USB device with serial: %s\n", serialFlag)
			err = serial.ResetUSBDeviceBySerial(serialFlag)
		} else {
			// Reset by port path
			portPath := args[0]
			fmt.Printf("Resetting USB device: %s\n", portPath)
			err = serial.ResetUSBDevice(portPath)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			if errors.Is(err, serial.ErrUSBInfoNotAvailable) {
				fmt.Fprintln(os.Stderr, "This device does not appear to be a USB device")
			}
			os.Exit(1)
		}

		fmt.Println("USB device reset successfully")
		fmt.Println("Device will re-enumerate (port path may change)")
		fmt.Println("\nUse 'serial list --table' to see updated device list")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)

	resetCmd.Flags().StringP("serial", "s", "", "Reset device by serial number")
}
