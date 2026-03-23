// Package cli provides human-readable CLI output with verbosity control.
//
// All output goes to stderr so stdout stays clean for machine-parseable JSON
// (used by pkg/self/output). Use SetVerbose/SetQuiet to control what prints.
package cli

import (
	"fmt"
	"os"
	"sync"
)

var (
	mu      sync.Mutex
	verbose bool
	quiet   bool
)

// SetVerbose enables Debug output.
func SetVerbose(v bool) {
	mu.Lock()
	verbose = v
	mu.Unlock()
}

// SetQuiet suppresses Info and Success output. Warn and Error always print.
func SetQuiet(q bool) {
	mu.Lock()
	quiet = q
	mu.Unlock()
}

// IsVerbose returns whether verbose mode is enabled.
func IsVerbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

// IsQuiet returns whether quiet mode is enabled.
func IsQuiet() bool {
	mu.Lock()
	defer mu.Unlock()
	return quiet
}

// Info prints a message. Suppressed by quiet mode.
func Info(format string, args ...any) {
	if IsQuiet() {
		return
	}
	mu.Lock()
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	mu.Unlock()
}

// Success prints a message with a checkmark prefix. Suppressed by quiet mode.
func Success(format string, args ...any) {
	if IsQuiet() {
		return
	}
	mu.Lock()
	fmt.Fprintf(os.Stderr, "✓ "+format+"\n", args...)
	mu.Unlock()
}

// Warn prints a warning. Always shown.
func Warn(format string, args ...any) {
	mu.Lock()
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
	mu.Unlock()
}

// Error prints an error. Always shown.
func Error(format string, args ...any) {
	mu.Lock()
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	mu.Unlock()
}

// Debug prints a message only when verbose mode is enabled.
func Debug(format string, args ...any) {
	if !IsVerbose() {
		return
	}
	mu.Lock()
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	mu.Unlock()
}
