// Package main is the entry point for matlas-cli.
package main

import "github.com/teabranch/matlas-cli/cmd"

// Build-time variables (set via -ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	builtBy   = "manual"
)

func main() {
	cmd.Execute(version, commit, buildTime, builtBy)
}
