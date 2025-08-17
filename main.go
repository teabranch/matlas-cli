// Package main is the entry point for matlas-cli.
package main

import "github.com/teabranch/matlas-cli/cmd"

// Build-time variables (set via -ldflags)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
	builtBy = "manual"
)

func main() {
	cmd.Execute(version, commit, date, builtBy)
}
