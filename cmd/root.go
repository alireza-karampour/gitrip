package cmd

import (
	"alireza-karampour/gitrip/git"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	OldWd  string
	Remote *string
	Paths  *[]string
	Tree   *string
	Dest   *string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitrip",
	Short: "A simple cli for downloading a subset of files or directories from a git repo",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		if Dest != nil {
			dst, err := filepath.Abs(*Dest)
			if err != nil {
				return err
			}
			*Dest = dst
		}
		dir, err := os.MkdirTemp("/tmp", "gr.*")
		if err != nil {
			return err
		}
		defer func() {
			err := os.RemoveAll(dir)
			if err != nil {
				logrus.WithError(err).Error("failed to remove tmp dir")
			}
		}()
		// clone
		clone := git.Git().Clone(*Remote, dir)
		_, _, err = clone.Exec(context.Background(), cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		// go inside
		OldWd, err = os.Getwd()
		if err != nil {
			return err
		}
		logrus.Debugf("cd %s --> %s", OldWd, dir)
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
		// copy all to dest
		err = fs.WalkDir(os.DirFS("."), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			if strings.HasSuffix(path, ".git") {
				return fs.SkipDir
			}
			logrus.Debug(path)
			dst := filepath.Join(*Dest, path)
			logrus.Debugf("dest: %s", dst)
			if d.IsDir() {

			}
			return nil
		})
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
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		QuoteEmptyFields:       true,
	})
	Remote = rootCmd.Flags().StringP("remote", "r", "", "address of the git repo to download from (required)")
	rootCmd.MarkFlagRequired("remote")
	Paths = rootCmd.Flags().StringSliceP("paths", "p", nil, "files or directories to download (required)")
	rootCmd.MarkFlagRequired("paths")
	Tree = rootCmd.Flags().StringP("tree", "t", "", "tree to download files/directories from (required)")
	rootCmd.MarkFlagRequired("tree")

	// optional
	Dest = rootCmd.Flags().StringP("dest", "d", ".", "destination to download files/directories to")
}
