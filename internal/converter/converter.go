package converter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/user/convert-vid-format/internal/progress"
	"github.com/user/convert-vid-format/internal/utils"
	"github.com/user/convert-vid-format/pkg/types"
)

// Converter handles video conversion operations
type Converter struct {
	options *types.ConversionOptions
}

// NewConverter creates a new Converter instance
func NewConverter(opts *types.ConversionOptions) *Converter {
	return &Converter{
		options: opts,
	}
}

// Convert performs the conversion based on the options
func (c *Converter) Convert() error {
	// Check if ffmpeg is installed
	if err := utils.CheckFFmpegInstalled(); err != nil {
		return err
	}

	// Determine if input is a file or directory
	info, err := os.Stat(c.options.InputPath)
	if err != nil {
		return fmt.Errorf("input path does not exist: %w", err)
	}

	if info.IsDir() {
		return c.convertDirectory()
	}
	return c.convertSingleFile(c.options.InputPath, c.options.OutputPath)
}

// convertSingleFile converts a single video file
func (c *Converter) convertSingleFile(inputPath, outputPath string) error {
	// Validate input file
	if !utils.IsVideoFile(inputPath) {
		return fmt.Errorf("%s does not appear to be a video file", inputPath)
	}

	// Generate output path if not specified
	if outputPath == "" {
		outputPath = utils.GetOutputPath(inputPath, c.options)
	}

	// Check if output file exists
	if utils.FileExists(outputPath) && !c.options.Overwrite {
		return fmt.Errorf("output file %s already exists. Use --overwrite to replace it", outputPath)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := utils.EnsureDirectoryExists(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get video duration for progress tracking
	duration, err := c.getVideoDuration(inputPath)
	if err != nil {
		// If we can't get duration, fall back to spinner
		return c.convertWithSpinner(inputPath, outputPath)
	}

	// Start progress bar
	bar := progress.NewProgressBar(fmt.Sprintf("Converting %s", filepath.Base(inputPath)))
	bar.Start()

	// Build ffmpeg command
	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	// Capture stderr for progress
	stderr, err := cmd.StderrPipe()
	if err != nil {
		bar.Stop()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		bar.Stop()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress from stderr
	go c.parseProgress(stderr, bar, duration)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		bar.Error(fmt.Sprintf("Failed to convert %s", filepath.Base(inputPath)))
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	bar.Success(fmt.Sprintf("Converted %s → %s", filepath.Base(inputPath), filepath.Base(outputPath)))
	return nil
}

// convertWithSpinner converts using a simple spinner (fallback)
func (c *Converter) convertWithSpinner(inputPath, outputPath string) error {
	spinner := progress.NewSpinner(fmt.Sprintf("Converting %s...", filepath.Base(inputPath)))
	spinner.Start()

	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	if err := cmd.Run(); err != nil {
		spinner.Error(fmt.Sprintf("Failed to convert %s", filepath.Base(inputPath)))
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	spinner.Success(fmt.Sprintf("Converted %s → %s", filepath.Base(inputPath), filepath.Base(outputPath)))
	return nil
}

// convertDirectory converts all video files in a directory
func (c *Converter) convertDirectory() error {
	// Get all video files
	videoFiles, err := utils.GetVideoFiles(c.options.InputPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(videoFiles) == 0 {
		return fmt.Errorf("no video files found in %s", c.options.InputPath)
	}

	progress.Info(fmt.Sprintf("Found %d video file(s) to convert", len(videoFiles)))

	// Ensure output directory exists
	outputDir := c.options.OutputPath
	if outputDir == "" {
		outputDir = filepath.Join(c.options.InputPath, "converted")
	}
	if err := utils.EnsureDirectoryExists(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine concurrency
	concurrent := c.options.Concurrent
	if concurrent <= 0 {
		concurrent = runtime.NumCPU()
	}

	// Convert files concurrently
	return c.convertFilesConcurrently(videoFiles, outputDir, concurrent)
}

// convertFilesConcurrently converts multiple files using worker pool pattern
func (c *Converter) convertFilesConcurrently(files []string, outputDir string, workers int) error {
	var wg sync.WaitGroup
	fileChan := make(chan string, len(files))
	resultChan := make(chan types.ConversionResult, len(files))

	// Create batch progress tracker
	batchProgress := progress.NewBatchProgress(len(files))
	batchProgress.Start()

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for inputFile := range fileChan {
				result := c.convertFileWorkerWithBatchProgress(inputFile, outputDir, batchProgress)
				resultChan <- result
			}
		}()
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	successCount := 0
	failCount := 0
	for result := range resultChan {
		if result.Success {
			successCount++
		} else {
			failCount++
		}
	}

	// Stop batch progress
	batchProgress.Stop()

	// Print summary
	fmt.Printf("\n")
	progress.Info(fmt.Sprintf("Conversion complete: %d succeeded, %d failed", successCount, failCount))

	if failCount > 0 {
		return fmt.Errorf("%d file(s) failed to convert", failCount)
	}
	return nil
}

// convertFileWorkerWithBatchProgress handles conversion with batch progress tracking
func (c *Converter) convertFileWorkerWithBatchProgress(inputPath, outputDir string, batchProgress *progress.BatchProgress) types.ConversionResult {
	result := types.ConversionResult{
		InputFile: inputPath,
		Success:   false,
	}

	// Generate output filename
	baseName := filepath.Base(inputPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.%s", nameWithoutExt, c.options.Format))
	result.OutputFile = outputPath

	// Check if output exists
	if utils.FileExists(outputPath) && !c.options.Overwrite {
		result.Error = fmt.Errorf("output file already exists")
		batchProgress.CompleteFile(baseName, false)
		return result
	}

	// Get video duration for progress tracking
	duration, err := c.getVideoDuration(inputPath)
	if err != nil {
		// If we can't get duration, use a simpler approach
		result = c.convertFileWorkerSimple(inputPath, outputPath, baseName)
		batchProgress.CompleteFile(baseName, result.Success)
		return result
	}

	// Build ffmpeg command
	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	// Capture stderr for progress
	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		batchProgress.CompleteFile(baseName, false)
		return result
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start ffmpeg: %w", err)
		batchProgress.CompleteFile(baseName, false)
		return result
	}

	// Parse progress from stderr
	go c.parseProgressBatch(stderr, batchProgress, baseName, duration)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		result.Error = err
		batchProgress.CompleteFile(baseName, false)
		return result
	}

	result.Success = true
	batchProgress.CompleteFile(baseName, true)
	return result
}

// convertFileWorker handles conversion of a single file in a worker
func (c *Converter) convertFileWorker(inputPath, outputDir string) types.ConversionResult {
	result := types.ConversionResult{
		InputFile: inputPath,
		Success:   false,
	}

	// Generate output filename
	baseName := filepath.Base(inputPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.%s", nameWithoutExt, c.options.Format))
	result.OutputFile = outputPath

	// Check if output exists
	if utils.FileExists(outputPath) && !c.options.Overwrite {
		result.Error = fmt.Errorf("output file already exists")
		progress.Warning(fmt.Sprintf("Skipped %s (output exists)", baseName))
		return result
	}

	// Get video duration for progress tracking
	duration, err := c.getVideoDuration(inputPath)
	if err != nil {
		// If we can't get duration, fall back to spinner
		return c.convertFileWorkerWithSpinner(inputPath, outputPath, baseName)
	}

	// Start progress bar
	bar := progress.NewProgressBar(fmt.Sprintf("Converting %s", baseName))
	bar.Start()

	// Build ffmpeg command
	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	// Capture stderr for progress
	stderr, err := cmd.StderrPipe()
	if err != nil {
		bar.Stop()
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		return result
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		bar.Stop()
		result.Error = fmt.Errorf("failed to start ffmpeg: %w", err)
		return result
	}

	// Parse progress from stderr
	go c.parseProgress(stderr, bar, duration)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		result.Error = err
		bar.Error(fmt.Sprintf("Failed to convert %s", baseName))
		return result
	}

	result.Success = true
	bar.Success(fmt.Sprintf("Converted %s", baseName))
	return result
}

// convertFileWorkerWithSpinner is a fallback conversion method using spinner
func (c *Converter) convertFileWorkerWithSpinner(inputPath, outputPath, baseName string) types.ConversionResult {
	result := types.ConversionResult{
		InputFile:  inputPath,
		OutputFile: outputPath,
		Success:    false,
	}

	spinner := progress.NewSpinner(fmt.Sprintf("Converting %s...", baseName))
	spinner.Start()

	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	if err := cmd.Run(); err != nil {
		result.Error = err
		spinner.Error(fmt.Sprintf("Failed to convert %s", baseName))
		return result
	}

	result.Success = true
	spinner.Success(fmt.Sprintf("Converted %s", baseName))
	return result
}

// getVideoDuration gets the duration of a video file using ffprobe
func (c *Converter) getVideoDuration(inputPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

// parseProgress parses FFmpeg's stderr output for progress information
func (c *Converter) parseProgress(stderr io.ReadCloser, bar *progress.ProgressBar, totalDuration float64) {
	defer stderr.Close()

	scanner := bufio.NewScanner(stderr)
	timeRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)

	for scanner.Scan() {
		line := scanner.Text()

		// Look for time= in the output
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			// Parse HH:MM:SS.ss format
			hours, _ := strconv.ParseFloat(matches[1], 64)
			minutes, _ := strconv.ParseFloat(matches[2], 64)
			seconds, _ := strconv.ParseFloat(matches[3], 64)

			currentTime := hours*3600 + minutes*60 + seconds

			// Update progress bar
			bar.Update(currentTime, totalDuration)
		}
	}
}

// parseProgressBatch parses FFmpeg's stderr output for batch progress tracking
func (c *Converter) parseProgressBatch(stderr io.ReadCloser, batchProgress *progress.BatchProgress, filename string, totalDuration float64) {
	defer stderr.Close()

	scanner := bufio.NewScanner(stderr)
	timeRegex := regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2}\.\d{2})`)

	for scanner.Scan() {
		line := scanner.Text()

		// Look for time= in the output
		matches := timeRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			// Parse HH:MM:SS.ss format
			hours, _ := strconv.ParseFloat(matches[1], 64)
			minutes, _ := strconv.ParseFloat(matches[2], 64)
			seconds, _ := strconv.ParseFloat(matches[3], 64)

			currentTime := hours*3600 + minutes*60 + seconds

			// Update batch progress
			batchProgress.UpdateFile(filename, currentTime, totalDuration)
		}
	}
}

// convertFileWorkerSimple performs conversion without progress tracking (fallback)
func (c *Converter) convertFileWorkerSimple(inputPath, outputPath, baseName string) types.ConversionResult {
	result := types.ConversionResult{
		InputFile:  inputPath,
		OutputFile: outputPath,
		Success:    false,
	}

	args := c.buildFFmpegArgs(inputPath, outputPath)
	cmd := exec.Command("ffmpeg", args...)

	if err := cmd.Run(); err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	return result
}

// buildFFmpegArgs builds the ffmpeg command arguments based on options
func (c *Converter) buildFFmpegArgs(inputPath, outputPath string) []string {
	args := []string{
		"-i", inputPath,
		"-progress", "pipe:2", // Output progress to stderr
		"-stats",              // Show statistics
	}

	// Add quality settings
	switch strings.ToLower(c.options.Quality) {
	case "high":
		args = append(args, "-crf", "18", "-preset", "slow")
	case "medium":
		args = append(args, "-crf", "23", "-preset", "medium")
	case "low":
		args = append(args, "-crf", "28", "-preset", "fast")
	default:
		// Default to medium quality
		args = append(args, "-crf", "23", "-preset", "medium")
	}

	// Add overwrite flag
	if c.options.Overwrite {
		args = append(args, "-y")
	} else {
		args = append(args, "-n")
	}

	// Add output path
	args = append(args, outputPath)

	return args
}
