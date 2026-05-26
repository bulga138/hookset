package commands

import (
	"fmt"
	"os"

	"github.com/bulga138/hookset/internal/version"
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

See https://bulga138.github.io/hookset/ for documentation.`,
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
	v := version.Version
	if v == "dev" || v == "" {
		return "dev"
	}
	s := v
	if version.Commit != "none" && version.Commit != "" {
		s += " (" + version.Commit + ")"
	}
	if version.BuildTime != "unknown" && version.BuildTime != "" {
		s += " built " + version.BuildTime
	}
	return s
}
