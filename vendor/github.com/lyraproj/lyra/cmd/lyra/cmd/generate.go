package cmd

import (
	"os"

	"github.com/leonelquinteros/gotext"
	"github.com/lyraproj/lyra/cmd/lyra/ui"
	"github.com/lyraproj/lyra/pkg/generate"
	"github.com/spf13/cobra"
)

var targetDirectory = ``

//NewGenerateCmd generates typesets in the languge of choice
func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     gotext.Get("generate <target-language>"),
		Short:   gotext.Get("Generate all typesets in the target language"),
		Long:    gotext.Get("Generate all typesets in the target language"),
		Example: gotext.Get("\n  # Generate all typesets in typescript\n  lyra generate typescript\n"),
		Run:     runGenerateCmd,
		Args:    cobra.ExactArgs(1),
	}

	cmd.Flags().StringVarP(&homeDir, "root", "r", "", gotext.Get("path to root directory"))
	cmd.Flags().StringVarP(&targetDirectory, "target-directory", "t", "", gotext.Get("path to target directory"))

	cmd.SetHelpTemplate(ui.HelpTemplate)
	cmd.SetUsageTemplate(ui.UsageTemplate)

	return cmd
}

func runGenerateCmd(cmd *cobra.Command, args []string) {
	exitCode := generate.Generate(args[0], targetDirectory)
	if exitCode == 0 {
		ui.ShowMessage("Generation complete")
	}
	os.Exit(exitCode)
}
