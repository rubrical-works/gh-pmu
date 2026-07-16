// Package ui provides styled terminal output helpers for gh-pmu.
package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// Box style characters
const (
	BoxTopLeft        = "╭"
	BoxTopRight       = "╮"
	BoxBottomLeft     = "╰"
	BoxBottomRight    = "╯"
	BoxHorizontal     = "─"
	BoxVertical       = "│"
	BoxTopLeftAlt     = "┌"
	BoxTopRightAlt    = "┐"
	BoxBottomLeftAlt  = "└"
	BoxBottomRightAlt = "┘"
)

// Symbols for status indicators
const (
	SymbolCheck   = "✓"
	SymbolCross   = "✗"
	SymbolWarning = "⚠"
	SymbolInfo    = "ℹ"
	SymbolArrow   = "→"
)

// Spinner frames for loading animation
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// UI provides styled terminal output
type UI struct {
	out     io.Writer
	noColor bool
}

// New creates a new UI instance
func New(out io.Writer) *UI {
	return &UI{
		out:     out,
		noColor: false,
	}
}

// NewWithOptions creates a new UI with options
func NewWithOptions(out io.Writer, noColor bool) *UI {
	return &UI{
		out:     out,
		noColor: noColor,
	}
}

// color wraps text in ANSI color codes if color is enabled
func (u *UI) color(c, text string) string {
	if u.noColor {
		return text
	}
	return c + text + Reset
}

// Success prints a green checkmark with message
func (u *UI) Success(msg string) {
	fmt.Fprintf(u.out, "%s %s\n", u.color(Green, SymbolCheck), msg)
}

// Error prints a red cross with message
func (u *UI) Error(msg string) {
	fmt.Fprintf(u.out, "%s %s\n", u.color(Red, SymbolCross), msg)
}

// Warning prints a yellow warning with message
func (u *UI) Warning(msg string) {
	fmt.Fprintf(u.out, "%s %s\n", u.color(Yellow, SymbolWarning), msg)
}

// Info prints an info symbol with message
func (u *UI) Info(msg string) {
	fmt.Fprintf(u.out, "%s %s\n", u.color(Cyan, SymbolInfo), msg)
}

// Step prints a step indicator (e.g., "Step 1 of 3: Title")
func (u *UI) Step(current, total int, title string) {
	fmt.Fprintf(u.out, "\n%s\n", u.color(Bold+Cyan, fmt.Sprintf("Step %d of %d: %s", current, total, title)))
}

// Header prints a styled header box
func (u *UI) Header(title, subtitle string) {
	width := max(visibleWidth(title), visibleWidth(subtitle)) + 4
	if width < 40 {
		width = 40
	}

	// Top border
	fmt.Fprintf(u.out, "%s%s%s\n",
		u.color(Cyan, BoxTopLeft),
		u.color(Cyan, strings.Repeat(BoxHorizontal, width)),
		u.color(Cyan, BoxTopRight))

	// Title line
	titlePadding := width - visibleWidth(title) - 2
	fmt.Fprintf(u.out, "%s  %s%s%s\n",
		u.color(Cyan, BoxVertical),
		u.color(Bold+White, title),
		strings.Repeat(" ", titlePadding),
		u.color(Cyan, BoxVertical))

	// Subtitle line
	if subtitle != "" {
		subtitlePadding := width - visibleWidth(subtitle) - 2
		fmt.Fprintf(u.out, "%s  %s%s%s\n",
			u.color(Cyan, BoxVertical),
			u.color(Dim+White, subtitle),
			strings.Repeat(" ", subtitlePadding),
			u.color(Cyan, BoxVertical))
	}

	// Bottom border
	fmt.Fprintf(u.out, "%s%s%s\n",
		u.color(Cyan, BoxBottomLeft),
		u.color(Cyan, strings.Repeat(BoxHorizontal, width)),
		u.color(Cyan, BoxBottomRight))
}

// Box prints content in a box
func (u *UI) Box(lines []string) {
	if len(lines) == 0 {
		return
	}

	// Find max visible width (rune count without ANSI codes)
	maxWidth := 0
	for _, line := range lines {
		w := visibleWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	width := maxWidth + 4
	if width < 40 {
		width = 40
	}

	// Top border
	fmt.Fprintf(u.out, "%s%s%s\n",
		BoxTopLeftAlt,
		strings.Repeat(BoxHorizontal, width),
		BoxTopRightAlt)

	// Content lines
	for _, line := range lines {
		w := visibleWidth(line)
		padding := width - w - 2
		if padding < 0 {
			padding = 0
		}
		fmt.Fprintf(u.out, "%s  %s%s%s\n",
			BoxVertical,
			line,
			strings.Repeat(" ", padding),
			BoxVertical)
	}

	// Bottom border
	fmt.Fprintf(u.out, "%s%s%s\n",
		BoxBottomLeftAlt,
		strings.Repeat(BoxHorizontal, width),
		BoxBottomRightAlt)
}

// SummaryBox prints a styled summary box with key-value pairs
func (u *UI) SummaryBox(title string, items map[string]string, order []string) {
	// Calculate widths
	maxKeyLen := 0
	maxValLen := 0
	for _, key := range order {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
		if val, ok := items[key]; ok && len(val) > maxValLen {
			maxValLen = len(val)
		}
	}

	// Build lines
	var lines []string
	lines = append(lines, u.color(Green, SymbolCheck)+" "+u.color(Bold, title))
	lines = append(lines, "")

	for _, key := range order {
		if val, ok := items[key]; ok {
			keyPadded := key + ":" + strings.Repeat(" ", maxKeyLen-len(key)+1)
			lines = append(lines, u.color(Dim, keyPadded)+val)
		}
	}

	// Find max visible width (rune count without ANSI codes)
	maxWidth := 0
	for _, line := range lines {
		w := visibleWidth(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	width := maxWidth + 4
	if width < 40 {
		width = 40
	}

	// Top border
	fmt.Fprintf(u.out, "\n%s%s%s\n",
		u.color(Green, BoxTopLeft),
		u.color(Green, strings.Repeat(BoxHorizontal, width)),
		u.color(Green, BoxTopRight))

	// Content lines
	for _, line := range lines {
		w := visibleWidth(line)
		padding := width - w - 2
		if padding < 0 {
			padding = 0
		}
		fmt.Fprintf(u.out, "%s  %s%s%s\n",
			u.color(Green, BoxVertical),
			line,
			strings.Repeat(" ", padding),
			u.color(Green, BoxVertical))
	}

	// Bottom border
	fmt.Fprintf(u.out, "%s%s%s\n",
		u.color(Green, BoxBottomLeft),
		u.color(Green, strings.Repeat(BoxHorizontal, width)),
		u.color(Green, BoxBottomRight))
}

// Menu prints a selection menu and returns formatted lines
func (u *UI) Menu(options []string, includeManualOption bool) []string {
	var lines []string
	for i, opt := range options {
		lines = append(lines, fmt.Sprintf("  %s %s", u.color(Cyan, fmt.Sprintf("%d.", i+1)), opt))
	}
	if includeManualOption {
		lines = append(lines, fmt.Sprintf("  %s %s", u.color(Dim, "0."), u.color(Dim, "Enter project number manually")))
	}
	return lines
}

// PrintMenu prints a menu with a box
func (u *UI) PrintMenu(options []string, includeManualOption bool) {
	menuLines := u.Menu(options, includeManualOption)
	u.Box(menuLines)
}

// Prompt prints a prompt and returns the formatted string
func (u *UI) Prompt(label string, defaultVal string) string {
	if defaultVal != "" {
		return fmt.Sprintf("%s [%s]: ", u.color(Cyan, label), u.color(Dim, defaultVal))
	}
	return fmt.Sprintf("%s: ", u.color(Cyan, label))
}

// Spinner provides an animated loading indicator
type Spinner struct {
	mu       sync.Mutex
	out      io.Writer
	message  string
	active   bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	frameIdx int
}

// NewSpinner creates a new spinner
func NewSpinner(out io.Writer, message string) *Spinner {
	return &Spinner{
		out:     out,
		message: message,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start begins the spinner animation. It is safe to call after Stop: fresh
// channels are created for each run so the spinner is restartable.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	// Recreate the channels for this run. Stop closes stopCh and waits on doneCh;
	// reusing the already-closed channels on a restart would panic with
	// "close of closed channel". Capture local copies for the goroutine so it
	// never races with a subsequent Start that reassigns the fields.
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		defer close(doneCh)

		// Track the widest line rendered so the clear at the end overwrites any
		// residue left by a previously longer message. Goroutine-local, so the
		// clear branch never reads s.message without the mutex.
		maxWidth := 0
		for {
			select {
			case <-stopCh:
				fmt.Fprintf(s.out, "\r%s\r", strings.Repeat(" ", maxWidth))
				return
			case <-ticker.C:
				s.mu.Lock()
				frame := SpinnerFrames[s.frameIdx%len(SpinnerFrames)]
				s.frameIdx++
				msg := s.message
				s.mu.Unlock()

				// Visible width = frame rune + space + message runes (ANSI codes
				// are zero-width). Grow maxWidth so shrinking messages still clear.
				if w := 2 + len([]rune(msg)); w > maxWidth {
					maxWidth = w
				}
				fmt.Fprintf(s.out, "\r%s %s", Cyan+frame+Reset, msg)
			}
		}
	}()
}

// Stop stops the spinner. Safe to call multiple times or without a prior Start.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	// Capture the current run's channels under the lock so a concurrent Start
	// (which reassigns them) cannot cause us to close/wait on the wrong ones.
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	close(stopCh)
	<-doneCh
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

// visibleWidth returns the visible width of a string (rune count without ANSI codes)
func visibleWidth(s string) int {
	stripped := stripANSI(s)
	return len([]rune(stripped))
}

// max returns the larger of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
