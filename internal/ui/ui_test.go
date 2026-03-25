package ui

import (
	"bytes"
"strings"
	"testing"
	"time"
)

func TestUI_Success(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Success("Operation completed")

	output := buf.String()
	if !strings.Contains(output, SymbolCheck) {
		t.Errorf("Success output should contain check symbol, got: %s", output)
	}
	if !strings.Contains(output, "Operation completed") {
		t.Errorf("Success output should contain message, got: %s", output)
	}
}

func TestUI_Error(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Error("Something failed")

	output := buf.String()
	if !strings.Contains(output, SymbolCross) {
		t.Errorf("Error output should contain cross symbol, got: %s", output)
	}
	if !strings.Contains(output, "Something failed") {
		t.Errorf("Error output should contain message, got: %s", output)
	}
}

func TestUI_Warning(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Warning("Be careful")

	output := buf.String()
	if !strings.Contains(output, SymbolWarning) {
		t.Errorf("Warning output should contain warning symbol, got: %s", output)
	}
	if !strings.Contains(output, "Be careful") {
		t.Errorf("Warning output should contain message, got: %s", output)
	}
}

func TestUI_Info(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Info("FYI")

	output := buf.String()
	if !strings.Contains(output, SymbolInfo) {
		t.Errorf("Info output should contain info symbol, got: %s", output)
	}
	if !strings.Contains(output, "FYI") {
		t.Errorf("Info output should contain message, got: %s", output)
	}
}

func TestUI_Step(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Step(1, 3, "First step")

	output := buf.String()
	if !strings.Contains(output, "Step 1 of 3") {
		t.Errorf("Step output should contain step indicator, got: %s", output)
	}
	if !strings.Contains(output, "First step") {
		t.Errorf("Step output should contain title, got: %s", output)
	}
}

func TestUI_Header(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Header("Test Header", "Subtitle here")

	output := buf.String()
	if !strings.Contains(output, BoxTopLeft) {
		t.Errorf("Header should contain box characters, got: %s", output)
	}
	if !strings.Contains(output, "Test Header") {
		t.Errorf("Header should contain title, got: %s", output)
	}
	if !strings.Contains(output, "Subtitle here") {
		t.Errorf("Header should contain subtitle, got: %s", output)
	}
}

func TestUI_HeaderWithoutSubtitle(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Header("Title Only", "")

	output := buf.String()
	if !strings.Contains(output, "Title Only") {
		t.Errorf("Header should contain title, got: %s", output)
	}
}

func TestUI_Box(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Box([]string{"Line 1", "Line 2", "Line 3"})

	output := buf.String()
	if !strings.Contains(output, BoxTopLeftAlt) {
		t.Errorf("Box should contain box characters, got: %s", output)
	}
	if !strings.Contains(output, "Line 1") {
		t.Errorf("Box should contain Line 1, got: %s", output)
	}
	if !strings.Contains(output, "Line 2") {
		t.Errorf("Box should contain Line 2, got: %s", output)
	}
	if !strings.Contains(output, "Line 3") {
		t.Errorf("Box should contain Line 3, got: %s", output)
	}
}

func TestUI_BoxEmpty(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	u.Box([]string{})

	output := buf.String()
	if output != "" {
		t.Errorf("Empty box should produce no output, got: %s", output)
	}
}

func TestUI_Menu(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	lines := u.Menu([]string{"Option A", "Option B"}, true)

	if len(lines) != 3 { // 2 options + manual entry
		t.Errorf("Menu should have 3 lines, got: %d", len(lines))
	}

	// Check that lines contain expected content (with ANSI stripped)
	combined := strings.Join(lines, "\n")
	if !strings.Contains(stripANSI(combined), "1.") {
		t.Errorf("Menu should contain option 1, got: %s", combined)
	}
	if !strings.Contains(stripANSI(combined), "Option A") {
		t.Errorf("Menu should contain Option A, got: %s", combined)
	}
	if !strings.Contains(stripANSI(combined), "0.") {
		t.Errorf("Menu should contain manual option 0, got: %s", combined)
	}
}

func TestUI_MenuWithoutManualOption(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	lines := u.Menu([]string{"Option A", "Option B"}, false)

	if len(lines) != 2 {
		t.Errorf("Menu without manual option should have 2 lines, got: %d", len(lines))
	}
}

func TestUI_Prompt(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	prompt := u.Prompt("Enter value", "default")
	if !strings.Contains(stripANSI(prompt), "Enter value") {
		t.Errorf("Prompt should contain label, got: %s", prompt)
	}
	if !strings.Contains(stripANSI(prompt), "default") {
		t.Errorf("Prompt should contain default value, got: %s", prompt)
	}
}

func TestUI_PromptWithoutDefault(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	prompt := u.Prompt("Enter value", "")
	if !strings.Contains(stripANSI(prompt), "Enter value:") {
		t.Errorf("Prompt without default should end with colon, got: %s", prompt)
	}
}

func TestUI_SummaryBox(t *testing.T) {
	var buf bytes.Buffer
	u := New(&buf)

	items := map[string]string{
		"Key1": "Value1",
		"Key2": "Value2",
	}
	order := []string{"Key1", "Key2"}

	u.SummaryBox("Summary Title", items, order)

	output := buf.String()
	if !strings.Contains(output, BoxTopLeft) {
		t.Errorf("SummaryBox should contain box characters, got: %s", output)
	}
	if !strings.Contains(output, "Summary Title") {
		t.Errorf("SummaryBox should contain title, got: %s", output)
	}
	if !strings.Contains(output, "Value1") {
		t.Errorf("SummaryBox should contain Value1, got: %s", output)
	}
	if !strings.Contains(output, "Value2") {
		t.Errorf("SummaryBox should contain Value2, got: %s", output)
	}
}

func TestUI_NoColor(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithOptions(&buf, true) // noColor = true

	u.Success("No colors")

	output := buf.String()
	// Should not contain ANSI escape codes
	if strings.Contains(output, "\033[") {
		t.Errorf("NoColor output should not contain ANSI codes, got: %s", output)
	}
	if !strings.Contains(output, SymbolCheck) {
		t.Errorf("NoColor output should still contain symbol, got: %s", output)
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"\033[31mred\033[0m", "red"},
		{"\033[1m\033[32mbold green\033[0m", "bold green"},
		{"", ""},
		{"no escape", "no escape"},
	}

	for _, tt := range tests {
		result := stripANSI(tt.input)
		if result != tt.expected {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{0, 0, 0},
		{-1, 1, 1},
	}

	for _, tt := range tests {
		result := max(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"plain text", 10},
		{"✓ check", 7},                // checkmark is 1 visible char
		{"\033[32m✓\033[0m check", 7}, // with ANSI codes
		{"", 0},
		{"hello", 5},
		{"日本語", 3},                          // 3 Japanese characters
		{"\033[1m\033[32m✓ Done\033[0m", 6}, // bold green checkmark
	}

	for _, tt := range tests {
		result := visibleWidth(tt.input)
		if result != tt.expected {
			t.Errorf("visibleWidth(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestSpinner_StartStop(t *testing.T) {
	t.Run("starts and stops without panic", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Loading...")

		// This should not panic
		s.Start()

		// Give the spinner a moment to run
		time.Sleep(100 * time.Millisecond)

		// Stop should not panic
		s.Stop()
	})

	t.Run("double start is safe", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Test")

		s.Start()
		s.Start() // Should be safe to call twice

		time.Sleep(50 * time.Millisecond)
		s.Stop()
	})

	t.Run("double stop is safe", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Test")

		s.Start()
		time.Sleep(50 * time.Millisecond)

		s.Stop()
		s.Stop() // Should be safe to call twice
	})

	t.Run("stop without start is safe", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Never started")

		// This should not panic even though we never called Start
		s.Stop()
	})
}

func TestSpinner_UpdateMessage(t *testing.T) {
	t.Run("updates message while running", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Initial message")

		s.Start()
		time.Sleep(50 * time.Millisecond)

		// Update the message while spinner is running
		s.UpdateMessage("Updated message")
		time.Sleep(100 * time.Millisecond)

		s.Stop()

		// Verify the output contains the updated message
		output := buf.String()
		if !strings.Contains(output, "Updated message") {
			t.Errorf("Output should contain 'Updated message', got: %s", output)
		}
	})

	t.Run("can update message before start", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Initial")

		// Update before start - should not panic
		s.UpdateMessage("Changed before start")

		s.Start()
		time.Sleep(100 * time.Millisecond)
		s.Stop()

		output := buf.String()
		if !strings.Contains(output, "Changed before start") {
			t.Errorf("Output should contain updated message, got: %s", output)
		}
	})

	t.Run("can update message after stop", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Initial")

		s.Start()
		time.Sleep(50 * time.Millisecond)
		s.Stop()

		// Update after stop - should not panic
		s.UpdateMessage("After stop")
	})
}

func TestSpinner_OutputFormat(t *testing.T) {
	t.Run("clears line on stop", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Test")

		s.Start()
		time.Sleep(100 * time.Millisecond)
		s.Stop()

		output := buf.String()
		// Should contain carriage return from clearing
		if !strings.Contains(output, "\r") {
			t.Errorf("Output should contain carriage return for line clearing, got: %s", output)
		}
	})

	t.Run("uses spinner frames", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Loading")

		s.Start()
		time.Sleep(200 * time.Millisecond) // Let it cycle through a few frames
		s.Stop()

		output := buf.String()
		// At least one spinner frame should appear
		hasFrame := false
		for _, frame := range SpinnerFrames {
			if strings.Contains(output, frame) {
				hasFrame = true
				break
			}
		}
		if !hasFrame {
			t.Errorf("Output should contain at least one spinner frame, got: %s", output)
		}
	})
}

func TestNewSpinner(t *testing.T) {
	t.Run("creates spinner with message", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "Test message")

		if s == nil {
			t.Fatal("NewSpinner should return non-nil spinner")
		}
		if s.message != "Test message" {
			t.Errorf("Expected message 'Test message', got %q", s.message)
		}
		if s.out != &buf {
			t.Error("Spinner output should be set to provided writer")
		}
		if s.active {
			t.Error("Spinner should not be active initially")
		}
	})

	t.Run("creates spinner with empty message", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner(&buf, "")

		if s.message != "" {
			t.Errorf("Expected empty message, got %q", s.message)
		}
	})
}

// ============================================================================
// Header Multi-byte UTF-8 Tests
// ============================================================================

func TestHeader_MultiByte_AlignedBorders(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithOptions(&buf, true) // No color to simplify output parsing

	// Use multi-byte UTF-8 title: 3 runes, 9 bytes
	title := "テスト" // 3 CJK characters, each 3 bytes
	u.Header(title, "")

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines (top, title, bottom), got %d", len(lines))
	}

	// All lines should have the same visible width
	topWidth := visibleWidth(lines[0])
	titleLineWidth := visibleWidth(lines[1])
	bottomWidth := visibleWidth(lines[len(lines)-1])

	if topWidth != titleLineWidth {
		t.Errorf("Top border width (%d) != title line width (%d)\nTop:   %q\nTitle: %q",
			topWidth, titleLineWidth, lines[0], lines[1])
	}
	if topWidth != bottomWidth {
		t.Errorf("Top border width (%d) != bottom border width (%d)", topWidth, bottomWidth)
	}
}

func TestHeader_MultiByte_WithSubtitle(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithOptions(&buf, true)

	u.Header("テスト", "サブタイトル") // 3 runes title, 6 runes subtitle

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) < 4 {
		t.Fatalf("Expected at least 4 lines (top, title, subtitle, bottom), got %d", len(lines))
	}

	// All lines should have the same visible width
	widths := make([]int, len(lines))
	for i, line := range lines {
		widths[i] = visibleWidth(line)
	}

	for i := 1; i < len(widths); i++ {
		if widths[i] != widths[0] {
			t.Errorf("Line %d width (%d) != line 0 width (%d)\nLine 0: %q\nLine %d: %q",
				i, widths[i], widths[0], lines[0], i, lines[i])
		}
	}
}
