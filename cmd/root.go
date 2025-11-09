package cmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alireza-karampour/gitrip/git"

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
	Temp    *bool
)

// parseRepoInfo extracts owner and project name from a git remote URL
func parseRepoInfo(remote string) (owner, project string, err error) {
	// Remove .git suffix if present
	remote = strings.TrimSuffix(remote, ".git")

	// Handle git@host:owner/repo format (e.g., git@github.com:owner/repo)
	if strings.Contains(remote, "@") && strings.Contains(remote, ":") {
		parts := strings.Split(remote, ":")
		if len(parts) >= 2 {
			pathParts := strings.Split(parts[len(parts)-1], "/")
			if len(pathParts) >= 2 {
				owner = pathParts[len(pathParts)-2]
				project = pathParts[len(pathParts)-1]
				// Remove any query params or fragments
				project = strings.Split(project, "?")[0]
				project = strings.Split(project, "#")[0]
				return owner, project, nil
			}
		}
	}

	// Handle https://host/owner/repo format
	if strings.HasPrefix(remote, "http://") || strings.HasPrefix(remote, "https://") {
		parts := strings.Split(remote, "/")
		if len(parts) >= 3 {
			owner = parts[len(parts)-2]
			project = parts[len(parts)-1]
			// Remove any query params or fragments
			project = strings.Split(project, "?")[0]
			project = strings.Split(project, "#")[0]
			return owner, project, nil
		}
	}

	// Fallback: try to extract from common patterns
	parts := strings.Split(remote, "/")
	if len(parts) >= 2 {
		owner = parts[len(parts)-2]
		project = parts[len(parts)-1]
		// Remove any query params or fragments
		project = strings.Split(project, "?")[0]
		project = strings.Split(project, "#")[0]
		return owner, project, nil
	}

	return "", "", fmt.Errorf("unable to parse repo owner and project from remote: %s", remote)
}

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
		nullDev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0o666)
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

		var dir string
		if *Temp {
			// Use temporary directory
			dir, err = os.MkdirTemp("/tmp", "gr.*")
			if err != nil {
				return err
			}
			defer func() {
				err := os.RemoveAll(dir)
				if err != nil {
					logrus.WithError(err).Error("failed to remove tmp dir")
				}
			}()
		} else {
			// Use persistent directory in ~/.gitrip/<REPO_OWNER>/<PROJECT>
			owner, project, err := parseRepoInfo(*Remote)
			if err != nil {
				return err
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dir = filepath.Join(home, ".gitrip", owner, project)
			// Create directory if it doesn't exist
			err = os.MkdirAll(dir, 0o755)
			if err != nil {
				return err
			}
		}

		// clone (only if .git doesn't exist, meaning it's a fresh clone)
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			clone := git.Git().Clone(*Remote, dir)
			_, _, err = clone.Exec(context.Background(), output)
			if err != nil {
				return err
			}
		} else {
			logrus.Debugf("Repository already exists at %s, skipping clone", dir)
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
		defer func() {
			err := os.Chdir(OldWd)
			if err != nil {
				logrus.WithError(err).Error("failed to restore working directory")
			}
		}()

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
				err = os.MkdirAll(dst, 0o777)
				if err != nil {
					return err
				}
			} else {
				dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o666)
				if err != nil {
					return err
				}
				srcFile, err := os.OpenFile(path, os.O_RDONLY, 0o666)
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
	Paths = rootCmd.Flags().StringSliceP("pattern", "p", nil, "files or directories to download (required)")
	rootCmd.MarkFlagRequired("paths")
	Tree = rootCmd.Flags().StringP("branch", "b", "", "branch or tree to download files/directories from (required)")
	rootCmd.MarkFlagRequired("branch")
	Temp = rootCmd.Flags().BoolP("temp", "t", false, "if true keeps the repo history to avoid redownload. (use if history size is high or you download from the repo often)")

	// optional
	Dest = rootCmd.Flags().StringP("dest", "d", ".", "destination to download files/directories to")
	Verbose = rootCmd.Flags().BoolP("verbose", "v", false, "whether debug logs should be printed")
}
