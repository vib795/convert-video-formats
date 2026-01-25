package utils

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DependencyStatus represents the status of a dependency check
type DependencyStatus struct {
	Name      string
	Available bool
	Required  bool
	Message   string
}

// CheckAllDependencies performs comprehensive dependency checks
// Returns an error if required dependencies are missing, or nil with warnings printed
func CheckAllDependencies() error {
	// Check ffmpeg (required)
	if !isCommandAvailable("ffmpeg") {
		return fmt.Errorf("%s", getFFmpegMissingMessage())
	}

	// Check ffprobe (required, usually bundled with ffmpeg)
	if !isCommandAvailable("ffprobe") {
		return fmt.Errorf("%s", getFFprobeMissingMessage())
	}

	// Check for AV1 decoder support (optional, but warn if missing)
	if !HasAV1DecoderSupport() {
		printAV1Warning()
	}

	return nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// HasAV1DecoderSupport checks if ffmpeg has AV1 decoder support (libdav1d or av1)
func HasAV1DecoderSupport() bool {
	cmd := exec.Command("ffmpeg", "-decoders")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	outputStr := string(output)
	return strings.Contains(outputStr, "libdav1d") || strings.Contains(outputStr, " av1 ")
}

// HasEncoder checks if ffmpeg has a specific encoder
func HasEncoder(encoder string) bool {
	cmd := exec.Command("ffmpeg", "-encoders")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), encoder)
}

// GetInputVideoCodec detects the video codec of an input file using ffprobe
func GetInputVideoCodec(inputPath string) string {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getFFmpegMissingMessage returns OS-specific install instructions for ffmpeg
func getFFmpegMissingMessage() string {
	base := "ffmpeg is not installed.\n\n"

	switch runtime.GOOS {
	case "darwin":
		return base + `To install ffmpeg on macOS:
  brew install ffmpeg

This will also install all required codecs including AV1 support.`

	case "linux":
		return base + `To install ffmpeg on Linux:
  Ubuntu/Debian:  sudo apt install ffmpeg
  Fedora/RHEL:    sudo dnf install ffmpeg
  Arch Linux:     sudo pacman -S ffmpeg

For AV1 support, ensure your distribution's ffmpeg includes libdav1d.`

	case "windows":
		return base + `To install ffmpeg on Windows:
  Using Chocolatey:  choco install ffmpeg
  Using Winget:      winget install ffmpeg
  Using Scoop:       scoop install ffmpeg

Or download from: https://ffmpeg.org/download.html`

	default:
		return base + `Please install ffmpeg for your system.
Visit https://ffmpeg.org/download.html for instructions.`
	}
}

// getFFprobeMissingMessage returns instructions when ffprobe is missing
func getFFprobeMissingMessage() string {
	return `ffprobe is not installed (usually bundled with ffmpeg).

Please reinstall ffmpeg to include ffprobe:
` + getInstallCommand()
}

// getInstallCommand returns the OS-specific install command
func getInstallCommand() string {
	switch runtime.GOOS {
	case "darwin":
		return "  brew reinstall ffmpeg"
	case "linux":
		return "  sudo apt install ffmpeg  (or equivalent for your distro)"
	case "windows":
		return "  choco install ffmpeg --force"
	default:
		return "  Reinstall ffmpeg from https://ffmpeg.org/download.html"
	}
}

// printAV1Warning prints a warning about missing AV1 decoder support
func printAV1Warning() {
	warning := `
Warning: AV1 decoder (libdav1d) not found in your ffmpeg installation.
         Videos encoded with AV1 codec (common in recent YouTube downloads)
         may fail to convert.

`
	switch runtime.GOOS {
	case "darwin":
		warning += "To fix this on macOS:\n  brew reinstall ffmpeg\n"
	case "linux":
		warning += "To fix this on Linux:\n  Reinstall ffmpeg with AV1/dav1d support enabled\n"
	case "windows":
		warning += "To fix this on Windows:\n  Download ffmpeg with libdav1d from https://github.com/BtbN/FFmpeg-Builds/releases\n"
	}

	fmt.Println(warning)
}

// GetAV1ErrorMessage returns a helpful error message when AV1 conversion fails
func GetAV1ErrorMessage() string {
	base := `AV1 codec detected but conversion failed. Your ffmpeg may not have AV1 decoder support.

`
	switch runtime.GOOS {
	case "darwin":
		return base + `To fix this on macOS:
  brew reinstall ffmpeg

This will install libdav1d (fast AV1 decoder).`

	case "linux":
		return base + `To fix this on Linux:
  Reinstall ffmpeg with AV1/libdav1d support.

  Ubuntu/Debian: sudo apt install ffmpeg libdav1d-dev
  Fedora:        sudo dnf install ffmpeg libdav1d`

	case "windows":
		return base + `To fix this on Windows:
  Download ffmpeg with libdav1d support from:
  https://github.com/BtbN/FFmpeg-Builds/releases

  Choose a build that includes "gpl" in the name.`

	default:
		return base + "Please reinstall ffmpeg with AV1 decoder support (libdav1d)."
	}
}
