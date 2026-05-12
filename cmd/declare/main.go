// Command declare is the CLI entry point for the `.dx` toolchain.
//
// It contains no LLM logic (per ARCHITECTURE.md §4): every subcommand is a
// deterministic operation over the `.dx` AST.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dewitt/declare/pkg/diff"
	"github.com/dewitt/declare/pkg/export"
	"github.com/dewitt/declare/pkg/lint"
)

// version is overwritten at build time via -ldflags.
var version = "0.1.0-dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// Cobra has already printed the error; exit non-zero so shells
		// and CI runners pick it up.
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "declare",
		Short:         "Toolchain for the .dx declarative specification language",
		Long:          "declare is a deterministic toolchain for authoring, validating, and exporting .dx files.",
		Version:       version,
		SilenceUsage:  true, // do not dump usage on every command-level error
		SilenceErrors: false,
	}
	root.AddCommand(newLintCmd(), newFmtCmd(), newDiffCmd(), newExportCmd())
	return root
}

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <old> <new>",
		Short: "Emit a semantic ledger of operations between two .dx sources",
		Long: "Parses both sources into the AST and reports a stable, " +
			"machine-parseable list of operations that describe how the " +
			"declaration's intent and constraints changed (per " +
			"ARCHITECTURE.md §4 and AGENTS.md §5). Use this -- not text " +
			"diff -- to communicate spec changes to a human or another " +
			"agent.\n\n" +
			"Each source may be either a filesystem path or a git " +
			"revision spec of the form <rev>:<path>, mirroring " +
			"`git show` syntax. Examples:\n\n" +
			"  declare diff old.dx new.dx\n" +
			"  declare diff HEAD~1:system.dx system.dx\n" +
			"  declare diff main:examples/hello.dx HEAD:examples/hello.dx\n",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldRes, err := lint.LintSource(args[0])
			if err != nil {
				return err
			}
			newRes, err := lint.LintSource(args[1])
			if err != nil {
				return err
			}
			// We tolerate lint warnings on either side here: an
			// architect may legitimately want to diff a known-broken
			// spec against a fix. We refuse only when the source
			// failed to decode at all (Declaration is nil).
			if oldRes.Declaration == nil {
				for _, i := range oldRes.Issues {
					fmt.Fprintln(cmd.ErrOrStderr(), i)
				}
				return fmt.Errorf("diff aborted: %s did not decode", args[0])
			}
			if newRes.Declaration == nil {
				for _, i := range newRes.Issues {
					fmt.Fprintln(cmd.ErrOrStderr(), i)
				}
				return fmt.Errorf("diff aborted: %s did not decode", args[1])
			}
			changes := diff.Diff(oldRes.Declaration, newRes.Declaration)
			if len(changes) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no semantic changes)")
				return nil
			}
			for _, c := range changes {
				fmt.Fprintln(cmd.OutOrStdout(), c)
			}
			return nil
		},
	}
}

func newLintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lint [source...]",
		Short: "Validate one or more .dx sources against the SPEC",
		Long: "Validates each source against SPEC §2 (physical rules) " +
			"and §3 (required keys). Each source may be a filesystem " +
			"path or a git revision spec of the form <rev>:<path> " +
			"(see `declare diff --help` for examples).",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var failed bool
			for _, source := range args {
				res, err := lint.LintSource(source)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", source, err)
					failed = true
					continue
				}
				for _, issue := range res.Issues {
					fmt.Fprintln(cmd.ErrOrStderr(), issue)
				}
				if !res.OK() {
					failed = true
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: ok\n", source)
			}
			if failed {
				return fmt.Errorf("lint failed")
			}
			return nil
		},
	}
}

func newFmtCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fmt [file...]",
		Short: "Canonicalize the formatting of .dx files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "fmt: not yet implemented")
			return nil
		},
	}
}

func newExportCmd() *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "export <source>",
		Short: "Emit the AST in an agent-optimized format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := lint.LintSource(args[0])
			if err != nil {
				return err
			}
			if !res.OK() {
				for _, issue := range res.Issues {
					fmt.Fprintln(cmd.ErrOrStderr(), issue)
				}
				return fmt.Errorf("export aborted: %s has lint errors", args[0])
			}
			if err := export.Write(cmd.OutOrStdout(), res.Declaration, export.Format(format)); err != nil {
				return err
			}
			return nil
		},
	}
	c.Flags().StringVarP(&format, "format", "f", string(export.FormatJSON), "output format (json)")
	return c
}
