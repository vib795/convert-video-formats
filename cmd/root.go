package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "convert-vid",
	Short: "A CLI tool for converting video formats",
	Long: `convert-vid is a command-line tool for converting video files from one format to another.
It supports batch conversion of entire folders and uses FFmpeg for the actual conversion.

Supported formats: mp4, avi, mov, mkv, webm, flv

Examples:
  # Convert a single file
  convert-vid convert input.avi --format mp4 --output output.mp4

  # Convert all videos in a folder
  convert-vid convert ./videos --format mp4 --output ./converted

  # Convert with specific quality
  convert-vid convert input.mov --format mp4 --quality high`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("convert-vid v%s\n", Version)
	},
}