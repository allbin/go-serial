package serial

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ListPorts returns a list of available serial ports on the system
// Filters for communication-capable devices and excludes virtual terminals
func ListPorts() ([]string, error) {
	var ports []string

	// Check /dev directory for serial devices
	devDir := "/dev"
	entries, err := os.ReadDir(devDir)
	if err != nil {
		return nil, err
	}

	// Regular expressions for different types of serial devices
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^ttyUSB\d+$`), // USB serial adapters
		regexp.MustCompile(`^ttyACM\d+$`), // USB CDC/ACM devices
		regexp.MustCompile(`^ttyS\d+$`),   // Standard serial ports
		regexp.MustCompile(`^ttyAMA\d+$`), // ARM/Raspberry Pi serial
		regexp.MustCompile(`^ttymxc\d+$`), // i.MX serial ports
		regexp.MustCompile(`^ttyO\d+$`),   // OMAP serial ports
		regexp.MustCompile(`^ttySAC\d+$`), // Samsung serial ports
		regexp.MustCompile(`^ttyTHS\d+$`), // Tegra serial ports
	}

	// Exclude patterns for virtual terminals and other non-serial devices
	excludePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^tty\d+$`),  // Virtual terminals (tty1, tty2, etc.)
		regexp.MustCompile(`^console$`), // Console
		regexp.MustCompile(`^ptmx$`),    // Pseudo-terminal multiplexer
		regexp.MustCompile(`^pty.*$`),   // Pseudo-terminals
		regexp.MustCompile(`^pts/.*$`),  // Pseudo-terminal slaves
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip if it matches exclude patterns
		excluded := false
		for _, excludePattern := range excludePatterns {
			if excludePattern.MatchString(name) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Check if it matches any of our serial device patterns
		matched := false
		for _, pattern := range patterns {
			if pattern.MatchString(name) {
				matched = true
				break
			}
		}

		if matched {
			fullPath := filepath.Join(devDir, name)

			// Verify it's a character device (not a directory or regular file)
			if isCharacterDevice(fullPath) {
				ports = append(ports, fullPath)
			}
		}
	}

	// Sort the ports for consistent ordering
	sort.Strings(ports)

	return ports, nil
}

// isCharacterDevice checks if the given path is a character device
func isCharacterDevice(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a character device
	mode := info.Mode()
	return mode&os.ModeCharDevice != 0
}

// GetPortInfo returns detailed information about a serial port
type PortInfo struct {
	Name         string
	Path         string
	Description  string
	VendorID     string
	ProductID    string
	SerialNumber string
}

// GetPortInfo returns detailed information about a specific port
func GetPortInfo(portPath string) (*PortInfo, error) {
	// Basic validation
	if !isCharacterDevice(portPath) {
		return nil, ErrDeviceNotFound
	}

	// Extract the device name from the path
	name := filepath.Base(portPath)

	info := &PortInfo{
		Name:        name,
		Path:        portPath,
		Description: getPortDescription(name),
	}

	// Try to get USB device information if it's a USB device
	if strings.HasPrefix(name, "ttyUSB") || strings.HasPrefix(name, "ttyACM") {
		enrichUSBInfo(info)
	}

	return info, nil
}

// getPortDescription provides human-readable descriptions for different port types
func getPortDescription(name string) string {
	switch {
	case strings.HasPrefix(name, "ttyUSB"):
		return "USB Serial Port"
	case strings.HasPrefix(name, "ttyACM"):
		return "USB CDC/ACM Device"
	case strings.HasPrefix(name, "ttyAMA"):
		return "ARM Serial Port"
	case strings.HasPrefix(name, "ttymxc"):
		return "i.MX Serial Port"
	case strings.HasPrefix(name, "ttySAC"):
		return "Samsung Serial Port"
	case strings.HasPrefix(name, "ttyTHS"):
		return "Tegra Serial Port"
	case strings.HasPrefix(name, "ttyO"):
		return "OMAP Serial Port"
	case strings.HasPrefix(name, "ttyS"):
		return "Standard Serial Port"
	default:
		return "Serial Port"
	}
}

// enrichUSBInfo attempts to get USB device information from sysfs
func enrichUSBInfo(info *PortInfo) {
	// This is a simplified version - full implementation would read from
	// /sys/class/tty/{device}/device/... to get USB vendor/product info
	// For now, we'll leave these fields empty as they require more complex
	// sysfs parsing which can vary between systems

	// TODO: Implement full USB device info parsing from sysfs
	// This would involve:
	// 1. Following symlinks in /sys/class/tty/{device}/
	// 2. Reading idVendor, idProduct, serial files
	// 3. Looking up vendor/product names from USB ID database
}
