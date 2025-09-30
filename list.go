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
	Name        string // Device name (e.g., "ttyACM0")
	Path        string // Full device path (e.g., "/dev/ttyACM0")
	Description string // Human-readable description

	// USB Device Information (Linux-specific, empty on other platforms)
	VendorID        string // USB Vendor ID (hex, e.g., "1a86")
	ProductID       string // USB Product ID (hex, e.g., "55d2")
	SerialNumber    string // USB Serial Number (e.g., "5481031032")
	InterfaceNumber string // USB Interface Number (hex, e.g., "02")
	BusNumber       string // USB Bus Number (decimal, e.g., "001")
	DeviceNumber    string // USB Device Number (decimal, e.g., "003")

	// Additional metadata
	Manufacturer string // USB Manufacturer string (if available)
	Product      string // USB Product string (if available)
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
// This function is Linux-specific and gracefully handles missing/inaccessible files
func enrichUSBInfo(info *PortInfo) {
	// Construct sysfs path for this device
	// /sys/class/tty/{device}/device/ is a symlink that resolves to the tty subdirectory
	devicePath := filepath.Join("/sys/class/tty", info.Name, "device")

	// Resolve the symlink to get the actual device path
	// This typically resolves to: .../5-2.3.1:1.0/ttyUSB0
	resolvedPath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return // Can't resolve symlink, gracefully fail
	}

	// Go up one level to get to the interface directory (.../5-2.3.1:1.0)
	interfacePath := filepath.Dir(resolvedPath)

	// Read interface-level properties
	info.InterfaceNumber = readSysfsFile(filepath.Join(interfacePath, "bInterfaceNumber"))

	// Go up one more level to get to the USB device directory (.../5-2.3.1)
	usbDevicePath := filepath.Dir(interfacePath)

	info.VendorID = readSysfsFile(filepath.Join(usbDevicePath, "idVendor"))
	info.ProductID = readSysfsFile(filepath.Join(usbDevicePath, "idProduct"))
	info.SerialNumber = readSysfsFile(filepath.Join(usbDevicePath, "serial"))
	info.Manufacturer = readSysfsFile(filepath.Join(usbDevicePath, "manufacturer"))
	info.Product = readSysfsFile(filepath.Join(usbDevicePath, "product"))

	// Read bus and device numbers for USB reset
	info.BusNumber = readSysfsFile(filepath.Join(usbDevicePath, "busnum"))
	info.DeviceNumber = readSysfsFile(filepath.Join(usbDevicePath, "devnum"))
}

// readSysfsFile reads a single-line sysfs file and returns trimmed content
// Returns empty string on any error (graceful degradation)
func readSysfsFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "" // File doesn't exist or no permission
	}
	return strings.TrimSpace(string(data))
}
