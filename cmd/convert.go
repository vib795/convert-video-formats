package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/vib795/convert-video-formats/internal/converter"
	"github.com/vib795/convert-video-formats/internal/progress"
	"github.com/vib795/convert-video-formats/internal/utils"
	"github.com/vib795/convert-video-formats/pkg/types"
)

var (
	inputPath  string
	format     string
	outputPath string
	quality    string
	overwrite  bool
	concurrent int
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert [input]",
	Short: "Convert video file(s) to a different format",
	Long: `Convert a single video file or all video files in a directory to a different format.

The tool uses FFmpeg for conversion, so FFmpeg must be installed on your system.

Examples:
  # Convert a single file (use quotes for paths with spaces)
  convert-vid convert input.avi --format mp4 --output output.mp4
  convert-vid convert "My Video.mov" -f mp4 -o "Output.mp4"

  # Using the --input flag (recommended for paths with spaces)
  convert-vid convert --input "~/Downloads/My Videos/video.webm" -f mp4 -o "output.mp4"
  convert-vid convert -i "path with spaces.avi" -f mp4

  # Convert all videos in a folder
  convert-vid convert ./videos --format mp4 --output ./converted
  convert-vid convert --input "./My Videos" -f mp4 -o "./Converted"

  # Convert with high quality
  convert-vid convert input.mov --format mp4 --quality high

  # Convert with custom concurrency
  convert-vid convert ./videos -f mp4 -c 4`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConvert,
}

func init() {
	rootCmd.AddCommand(convertCmd)

	convertCmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input video file or directory path (use quotes for paths with spaces)")
	convertCmd.Flags().StringVarP(&format, "format", "f", "mp4", "Target video format (mp4, avi, mov, mkv, webm, flv)")
	convertCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file or directory path (use quotes for paths with spaces)")
	convertCmd.Flags().StringVarP(&quality, "quality", "q", "medium", "Quality preset (high, medium, low)")
	convertCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing output files")
	convertCmd.Flags().IntVarP(&concurrent, "concurrent", "c", runtime.NumCPU(), "Number of concurrent conversions for batch processing")

	convertCmd.MarkFlagRequired("format")
}

func runConvert(cmd *cobra.Command, args []string) error {
	// Register signal handler for graceful terminal cleanup
	progress.RegisterCleanupHandler()

	// Determine input path from either flag or positional argument
	var input string
	if inputPath != "" {
		// Use flag value if provided
		input = inputPath
	} else if len(args) > 0 {
		// Use positional argument if provided
		input = args[0]
	} else {
		return fmt.Errorf("input path is required. Use --input flag or provide as positional argument.\n\nFor paths with spaces, use quotes:\n  convert-vid convert --input \"My Videos/video.mp4\" -f mp4\n  convert-vid convert \"My Videos/video.mp4\" -f mp4")
	}

	// Expand paths (handles ~ for home directory)
	expandedInput, err := utils.ExpandPath(input)
	if err != nil {
		return fmt.Errorf("failed to expand input path: %w", err)
	}
	input = expandedInput

	if outputPath != "" {
		expandedOutput, err := utils.ExpandPath(outputPath)
		if err != nil {
			return fmt.Errorf("failed to expand output path: %w", err)
		}
		outputPath = expandedOutput
	}

	// Validate format
	if !utils.IsValidFormat(format) {
		return fmt.Errorf("unsupported format: %s. Supported formats: mp4, avi, mov, mkv, webm, flv", format)
	}

	// Validate quality
	if !utils.IsValidQuality(quality) {
		return fmt.Errorf("invalid quality: %s. Valid options: high, medium, low", quality)
	}

	// Check if input exists
	if _, err := os.Stat(input); err != nil {
		return fmt.Errorf("input path does not exist: %s\n\nTip: If your path contains spaces, wrap it in quotes:\n  convert-vid convert --input \"path with spaces/video.mp4\" -f mp4", input)
	}

	// Create conversion options
	opts := &types.ConversionOptions{
		InputPath:  input,
		OutputPath: outputPath,
		Format:     format,
		Quality:    quality,
		Overwrite:  overwrite,
		Concurrent: concurrent,
	}

	// Create converter and run
	conv := converter.NewConverter(opts)
	if err := conv.Convert(); err != nil {
		return err
	}

	return nil
}
