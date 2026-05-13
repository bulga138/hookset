package commands

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "hookset",
	Short: "Git 2.54 config-native hook manager",
	Long: `hookset manages Git hooks entirely through Git's native [hook] configuration
sections (Git 2.54+). No Husky. No lint-staged. No per-repo scripts.

Hooks are defined in .hookset.toml and installed into git config with:

  hookset init

See https://hookset.dev for documentation.`,
	SilenceUsage: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print hookset version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hookset", buildVersion())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Print step-by-step output")
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
