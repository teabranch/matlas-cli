package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// newHelpCmd returns a help command that can also emit Markdown for CI doc generation
func newHelpCmd(root *cobra.Command) *cobra.Command {
	var format string

	helpCmd := &cobra.Command{
		Use:                   "help [command]",
		Short:                 "Show help for any command",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "markdown", "md":
				// If a specific command path is provided, render that command only
				if len(args) > 0 {
					target, _, err := root.Find(args)
					if err != nil || target == nil {
						return fmt.Errorf("unknown command: %v", args)
					}
					return renderMarkdownTree(target, os.Stdout)
				}
				// Otherwise, render the entire tree into a single markdown stream
				return renderMarkdownTree(root, os.Stdout)
			default:
				// Standard text help
				if len(args) > 0 {
					target, _, err := root.Find(args)
					if err == nil && target != nil {
						return target.Help()
					}
					return fmt.Errorf("unknown command: %v", args)
				}
				return root.Help()
			}
		},
	}

	helpCmd.Flags().StringVar(&format, "format", "text", "Output format: text, markdown")
	return helpCmd
}

func renderMarkdownTree(cmd *cobra.Command, w io.Writer) error {
	if err := renderMarkdownForCommand(cmd, w); err != nil {
		return err
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if _, err := fmt.Fprint(w, "\n---\n\n"); err != nil {
			return err
		}
		if err := renderMarkdownTree(c, w); err != nil {
			return err
		}
	}
	return nil
}

func renderMarkdownForCommand(cmd *cobra.Command, w io.Writer) error {
	var b bytes.Buffer

	// Title as command path
	path := commandPath(cmd)
	b.WriteString("## ")
	b.WriteString(path)
	b.WriteString("\n\n")

	// Short and Long
	if s := strings.TrimSpace(cmd.Short); s != "" {
		b.WriteString(s)
		b.WriteString("\n\n")
	}
	if l := strings.TrimSpace(cmd.Long); l != "" && l != cmd.Short {
		b.WriteString(l)
		b.WriteString("\n\n")
	}

	// Usage
	b.WriteString("**Usage**\n\n")
	b.WriteString("```\n")
	b.WriteString(cmd.UseLine())
	b.WriteString("\n```\n\n")

	// Aliases
	if len(cmd.Aliases) > 0 {
		b.WriteString("**Aliases**: ")
		b.WriteString(strings.Join(cmd.Aliases, ", "))
		b.WriteString("\n\n")
	}

	// Examples
	if ex := strings.TrimSpace(cmd.Example); ex != "" {
		b.WriteString("**Examples**\n\n")
		b.WriteString("```\n")
		b.WriteString(ex)
		b.WriteString("\n```\n\n")
	}

	// Flags
	localFlags := cmd.NonInheritedFlags()
	inheritedFlags := cmd.InheritedFlags()
	if localFlags.HasFlags() {
		b.WriteString("**Flags**\n\n")
		b.WriteString("```\n")
		b.WriteString(strings.TrimSpace(localFlags.FlagUsages()))
		b.WriteString("\n```\n\n")
	}
	if inheritedFlags.HasFlags() {
		b.WriteString("**Inherited Flags**\n\n")
		b.WriteString("```\n")
		b.WriteString(strings.TrimSpace(inheritedFlags.FlagUsages()))
		b.WriteString("\n```\n\n")
	}

	// Subcommands
	subs := cmd.Commands()
	var visible []*cobra.Command
	for _, c := range subs {
		if c.IsAvailableCommand() && !c.IsAdditionalHelpTopicCommand() {
			visible = append(visible, c)
		}
	}
	if len(visible) > 0 {
		b.WriteString("**Subcommands**\n\n")
		for _, c := range visible {
			b.WriteString("- ")
			b.WriteString(c.Name())
			if s := strings.TrimSpace(c.Short); s != "" {
				b.WriteString(": ")
				b.WriteString(s)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	_, err := w.Write(b.Bytes())
	return err
}

func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	names := []string{}
	current := cmd
	for current != nil {
		// stop before the hidden auto help if any
		names = append([]string{current.Name()}, names...)
		current = current.Parent()
	}
	return strings.Join(names, " ")
}
