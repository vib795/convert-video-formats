package progress

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Global state for cleanup
var (
	cleanupMu       sync.Mutex
	activeSpinner   *Spinner
	activeBar       *ProgressBar
	activeBatch     *BatchProgress
	signalRegistered bool
)

// RegisterCleanupHandler sets up signal handling for graceful terminal cleanup
func RegisterCleanupHandler() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	if signalRegistered {
		return
	}
	signalRegistered = true

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		CleanupTerminal()
		os.Exit(130) // Standard exit code for SIGINT
	}()
}

// CleanupTerminal cleans up any active progress displays and resets terminal
func CleanupTerminal() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	// Stop any active progress indicators
	if activeSpinner != nil {
		activeSpinner.Stop()
		activeSpinner = nil
	}
	if activeBar != nil {
		activeBar.Stop()
		activeBar = nil
	}
	if activeBatch != nil {
		activeBatch.clearDisplay()
		activeBatch = nil
	}

	// Reset terminal state
	fmt.Print("\r\033[K")     // Clear current line
	fmt.Print("\033[?25h")    // Ensure cursor is visible
}

// setActiveSpinner registers a spinner for cleanup
func setActiveSpinner(s *Spinner) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activeSpinner = s
}

// setActiveBar registers a progress bar for cleanup
func setActiveBar(b *ProgressBar) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activeBar = b
}

// setActiveBatch registers a batch progress for cleanup
func setActiveBatch(bp *BatchProgress) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	activeBatch = bp
}

// Spinner provides visual feedback during conversion
type Spinner struct {
	message string
	frames  []string
	delay   time.Duration
	active  bool
	mu      sync.Mutex
	done    chan bool
}

// NewSpinner creates a new spinner with default settings
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:   100 * time.Millisecond,
		active:  false,
		done:    make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	// Register for cleanup on interrupt
	setActiveSpinner(s)

	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				fmt.Printf("\r%s %s", s.frames[i], s.message)
				s.mu.Unlock()
				i = (i + 1) % len(s.frames)
				time.Sleep(s.delay)
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.done <- true
	fmt.Print("\r" + strings.Repeat(" ", len(s.message)+5) + "\r")
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Success displays a success message and stops the spinner
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Printf("✓ %s\n", message)
}

// Error displays an error message and stops the spinner
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Printf("✗ %s\n", message)
}

// Info displays an info message
func Info(message string) {
	fmt.Printf("ℹ %s\n", message)
}

// Warning displays a warning message
func Warning(message string) {
	fmt.Printf("⚠ %s\n", message)
}

// ProgressBar shows a progress bar with percentage
type ProgressBar struct {
	message  string
	total    float64
	current  float64
	width    int
	mu       sync.Mutex
	active   bool
	done     chan bool
	lastLine string
}

// NewProgressBar creates a new progress bar
func NewProgressBar(message string) *ProgressBar {
	return &ProgressBar{
		message: message,
		width:   40,
		active:  false,
		done:    make(chan bool),
	}
}

// Start begins the progress bar and shows initial state
func (p *ProgressBar) Start() {
	p.mu.Lock()
	if p.active {
		p.mu.Unlock()
		return
	}
	p.active = true
	p.mu.Unlock()

	// Register for cleanup on interrupt
	setActiveBar(p)

	// Show initial state (0% progress)
	p.showInitial()
}

// showInitial displays the initial progress bar state
func (p *ProgressBar) showInitial() {
	p.mu.Lock()
	defer p.mu.Unlock()

	line := fmt.Sprintf("\r%s [00:00 elapsed, 0 B written]", p.message)
	fmt.Print(line)
	p.lastLine = line
}

// Update updates the progress bar with current progress
func (p *ProgressBar) Update(current, total float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return
	}

	p.current = current
	p.total = total

	var percentage float64
	if total > 0 {
		percentage = (current / total) * 100
	}

	// Calculate filled portion
	filled := int((percentage / 100) * float64(p.width))
	if filled > p.width {
		filled = p.width
	}

	// Build progress bar
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	// Format time
	currentStr := formatDuration(current)
	totalStr := formatDuration(total)

	// Create the line
	line := fmt.Sprintf("\r%s [%s] %.1f%% (%s/%s)", p.message, bar, percentage, currentStr, totalStr)

	// Clear previous line if new one is shorter
	if len(p.lastLine) > len(line) {
		fmt.Print("\r" + strings.Repeat(" ", len(p.lastLine)) + "\r")
	}

	fmt.Print(line)
	p.lastLine = line
}

// UpdateWithMessage updates with a custom message (indeterminate progress)
func (p *ProgressBar) UpdateWithMessage(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return
	}

	line := fmt.Sprintf("\r%s", msg)

	// Clear previous line if new one is shorter
	if len(p.lastLine) > len(line) {
		fmt.Print("\r" + strings.Repeat(" ", len(p.lastLine)) + "\r")
	}

	fmt.Print(line)
	p.lastLine = line
}

// UpdateWithElapsedAndSize updates with elapsed time and file size (reliable progress display)
func (p *ProgressBar) UpdateWithElapsedAndSize(elapsed time.Duration, fileSize int64, inputSize int64, totalDuration float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return
	}

	// Format elapsed time
	elapsedSecs := int(elapsed.Seconds())
	elapsedStr := fmt.Sprintf("%02d:%02d", elapsedSecs/60, elapsedSecs%60)

	// Format file sizes
	currentSizeStr := formatFileSize(fileSize)
	inputSizeStr := formatFileSize(inputSize)

	// Format total duration
	totalStr := formatDuration(totalDuration)

	// Create the line with elapsed time and file size comparison
	line := fmt.Sprintf("\r%s [%s elapsed, %s / %s] (duration: %s)", p.message, elapsedStr, currentSizeStr, inputSizeStr, totalStr)

	// Clear previous line if new one is shorter
	if len(p.lastLine) > len(line) {
		fmt.Print("\r" + strings.Repeat(" ", len(p.lastLine)) + "\r")
	}

	fmt.Print(line)
	p.lastLine = line
}

// formatFileSize formats bytes into human readable format
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// Stop stops the progress bar
func (p *ProgressBar) Stop() {
	p.mu.Lock()
	if !p.active {
		p.mu.Unlock()
		return
	}
	p.active = false
	p.mu.Unlock()

	// Clear the line
	if p.lastLine != "" {
		fmt.Print("\r" + strings.Repeat(" ", len(p.lastLine)) + "\r")
	}
}

// Success displays a success message and stops the progress bar
func (p *ProgressBar) Success(message string) {
	p.Stop()
	fmt.Printf("✓ %s\n", message)
}

// Error displays an error message and stops the progress bar
func (p *ProgressBar) Error(message string) {
	p.Stop()
	fmt.Printf("✗ %s\n", message)
}

// formatDuration formats seconds into HH:MM:SS or MM:SS
func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}

	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

// BatchProgress tracks progress for multiple concurrent conversions
type BatchProgress struct {
	total          int
	completed      int
	failed         int
	active         map[string]*fileProgress
	mu             sync.Mutex
	lastUpdate     time.Time
	done           chan bool
	lastLineCount  int
}

type fileProgress struct {
	filename    string
	percentage  float64
	currentTime float64
	totalTime   float64
	elapsed     time.Duration
	fileSize    int64
	inputSize   int64
}

// NewBatchProgress creates a new batch progress tracker
func NewBatchProgress(total int) *BatchProgress {
	return &BatchProgress{
		total:      total,
		active:     make(map[string]*fileProgress),
		lastUpdate: time.Now(),
		done:       make(chan bool),
	}
}

// Start begins the batch progress display
func (bp *BatchProgress) Start() {
	// Register for cleanup on interrupt
	setActiveBatch(bp)
	go bp.displayLoop()
}

// UpdateFile updates progress for a specific file
func (bp *BatchProgress) UpdateFile(filename string, current, total float64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	percentage := 0.0
	if total > 0 {
		percentage = (current / total) * 100
	}

	bp.active[filename] = &fileProgress{
		filename:    filename,
		percentage:  percentage,
		currentTime: current,
		totalTime:   total,
	}
}

// UpdateFileWithSize updates progress for a specific file with elapsed time and file size
func (bp *BatchProgress) UpdateFileWithSize(filename string, elapsed time.Duration, fileSize int64, inputSize int64, totalDuration float64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.active[filename] = &fileProgress{
		filename:  filename,
		totalTime: totalDuration,
		elapsed:   elapsed,
		fileSize:  fileSize,
		inputSize: inputSize,
	}
}

// CompleteFile marks a file as completed
func (bp *BatchProgress) CompleteFile(filename string, success bool) {
	bp.mu.Lock()

	// Clear the current progress display
	if bp.lastLineCount > 0 {
		bp.clearAndResetDisplay()
	}

	delete(bp.active, filename)
	if success {
		bp.completed++
		fmt.Printf("✓ Converted %s\n", filename)
	} else {
		bp.failed++
		fmt.Printf("✗ Failed to convert %s\n", filename)
	}

	bp.mu.Unlock()
}

// Stop stops the batch progress display
func (bp *BatchProgress) Stop() {
	bp.done <- true
	time.Sleep(100 * time.Millisecond) // Give time for final display
}

// displayLoop continuously updates the display
func (bp *BatchProgress) displayLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-bp.done:
			bp.clearDisplay()
			return
		case <-ticker.C:
			bp.display()
		}
	}
}

// display shows the current batch progress
func (bp *BatchProgress) display() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if len(bp.active) == 0 {
		// Clear any previous display
		if bp.lastLineCount > 0 {
			bp.clearAndResetDisplay()
		}
		return
	}

	// Move cursor up to previous display position and clear
	if bp.lastLineCount > 0 {
		// Move up to start of previous display
		for i := 0; i < bp.lastLineCount; i++ {
			fmt.Print("\033[A") // Move up one line
		}
		// Clear all lines
		for i := 0; i < bp.lastLineCount; i++ {
			fmt.Print("\033[2K")  // Clear entire line
			if i < bp.lastLineCount-1 {
				fmt.Print("\033[B") // Move down to next line
			}
		}
		// Move back to start
		for i := 0; i < bp.lastLineCount-1; i++ {
			fmt.Print("\033[A") // Move up
		}
		fmt.Print("\r") // Move to start of line
	}

	// Show overall progress
	fmt.Printf("[%d/%d completed, %d active, %d failed]\n",
		bp.completed, bp.total, len(bp.active), bp.failed)

	// Show each active file (sorted for consistent display)
	activeFiles := make([]string, 0, len(bp.active))
	for filename := range bp.active {
		activeFiles = append(activeFiles, filename)
	}
	// Sort for consistent ordering
	for i := 0; i < len(activeFiles); i++ {
		for j := i + 1; j < len(activeFiles); j++ {
			if activeFiles[i] > activeFiles[j] {
				activeFiles[i], activeFiles[j] = activeFiles[j], activeFiles[i]
			}
		}
	}

	for _, filename := range activeFiles {
		fp := bp.active[filename]
		if fp.elapsed > 0 || fp.fileSize > 0 {
			// New format: elapsed time and file size
			elapsedSecs := int(fp.elapsed.Seconds())
			elapsedStr := fmt.Sprintf("%02d:%02d", elapsedSecs/60, elapsedSecs%60)
			currentSizeStr := formatFileSize(fp.fileSize)
			inputSizeStr := formatFileSize(fp.inputSize)
			totalStr := formatDuration(fp.totalTime)
			fmt.Printf("  %s [%s elapsed, %s / %s] (duration: %s)\n",
				truncateFilename(fp.filename, 40), elapsedStr, currentSizeStr, inputSizeStr, totalStr)
		} else {
			// Old format: percentage-based
			bar := buildProgressBar(fp.percentage, 30)
			currentStr := formatDuration(fp.currentTime)
			totalStr := formatDuration(fp.totalTime)
			fmt.Printf("  %s [%s] %.1f%% (%s/%s)\n",
				truncateFilename(fp.filename, 40), bar, fp.percentage, currentStr, totalStr)
		}
	}

	// Track how many lines we drew
	lineCount := 1 + len(bp.active) // 1 summary line + N file lines
	bp.lastLineCount = lineCount
}

// clearAndResetDisplay clears the display and resets state
func (bp *BatchProgress) clearAndResetDisplay() {
	// Move up to start of display
	for i := 0; i < bp.lastLineCount; i++ {
		fmt.Print("\033[A")
	}
	// Clear all lines
	for i := 0; i < bp.lastLineCount; i++ {
		fmt.Print("\033[2K\n")
	}
	// Move back up
	for i := 0; i < bp.lastLineCount; i++ {
		fmt.Print("\033[A")
	}
	bp.lastLineCount = 0
}

// clearDisplay clears the current display
func (bp *BatchProgress) clearDisplay() {
	if bp.lastLineCount > 0 {
		bp.clearAndResetDisplay()
	}
}

// buildProgressBar creates a progress bar string
func buildProgressBar(percentage float64, width int) string {
	filled := int((percentage / 100) * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// truncateFilename truncates a filename to fit within maxLen
func truncateFilename(filename string, maxLen int) string {
	if len(filename) <= maxLen {
		return filename
	}
	return filename[:maxLen-3] + "..."
}
