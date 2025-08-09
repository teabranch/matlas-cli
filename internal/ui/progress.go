package ui

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ProgressIndicator provides visual feedback for long-running operations
type ProgressIndicator struct {
	output  io.Writer
	verbose bool
	quiet   bool
	spinner *Spinner
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(verbose, quiet bool) *ProgressIndicator {
	return &ProgressIndicator{
		output:  os.Stderr,
		verbose: verbose,
		quiet:   quiet,
	}
}

// NewProgressIndicatorWithWriter creates a progress indicator with custom writer
func NewProgressIndicatorWithWriter(output io.Writer, verbose, quiet bool) *ProgressIndicator {
	return &ProgressIndicator{
		output:  output,
		verbose: verbose,
		quiet:   quiet,
	}
}

// StartSpinner starts a spinner with the given message
func (p *ProgressIndicator) StartSpinner(message string) {
	if p.quiet {
		return
	}

	p.spinner = NewSpinner(p.output, message)
	p.spinner.Start()
}

// StopSpinner stops the current spinner
func (p *ProgressIndicator) StopSpinner(successMessage string) {
	if p.spinner != nil {
		p.spinner.Stop(successMessage)
		p.spinner = nil
	}
}

// StopSpinnerWithError stops the spinner with an error message
func (p *ProgressIndicator) StopSpinnerWithError(errorMessage string) {
	if p.spinner != nil {
		p.spinner.StopWithError(errorMessage)
		p.spinner = nil
	}
}

// Print prints a message respecting quiet/verbose settings
func (p *ProgressIndicator) Print(message string) {
	if !p.quiet {
		fmt.Fprintln(p.output, message)
	}
}

// PrintVerbose prints a message only in verbose mode
func (p *ProgressIndicator) PrintVerbose(message string) {
	if p.verbose && !p.quiet {
		fmt.Fprintln(p.output, message)
	}
}

// Spinner provides a spinning animation for operations
type Spinner struct {
	output   io.Writer
	message  string
	frames   []string
	active   bool
	stopChan chan bool
}

// NewSpinner creates a new spinner
func NewSpinner(output io.Writer, message string) *Spinner {
	return &Spinner{
		output:   output,
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stopChan: make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	if s.active {
		return
	}

	s.active = true
	go s.animate()
}

// Stop stops the spinner with a success message
func (s *Spinner) Stop(successMessage string) {
	if !s.active {
		return
	}

	s.active = false
	s.stopChan <- true

	// Clear the spinner line and print success message
	fmt.Fprintf(s.output, "\r\033[K✓ %s\n", successMessage)
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(errorMessage string) {
	if !s.active {
		return
	}

	s.active = false
	s.stopChan <- true

	// Clear the spinner line and print error message
	fmt.Fprintf(s.output, "\r\033[K✗ %s\n", errorMessage)
}

// animate runs the spinner animation loop
func (s *Spinner) animate() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	frameIndex := 0

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if s.active {
				frame := s.frames[frameIndex%len(s.frames)]
				fmt.Fprintf(s.output, "\r%s %s", frame, s.message)
				frameIndex++
			}
		}
	}
}

// ProgressBar represents a progress bar for multi-step operations
type ProgressBar struct {
	output  io.Writer
	total   int
	current int
	width   int
	prefix  string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(output io.Writer, total int, prefix string) *ProgressBar {
	return &ProgressBar{
		output: output,
		total:  total,
		width:  40,
		prefix: prefix,
	}
}

// Update updates the progress bar
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pb.render()
}

// Increment increments the progress by 1
func (pb *ProgressBar) Increment() {
	pb.current++
	pb.render()
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish(message string) {
	pb.current = pb.total
	pb.render()
	fmt.Fprintf(pb.output, " %s\n", message)
}

// render draws the progress bar
func (pb *ProgressBar) render() {
	if pb.total <= 0 {
		return
	}

	percent := float64(pb.current) / float64(pb.total)
	if percent > 1 {
		percent = 1
	}

	filled := int(percent * float64(pb.width))
	bar := ""

	for i := 0; i < pb.width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	fmt.Fprintf(pb.output, "\r%s [%s] %d/%d (%.1f%%)",
		pb.prefix, bar, pb.current, pb.total, percent*100)
}
