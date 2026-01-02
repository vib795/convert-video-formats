package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/user/convert-vid-format/pkg/types"
)

// ExpandPath expands ~ to home directory and cleans the path
func ExpandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	// Expand tilde to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	} else if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = homeDir
	}

	// Clean the path
	return filepath.Clean(path), nil
}

// IsValidFormat checks if the format is supported
func IsValidFormat(format string) bool {
	validFormats := []string{"mp4", "avi", "mov", "mkv", "webm", "flv"}
	format = strings.ToLower(format)
	for _, f := range validFormats {
		if f == format {
			return true
		}
	}
	return false
}

// IsValidQuality checks if the quality preset is valid
func IsValidQuality(quality string) bool {
	validQualities := []string{"high", "medium", "low"}
	quality = strings.ToLower(quality)
	for _, q := range validQualities {
		if q == quality {
			return true
		}
	}
	return false
}

// CheckFFmpegInstalled checks if FFmpeg is installed
func CheckFFmpegInstalled() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg is not installed or not in PATH. Please install FFmpeg to use this tool")
	}
	return nil
}

// IsVideoFile checks if a file has a video extension
func IsVideoFile(filename string) bool {
	videoExtensions := []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv", ".m4v", ".mpg", ".mpeg", ".wmv"}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true
		}
	}
	return false
}

// GetOutputPath generates the output path based on input and options
func GetOutputPath(inputPath string, opts *types.ConversionOptions) string {
	if opts.OutputPath != "" {
		return opts.OutputPath
	}

	// Generate default output path
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return fmt.Sprintf("%s_converted.%s", base, opts.Format)
}

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetVideoFiles returns all video files in a directory
func GetVideoFiles(dirPath string) ([]string, error) {
	var videoFiles []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if IsVideoFile(entry.Name()) {
			videoFiles = append(videoFiles, filepath.Join(dirPath, entry.Name()))
		}
	}

	return videoFiles, nil
}
