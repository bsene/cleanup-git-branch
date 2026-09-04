// cleanup-git-branch deletes stale local git branches safely.
//
// The tool defaults to dry-run mode: it prints the branches it would delete but
// does not touch them. Pass --yes to perform the actual cleanup.
package main

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/bsene/cleanup-git-branch/internal/cleaner"
	"github.com/bsene/cleanup-git-branch/internal/config"
	"github.com/bsene/cleanup-git-branch/internal/git"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := config.Parse(args)
	if err != nil {
		return err
	}

	client := git.NewClient()
	c := cleaner.NewCleaner(client)
	results, err := c.Run(cfg)
	if err != nil {
		// Partial failures still carry per-branch results; show them
		// before surfacing the aggregated error.
		if len(results) > 0 {
			printResults(results, cfg.Yes, cfg.Verbose)
		}
		return err
	}

	if len(results) == 0 {
		fmt.Println("No stale branches found.")
		return nil
	}

	printResults(results, cfg.Yes, cfg.Verbose)
	if !cfg.Yes {
		fmt.Fprintf(os.Stderr, "\nThis was a dry run. Pass --yes to delete the %d listed branch(es).\n", len(results))
	}
	return nil
}

func printResults(results []cleaner.Result, executed, verbose bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "BRANCH\tSTATUS\tREASON")
	for _, r := range results {
		status := "would delete"
		if executed {
			if r.Deleted {
				status = "deleted"
			} else if r.Error != nil {
				status = "failed"
			} else {
				status = "skipped"
			}
		}
		line := fmt.Sprintf("%s\t%s\t%s", r.Branch.Name, status, r.Reason)
		if verbose {
			line += fmt.Sprintf("\t[%s]", r.Branch.Ref)
		}
		if r.Error != nil {
			line += fmt.Sprintf("\t(%v)", r.Error)
		}
		fmt.Fprintln(w, line)
	}
}
