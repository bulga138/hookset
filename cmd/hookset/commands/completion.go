package commands

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate the autocompletion script for the specified shell",
	Long: `Generate the autocompletion script for the specified shell.

To load completions:

Bash:

  $ source <(hookset completion bash)

  # To load completions for each session, execute once:
  $ hookset completion bash > /etc/bash_completion.d/hookset

Zsh:

  # If shell completion is not enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ hookset completion zsh > "${fpath[1]}/_hookset"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ hookset completion fish | source

  # To load completions for each session, execute once:
  $ hookset completion fish > ~/.config/fish/completions/hookset.fish

PowerShell:

  $ hookset completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  $ hookset completion powershell > hookset.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletion(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
