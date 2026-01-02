# convert-vid

A fast and efficient command-line tool for converting video files between different formats. Built with Go and powered by FFmpeg.

## Features

- **Single File Conversion**: Convert individual video files
- **Batch Conversion**: Convert entire directories of videos concurrently
- **Multiple Formats**: Support for mp4, avi, mov, mkv, webm, flv
- **Quality Presets**: Choose between high, medium, and low quality settings
- **Progress Tracking**: Visual feedback during conversion
- **Concurrent Processing**: Multi-threaded batch conversions for faster processing
- **Simple CLI**: Easy-to-use command-line interface

## Prerequisites

**FFmpeg** must be installed on your system:

### macOS
```bash
brew install ffmpeg
```

### Ubuntu/Debian
```bash
sudo apt update
sudo apt install ffmpeg
```

### Windows
Download from [ffmpeg.org](https://ffmpeg.org/download.html) or use chocolatey:
```bash
choco install ffmpeg
```

### Verify Installation
```bash
ffmpeg -version
```

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/user/convert-vid-format.git
cd convert-vid-format
```

2. Build the binary:
```bash
go build -o convert-vid
```

3. (Optional) Move to your PATH:
```bash
# macOS/Linux
sudo mv convert-vid /usr/local/bin/

# Or add to your PATH
export PATH=$PATH:$(pwd)
```

### Using Go Install

```bash
go install github.com/user/convert-vid-format@latest
```

## Usage

### Basic Commands

```bash
# Convert a single file to mp4
convert-vid convert input.avi --format mp4 --output output.mp4

# Convert using short flags
convert-vid convert input.mov -f mp4 -o output.mp4

# Convert with automatic output naming
convert-vid convert input.avi -f mp4

# Convert all videos in a folder
convert-vid convert ./videos --format mp4 --output ./converted

# Convert with high quality
convert-vid convert input.mov -f mp4 -q high

# Overwrite existing files
convert-vid convert input.avi -f mp4 --overwrite

# Specify concurrent conversions for batch processing
convert-vid convert ./videos -f mp4 -c 4
```

### Handling Paths with Spaces

**Important**: When working with file or folder paths that contain spaces, you must wrap them in quotes:

```bash
# Method 1: Use quotes with positional argument (recommended for simple cases)
convert-vid convert "My Video.mov" -f mp4 -o "Output Video.mp4"
convert-vid convert "~/Downloads/My Videos/lecture.webm" -f mp4

# Method 2: Use --input flag (recommended for complex paths)
convert-vid convert --input "~/Downloads/My Videos/lecture.webm" -f mp4 -o "output.mp4"
convert-vid convert -i "Backend from first principles/01 - Roadmap.webm" -f mp4

# Method 3: For directories with spaces
convert-vid convert --input "~/Downloads/My Video Collection" -f mp4 -o "./Converted Videos"

# Real-world example
convert-vid convert \
  --input "~/Downloads/pull-vids/Backend from first principles/01 - 1. Roadmap for backend from first principles.webm" \
  --format mp4 \
  --output "1. Roadmap for backend from first principles.mp4"
```

**Tips**:
- Always use **double quotes** (`"`) around paths with spaces
- The `--input` or `-i` flag is often clearer than positional arguments for complex paths
- You can use `\` at the end of a line to continue a long command on the next line

### Command Reference

```bash
convert-vid convert [input] [flags]
```

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--input` | `-i` | Input video file or directory path (recommended for paths with spaces) | - |
| `--format` | `-f` | Target video format (mp4, avi, mov, mkv, webm, flv) | mp4 |
| `--output` | `-o` | Output file or directory path (use quotes for paths with spaces) | Auto-generated |
| `--quality` | `-q` | Quality preset (high, medium, low) | medium |
| `--concurrent` | `-c` | Number of concurrent conversions for batch | CPU count |
| `--overwrite` | | Overwrite existing output files | false |
| `--help` | `-h` | Show help | |

**Note**: You can provide the input path either as a positional argument (`convert-vid convert "path"`) or using the `--input` flag (`convert-vid convert --input "path"`). The flag method is recommended for paths containing spaces.

#### Quality Presets

- **high**: CRF 18, slow preset (best quality, larger files)
- **medium**: CRF 23, medium preset (balanced)
- **low**: CRF 28, fast preset (smaller files, lower quality)

### Examples

#### Convert a Single File

```bash
# Basic conversion
convert-vid convert vacation.avi -f mp4

# With custom output and high quality
convert-vid convert vacation.avi -f mp4 -o vacation_hd.mp4 -q high

# Convert to different formats
convert-vid convert video.mp4 -f webm
convert-vid convert video.avi -f mkv
convert-vid convert video.mov -f flv
```

#### Batch Convert a Directory

```bash
# Convert all videos in ./videos to mp4
convert-vid convert ./videos -f mp4

# Convert with custom output directory
convert-vid convert ./videos -f mp4 -o ./converted_videos

# Convert with 4 concurrent workers
convert-vid convert ./videos -f mp4 -c 4 -o ./output

# Convert with high quality and overwrite
convert-vid convert ./videos -f mp4 -q high --overwrite
```

## Supported Formats

### Input Formats
The tool can read most common video formats including:
- mp4, avi, mov, mkv, webm, flv
- m4v, mpg, mpeg, wmv
- And many more supported by FFmpeg

### Output Formats
- **mp4** - MPEG-4 Part 14 (H.264)
- **avi** - Audio Video Interleave
- **mov** - QuickTime File Format
- **mkv** - Matroska Video
- **webm** - WebM
- **flv** - Flash Video

## How It Works

1. **FFmpeg Wrapper**: Uses FFmpeg for the actual video conversion
2. **Concurrent Processing**: Batch conversions use Go routines for parallel processing
3. **Progress Feedback**: Spinner animations show conversion progress
4. **Smart Defaults**: Automatically generates output filenames and uses optimal settings

## Performance

- **Single File**: Conversion speed depends on video size and quality settings
- **Batch Processing**: Uses worker pool pattern with concurrent processing
- **Default Workers**: Matches your CPU core count for optimal performance
- **Custom Concurrency**: Adjust with `-c` flag based on your system resources

## Project Structure

```
convert-vid-format/
├── main.go                 # Entry point
├── cmd/
│   ├── root.go            # Root command definition
│   └── convert.go         # Convert command implementation
├── internal/
│   ├── converter/
│   │   └── converter.go   # Core conversion logic
│   ├── progress/
│   │   └── progress.go    # Progress indicators
│   └── utils/
│       └── utils.go       # Helper functions
└── pkg/
    └── types/
        └── types.go       # Type definitions
```

## Development

### Build

```bash
go build -o convert-vid
```

### Run Tests

```bash
go test ./...
```

### Run Without Building

```bash
go run main.go convert [args]
```

## Troubleshooting

### FFmpeg Not Found
```
Error: ffmpeg is not installed or not in PATH
```
**Solution**: Install FFmpeg (see Prerequisites section)

### Permission Denied
```
Error: permission denied
```
**Solution**: Check file permissions or use `sudo` if needed

### Output File Exists
```
Error: output file already exists
```
**Solution**: Use `--overwrite` flag to replace existing files

### Unsupported Format
```
Error: unsupported format
```
**Solution**: Check that the format is one of: mp4, avi, mov, mkv, webm, flv

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Credits

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Powered by [FFmpeg](https://ffmpeg.org/) for video conversion

## Version

Current version: 1.0.0

---

**Note**: This tool requires FFmpeg to be installed on your system. Make sure FFmpeg is in your PATH before using convert-vid.
