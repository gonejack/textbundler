package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gonejack/textbundler"
	"github.com/gonejack/textbundler/util"
	"github.com/spf13/cobra"
)

var (
	processAttachments bool
	useGitDates        bool
	toAppend           string
	concurrent         int
	verbose            bool
)

// cmd handles the base case for textbundler: processing Markdown files.
var cmd = &cobra.Command{
	Use:   "textbundler [file] [file2] [file3]...",
	Short: "Convert markdown files into textbundles",
	Run: func(md *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Please pass at least one argument.")
			os.Exit(1)
		}

		for _, mdPath := range args {
			if err := process(mdPath); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().BoolVarP(
		&processAttachments,
		"process-attachments",
		"p",
		false,
		"Replace links to local files with Bear-compatible tags to ease processing",
	)
	cmd.PersistentFlags().BoolVarP(
		&useGitDates,
		"git-dates",
		"g",
		false,
		"Instead of using OS creation / modification dates of Markdown file, use the dates from git commit history (must be in a git repo & have git CLI)",
	)
	cmd.PersistentFlags().StringVarP(
		&toAppend,
		"append",
		"a",
		"",
		"Text to append to end of Markdown file. Use %f to template the original filename.",
	)
	cmd.PersistentFlags().IntVarP(
		&concurrent,
		"concurrent",
		"c",
		5,
		"Max concurrent image downloads",
	)
	cmd.PersistentFlags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"Verbose",
	)
}

func process(mdPath string) error {
	contents, err := ioutil.ReadFile(mdPath)
	if err != nil {
		return err
	}

	absMdPath, err := filepath.Abs(mdPath)
	if err != nil {
		return err
	}

	var creation, change time.Time

	if useGitDates {
		creation, err = util.GetGitBirthTime(absMdPath)
		if err != nil {
			return err
		}

		change, err = util.GetGitModTime(absMdPath)
		if err != nil {
			return err
		}
	} else {
		creation, err = util.GetBirthTime(absMdPath)
		if err != nil {
			return err
		}

		change, err = util.GetModTime(absMdPath)
		if err != nil {
			return err
		}
	}

	if verbose {
		fmt.Printf("Process %s\n", mdPath)
	}
	err = Textbundler.GenerateBundle(
		Textbundler.Config{
			MdContents:         contents,
			AbsMdPath:          absMdPath,
			Creation:           creation,
			Modification:       change,
			Dest:               filepath.Dir(absMdPath) + "/",
			ProcessAttachments: processAttachments,
			ToAppend:           strings.Replace(toAppend, `\n`, "\n", -1),
			Verbose:            verbose,
			Concurrent:         concurrent,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// Execute begins the CLI processing flow
func Execute() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
