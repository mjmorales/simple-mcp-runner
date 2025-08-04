package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information about the simple-mcp-runner binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("simple-mcp-runner %s\n", Version)
		fmt.Printf("  Commit: %s\n", Commit)
		fmt.Printf("  Built: %s\n", BuildTime)
		fmt.Printf("  Go version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}