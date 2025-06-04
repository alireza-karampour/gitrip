package cmd

import (
	"context"
	"github.com/alireza-karampour/gitrip/git"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	OldWd   string
	Remote  *string
	Paths   *[]string
	Tree    *string
	Dest    *string
	Verbose *bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitrip",
	Short: "A simple cli for downloading a subset of files or directories from a git repo",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	PreRun: func(cmd *cobra.Command, args []string) {
		if *Verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp:       true,
			DisableLevelTruncation: true,
			PadLevelText:           true,
			QuoteEmptyFields:       true,
		})
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		nullDev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer nullDev.Close()
		var output io.Writer = nullDev
		if *Verbose {
			output = cmd.ErrOrStderr()
		}

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
		_, _, err = clone.Exec(context.Background(), output)
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
		_, _, err = sp.Exec(context.Background(), output)
		if err != nil {
			return err
		}
		// checkout to download files
		chck := git.Git().Checkout(*Tree)
		_, _, err = chck.Exec(context.Background(), output)
		if err != nil {
			return err
		}
		// copy all to dest
		copyWg := sync.WaitGroup{}
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
				err = os.MkdirAll(dst, 0777)
				if err != nil {
					return err
				}
			} else {
				dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
				if err != nil {
					return err
				}
				srcFile, err := os.OpenFile(path, os.O_RDONLY, 0666)
				if err != nil {
					return err
				}
				copyWg.Add(1)
				go func() {
					defer logrus.Debugf("copied %s", path)
					defer copyWg.Done()
					defer dstFile.Close()
					defer srcFile.Close()

					_, err := io.Copy(dstFile, srcFile)
					if err != nil {
						logrus.WithError(err).Errorf("failed to copy file '%s'", path)
						return
					}
				}()
			}
			return nil
		})
		copyWg.Wait()
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

	// optional
	Dest = rootCmd.Flags().StringP("dest", "d", ".", "destination to download files/directories to")
	Verbose = rootCmd.Flags().BoolP("verbose", "v", false, "whether debug logs should be printed")
}
