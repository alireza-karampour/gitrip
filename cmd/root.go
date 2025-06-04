package cmd

import (
	"alireza-karampour/gitrip/git"
	"context"
	"os"

	"github.com/spf13/cobra"
)

var (
	Remote *string
	Paths  *[]string
	Tree   *string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitrip",
	Short: "A simple cli for downloading a subset of files or directories from a git repo",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.MkdirTemp("/tmp", "gr.*")
		if err != nil {
			return err
		}

		// clone
		clone := git.Git().Clone(*Remote, dir)
		_, _, err = clone.Exec(context.Background(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		// go inside
		err = os.Chdir(dir)
		if err != nil {
			return err
		}
		// enable sparse-checkout
		sp := git.Git().Sp(*Paths...)
		_, _, err = sp.Exec(context.Background(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		// checkout to download files
		chck := git.Git().Checkout(*Tree)
		_, _, err = chck.Exec(context.Background(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	Remote = rootCmd.Flags().StringP("remote", "r", "", "address of the git repo to download from (required)")
	rootCmd.MarkFlagRequired("remote")
	Paths = rootCmd.Flags().StringSliceP("paths", "p", nil, "files or directories to download (required)")
	rootCmd.MarkFlagRequired("paths")
	Tree = rootCmd.Flags().StringP("tree", "t", "", "tree to download files/directories from (required)")
	rootCmd.MarkFlagRequired("tree")
}
