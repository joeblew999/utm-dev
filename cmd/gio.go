package cmd

import "github.com/spf13/cobra"

var gioCmd = &cobra.Command{
	Use:   "gio",
	Short: "Build and run Gio applications",
	Long: `Build and run Gio applications (desktop, mobile, and web).

All dependencies are installed automatically if missing.

Build:
  utm-dev gio build android examples/hybrid-dashboard
  utm-dev gio build ios examples/hybrid-dashboard
  utm-dev gio build macos examples/hybrid-dashboard

Run:
  utm-dev gio run android examples/hybrid-dashboard
  utm-dev gio run ios-simulator examples/hybrid-dashboard

Package & Sign:
  utm-dev gio bundle macos examples/hybrid-dashboard
  utm-dev gio package android examples/hybrid-dashboard`,
}

func init() {
	gioCmd.GroupID = "build"
	rootCmd.AddCommand(gioCmd)
}
