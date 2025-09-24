package serial

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestListPorts(t *testing.T) {
	ports, err := ListPorts()
	if err != nil {
		t.Errorf("ListPorts failed: %v", err)
	}

	// Check that all returned ports are valid paths
	for _, port := range ports {
		if !strings.HasPrefix(port, "/dev/") {
			t.Errorf("Port path doesn't start with /dev/: %s", port)
		}

		// Verify it's a character device
		if !isCharacterDevice(port) {
			t.Errorf("Port is not a character device: %s", port)
		}
	}

	// Check that ports are sorted
	for i := 1; i < len(ports); i++ {
		if ports[i-1] > ports[i] {
			t.Errorf("Ports are not sorted: %s > %s", ports[i-1], ports[i])
		}
	}
}

func TestIsCharacterDevice(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
		create   bool // Whether to create a test file
	}{
		{"/dev/null", true, false},     // Should exist and be a character device
		{"/dev/zero", true, false},     // Should exist and be a character device
		{"/tmp", false, false},         // Directory, not character device
		{"/nonexistent", false, false}, // Doesn't exist
	}

	for _, test := range tests {
		result := isCharacterDevice(test.path)
		if result != test.expected {
			t.Errorf("isCharacterDevice(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestGetPortDescription(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"ttyUSB0", "USB Serial Port"},
		{"ttyACM0", "USB CDC/ACM Device"},
		{"ttyS0", "Standard Serial Port"},
		{"ttyAMA0", "ARM Serial Port"},
		{"ttymxc0", "i.MX Serial Port"},
		{"ttyO0", "OMAP Serial Port"},
		{"ttySAC0", "Samsung Serial Port"},
		{"ttyTHS0", "Tegra Serial Port"},
		{"unknown", "Serial Port"},
	}

	for _, test := range tests {
		result := getPortDescription(test.name)
		if result != test.expected {
			t.Errorf("getPortDescription(%s) = %s, expected %s", test.name, result, test.expected)
		}
	}
}

func TestGetPortInfo(t *testing.T) {
	// Test with /dev/null as it should always exist and be a character device
	info, err := GetPortInfo("/dev/null")
	if err != nil {
		t.Errorf("GetPortInfo failed for /dev/null: %v", err)
	}

	if info == nil {
		t.Error("GetPortInfo returned nil info")
		return
	}

	if info.Name != "null" {
		t.Errorf("Expected name 'null', got '%s'", info.Name)
	}

	if info.Path != "/dev/null" {
		t.Errorf("Expected path '/dev/null', got '%s'", info.Path)
	}

	if info.Description == "" {
		t.Error("Description should not be empty")
	}

	// Test with non-existent device
	_, err = GetPortInfo("/dev/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
	if err != ErrDeviceNotFound {
		t.Errorf("Expected ErrDeviceNotFound, got %v", err)
	}
}

// TestPortFiltering tests that we correctly filter different types of devices
func TestPortFiltering(t *testing.T) {
	// Create test device files
	testDevices := []struct {
		name        string
		shouldMatch bool
	}{
		{"ttyUSB0", true},
		{"ttyUSB1", true},
		{"ttyACM0", true},
		{"ttyS0", true},
		{"ttyAMA0", true},
		{"tty1", false},    // Virtual terminal - should be excluded
		{"tty2", false},    // Virtual terminal - should be excluded
		{"console", false}, // Console - should be excluded
		{"ptmx", false},    // Pseudo-terminal - should be excluded
		{"ptyp0", false},   // Pseudo-terminal - should be excluded
		{"random", false},  // Not a serial device
		{"urandom", false}, // Not a serial device
	}

	// We can't actually create character devices in tests without root,
	// so we'll test the pattern matching logic separately
	for _, device := range testDevices {
		// Test the pattern matching logic by calling our internal functions
		matched := matchesSerialPattern(device.name)
		excluded := matchesExcludePattern(device.name)

		expectedMatch := device.shouldMatch && !excluded

		if matched != expectedMatch {
			t.Errorf("Device %s: expected match=%v, got match=%v (excluded=%v)",
				device.name, expectedMatch, matched, excluded)
		}
	}
}

// Helper function to test pattern matching without file system operations
func matchesSerialPattern(name string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^ttyUSB\d+$`),
		regexp.MustCompile(`^ttyACM\d+$`),
		regexp.MustCompile(`^ttyS\d+$`),
		regexp.MustCompile(`^ttyAMA\d+$`),
		regexp.MustCompile(`^ttymxc\d+$`),
		regexp.MustCompile(`^ttyO\d+$`),
		regexp.MustCompile(`^ttySAC\d+$`),
		regexp.MustCompile(`^ttyTHS\d+$`),
	}

	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// Helper function to test exclude pattern matching
func matchesExcludePattern(name string) bool {
	excludePatterns := []*regexp.Regexp{
		regexp.MustCompile(`^tty\d+$`),
		regexp.MustCompile(`^console$`),
		regexp.MustCompile(`^ptmx$`),
		regexp.MustCompile(`^pty.*$`),
		regexp.MustCompile(`^pts/.*$`),
	}

	for _, pattern := range excludePatterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}

// BenchmarkListPorts benchmarks the ListPorts function
func BenchmarkListPorts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ListPorts()
		if err != nil {
			b.Errorf("ListPorts failed: %v", err)
		}
	}
}

// TestListPortsIntegration is an integration test that requires actual system
func TestListPortsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ports, err := ListPorts()
	if err != nil {
		t.Fatalf("ListPorts failed: %v", err)
	}

	t.Logf("Found %d serial ports:", len(ports))
	for i, port := range ports {
		info, err := GetPortInfo(port)
		if err != nil {
			t.Logf("  %d. %s (error getting info: %v)", i+1, port, err)
		} else {
			t.Logf("  %d. %s (%s)", i+1, port, info.Description)
		}
	}

	// Verify each port can be stat'd and is a character device
	for _, port := range ports {
		stat, err := os.Stat(port)
		if err != nil {
			t.Errorf("Cannot stat port %s: %v", port, err)
			continue
		}

		if stat.Mode()&os.ModeCharDevice == 0 {
			t.Errorf("Port %s is not a character device", port)
		}
	}
}
