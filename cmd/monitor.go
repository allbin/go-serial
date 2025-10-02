/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/allbin/go-serial"
	"github.com/spf13/cobra"
)

var (
	monitorSignals []string
	monitorTimeout time.Duration
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor <port>",
	Short: "Monitor modem signal changes",
	Long: `Monitor modem control signal changes in real-time.

Watches specified signals and reports when they change state. Press Ctrl+C to stop.

Examples:
  serial monitor /dev/ttyUSB0
  serial monitor /dev/ttyUSB0 --signals cts,dsr
  serial monitor /dev/ttyUSB0 --signals dcd --timeout 30s

Available signals: cts, dsr, ri, dcd`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		port, err := serial.Open(portPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening port: %v\n", err)
			os.Exit(1)
		}
		defer port.Close()

		// Parse signal mask from flags
		mask, err := parseSignalMask(monitorSignals)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing signals: %v\n", err)
			os.Exit(1)
		}

		// Setup signal handler for Ctrl+C
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\nStopping monitor...")
			cancel()
		}()

		fmt.Printf("Monitoring signals on %s (signals: %s)\n", portPath, strings.Join(monitorSignals, ", "))
		fmt.Println("Press Ctrl+C to stop")

		// Show initial state
		initialSignals, err := port.GetModemSignals()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading initial signals: %v\n", err)
			os.Exit(1)
		}
		printSignalState("Initial", initialSignals, mask)

		// Monitor loop
		for {
			var signals serial.ModemSignals
			var changed serial.SignalMask

			if monitorTimeout > 0 {
				timeoutCtx, timeoutCancel := context.WithTimeout(ctx, monitorTimeout)
				signals, changed, err = port.WaitForSignalChangeContext(timeoutCtx, mask)
				timeoutCancel()
			} else {
				signals, changed, err = port.WaitForSignalChangeContext(ctx, mask)
			}

			if err != nil {
				if err == context.Canceled {
					return
				}
				if err == serial.ErrSignalTimeout || err == context.DeadlineExceeded {
					fmt.Printf("[%s] Timeout - no signal changes\n", time.Now().Format("15:04:05"))
					continue
				}
				fmt.Fprintf(os.Stderr, "Error waiting for signal change: %v\n", err)
				os.Exit(1)
			}

			printSignalChange(signals, changed)
		}
	},
}

func parseSignalMask(signalNames []string) (serial.SignalMask, error) {
	if len(signalNames) == 0 {
		return serial.SignalCTS | serial.SignalDSR | serial.SignalRI | serial.SignalDCD, nil
	}

	var mask serial.SignalMask
	for _, name := range signalNames {
		switch strings.ToLower(name) {
		case "cts":
			mask |= serial.SignalCTS
		case "dsr":
			mask |= serial.SignalDSR
		case "ri":
			mask |= serial.SignalRI
		case "dcd":
			mask |= serial.SignalDCD
		default:
			return 0, fmt.Errorf("unknown signal: %s (valid: cts, dsr, ri, dcd)", name)
		}
	}
	return mask, nil
}

func printSignalState(prefix string, signals serial.ModemSignals, mask serial.SignalMask) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s state:\n", timestamp, prefix)
	if mask&serial.SignalCTS != 0 {
		fmt.Printf("  CTS: %s\n", formatSignalState(signals.CTS))
	}
	if mask&serial.SignalDSR != 0 {
		fmt.Printf("  DSR: %s\n", formatSignalState(signals.DSR))
	}
	if mask&serial.SignalRI != 0 {
		fmt.Printf("  RI:  %s\n", formatSignalState(signals.RI))
	}
	if mask&serial.SignalDCD != 0 {
		fmt.Printf("  DCD: %s\n", formatSignalState(signals.DCD))
	}
	fmt.Println()
}

func printSignalChange(signals serial.ModemSignals, changed serial.SignalMask) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] Signal change detected:\n", timestamp)
	if changed&serial.SignalCTS != 0 {
		fmt.Printf("  CTS: %s\n", formatSignalState(signals.CTS))
	}
	if changed&serial.SignalDSR != 0 {
		fmt.Printf("  DSR: %s\n", formatSignalState(signals.DSR))
	}
	if changed&serial.SignalRI != 0 {
		fmt.Printf("  RI:  %s\n", formatSignalState(signals.RI))
	}
	if changed&serial.SignalDCD != 0 {
		fmt.Printf("  DCD: %s\n", formatSignalState(signals.DCD))
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	monitorCmd.Flags().StringSliceVarP(&monitorSignals, "signals", "s", []string{"cts", "dsr", "ri", "dcd"},
		"Signals to monitor (comma-separated: cts,dsr,ri,dcd)")
	monitorCmd.Flags().DurationVarP(&monitorTimeout, "timeout", "t", 0,
		"Timeout for each wait operation (0 = no timeout)")
}
