/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mdjarv/serial"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available serial ports",
	Long: `List all available serial ports on the system.

This command scans for communication-capable serial devices including:
- USB serial adapters (ttyUSB*)
- USB CDC/ACM devices (ttyACM*)
- Standard serial ports (ttyS*)
- ARM/Raspberry Pi ports (ttyAMA*)
- And other platform-specific serial devices

Virtual terminals and pseudo-terminals are excluded from the listing.`,
	Run: func(cmd *cobra.Command, args []string) {
		ports, err := serial.ListPorts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing ports: %v\n", err)
			os.Exit(1)
		}

		if len(ports) == 0 {
			fmt.Println("No serial ports found")
			return
		}

		// Get filter flag
		filterType, _ := cmd.Flags().GetString("filter")
		tableFormat, _ := cmd.Flags().GetBool("table")

		// Filter ports if requested
		filteredPorts := filterPorts(ports, filterType)

		if len(filteredPorts) == 0 {
			if filterType != "" {
				fmt.Printf("No serial ports found matching filter: %s\n", filterType)
			} else {
				fmt.Println("No serial ports found")
			}
			return
		}

		if tableFormat {
			renderTable(filteredPorts)
		} else {
			renderSimple(filteredPorts)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Add flags for filtering and table format
	listCmd.Flags().StringP("filter", "f", "", "Filter by port type: usb, standard, arm, all")
	listCmd.Flags().BoolP("table", "t", false, "Display output in a styled table format")
}

// filterPorts filters the port list based on the specified filter type
func filterPorts(ports []string, filterType string) []string {
	if filterType == "" || filterType == "all" {
		return ports
	}

	var filtered []string
	for _, port := range ports {
		info, err := serial.GetPortInfo(port)
		if err != nil {
			continue
		}

		name := strings.ToLower(info.Name)
		switch strings.ToLower(filterType) {
		case "usb":
			if strings.HasPrefix(name, "ttyusb") || strings.HasPrefix(name, "ttyacm") {
				filtered = append(filtered, port)
			}
		case "standard":
			if strings.HasPrefix(name, "ttys") {
				filtered = append(filtered, port)
			}
		case "arm":
			if strings.HasPrefix(name, "ttyama") {
				filtered = append(filtered, port)
			}
		}
	}
	return filtered
}

// renderTable renders the port list in a styled static table format
func renderTable(ports []string) {
	fmt.Printf("Found %d serial port(s):\n\n", len(ports))

	// Define column widths
	portWidth := 15
	typeWidth := 20
	descWidth := 30

	// Create styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("240")).
		PaddingBottom(1)

	cellStyle := lipgloss.NewStyle().
		PaddingRight(2)

	// Print header
	header := fmt.Sprintf("%-*s %-*s %-*s",
		portWidth, "Port",
		typeWidth, "Type",
		descWidth, "Description")
	fmt.Println(headerStyle.Render(header))

	// Print rows
	for _, port := range ports {
		info, err := serial.GetPortInfo(port)
		if err != nil {
			row := fmt.Sprintf("%-*s %-*s %-*s",
				portWidth, port,
				typeWidth, "Unknown",
				descWidth, fmt.Sprintf("Error: %v", err))
			fmt.Println(cellStyle.Render(row))
			continue
		}

		portType := getPortType(info.Name)
		row := fmt.Sprintf("%-*s %-*s %-*s",
			portWidth, info.Name,
			typeWidth, portType,
			descWidth, info.Description)
		fmt.Println(cellStyle.Render(row))
	}
}

// renderSimple renders the port list in simple text format
func renderSimple(ports []string) {
	for _, port := range ports {
		fmt.Println(port)
	}
}

// getPortType returns a more specific type classification for the port
func getPortType(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.HasPrefix(name, "ttyusb"):
		return "USB Serial"
	case strings.HasPrefix(name, "ttyacm"):
		return "USB CDC/ACM"
	case strings.HasPrefix(name, "ttyama"):
		return "ARM Serial"
	case strings.HasPrefix(name, "ttymxc"):
		return "i.MX Serial"
	case strings.HasPrefix(name, "ttysac"):
		return "Samsung Serial"
	case strings.HasPrefix(name, "ttyths"):
		return "Tegra Serial"
	case strings.HasPrefix(name, "ttyo"):
		return "OMAP Serial"
	case strings.HasPrefix(name, "ttys"):
		return "Standard Serial"
	default:
		return "Serial Port"
	}
}
