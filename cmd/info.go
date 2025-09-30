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

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info <port>",
	Short: "Display detailed information about a serial port",
	Long: `Display detailed information about a serial port including USB metadata.

Examples:
  serial info /dev/ttyUSB0
  serial info /dev/ttyACM0

For USB devices, this displays vendor/product IDs, serial numbers, interface
numbers, and other USB-specific metadata extracted from sysfs.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		info, err := serial.GetPortInfo(portPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting port info: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Port Information: %s\n\n", info.Path)
		fmt.Printf("  Name:        %s\n", info.Name)
		fmt.Printf("  Description: %s\n", info.Description)

		// USB Device Information
		if info.VendorID != "" || info.ProductID != "" {
			fmt.Println("\nUSB Device Information:")
			if info.VendorID != "" {
				fmt.Printf("  Vendor ID:    %s\n", info.VendorID)
			}
			if info.ProductID != "" {
				fmt.Printf("  Product ID:   %s\n", info.ProductID)
			}
			if info.SerialNumber != "" {
				fmt.Printf("  Serial:       %s\n", info.SerialNumber)
			}
			if info.InterfaceNumber != "" {
				fmt.Printf("  Interface:    %s\n", info.InterfaceNumber)
			}
			if info.BusNumber != "" {
				fmt.Printf("  Bus:          %s\n", info.BusNumber)
			}
			if info.DeviceNumber != "" {
				fmt.Printf("  Device:       %s\n", info.DeviceNumber)
			}
			if info.Manufacturer != "" {
				fmt.Printf("  Manufacturer: %s\n", info.Manufacturer)
			}
			if info.Product != "" {
				fmt.Printf("  Product:      %s\n", info.Product)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
