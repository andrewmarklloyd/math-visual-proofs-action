package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "math-visual-proofs-action",
	Short: "todo",
	Long:  `todo`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
	// TODO: exit 1 if no sub command ran?
}

func init() {
	rootCmd.PersistentFlags().String("fileNames", "", "")
	rootCmd.PersistentFlags().String("repoURL", "", "")
}
