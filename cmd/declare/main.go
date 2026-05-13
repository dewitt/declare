// Command declare is the CLI entry point for the `.dx` toolchain.
//
// It contains no LLM logic (per SPEC.md §1: the binary is the referee, not
// a player): every subcommand is a deterministic operation over the `.dx`
// AST.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dewitt/declare/pkg/canonical"
	"github.com/dewitt/declare/pkg/contracts"
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
	root.AddCommand(
		newLintCmd(),
		newFmtCmd(),
		newDiffCmd(),
		newExportCmd(),
		newContractsCmd(),
	)
	return root
}

func newContractsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "contracts",
		Short: "Operations over the `contracts:` block of a .dx file",
		Long: "Subcommands that read the `contracts:` block. Today only " +
			"`list` exists; once `declare verify` ships in v0.2 it will " +
			"land here as `declare contracts run`.",
	}
	c.AddCommand(newContractsListCmd())
	return c
}

func newContractsListCmd() *cobra.Command {
	var (
		format  string
		verbose bool
	)
	c := &cobra.Command{
		Use:   "list <source>",
		Short: "List the contract identifiers in a .dx file",
		Long: "Reads the `contracts:` block of <source> and emits one " +
			"contract identifier per line in alphabetical order, " +
			"suitable for piping into a runner. With --verbose, each " +
			"identifier is followed by a one-line preview of given/" +
			"when/then. With --format=json, emits a structured object " +
			"with the full bodies; --verbose has no effect on JSON " +
			"output (which is always full-fidelity).\n\n" +
			"<source> may be a filesystem path or a git revision spec " +
			"(see `declare diff --help`). A spec with no contracts " +
			"prints nothing in text mode and `{\"contracts\":[]}` in " +
			"JSON mode; both exit 0.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := lint.LintSource(args[0])
			if err != nil {
				return err
			}
			if !res.OK() {
				for _, issue := range res.Issues {
					fmt.Fprintln(cmd.ErrOrStderr(), issue)
				}
				return fmt.Errorf("contracts list aborted: %s has lint errors", args[0])
			}
			entries := contracts.List(res.Declaration)
			return contracts.WriteList(cmd.OutOrStdout(), entries, contracts.WriteOptions{
				Format:  contracts.Format(format),
				Verbose: verbose,
			})
		},
	}
	c.Flags().StringVarP(&format, "format", "f", string(contracts.FormatText),
		"output format (text or json)")
	c.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"in text mode, show a one-line preview of given/when/then under each contract")
	return c
}

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <old> <new>",
		Short: "Emit a semantic ledger of operations between two .dx sources",
		Long: "Parses both sources into the AST and reports a stable, " +
			"machine-parseable list of operations that describe how the " +
			"declaration's intent and constraints changed (per " +
			"SPEC.md §6 and AGENTS.md §5). Use this -- not text " +
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
	var write bool
	c := &cobra.Command{
		Use:   "fmt <file> [file ...]",
		Short: "Canonicalize the formatting of .dx files",
		Long: "Reformats one or more .dx files into the canonical " +
			"form mandated by SPEC §2: top-level keys in canonical " +
			"order; map entries inside invariants/assumptions/" +
			"contracts/unconstrained sorted alphabetically; literal " +
			"block scalars (`|`) for any multi-line string; trailing " +
			"whitespace stripped; exactly one trailing newline.\n\n" +
			"By default, prints the formatted output to stdout " +
			"without modifying the input -- safe for piping into " +
			"`diff` or another tool. Pass --write (-w) to overwrite " +
			"the input file in place. Idempotent: " +
			"`fmt(fmt(x)) == fmt(x)` byte-for-byte.\n\n" +
			"Top-level head comments are preserved; comments inside " +
			"invariants/assumptions/contracts/unconstrained entries " +
			"are NOT preserved across formatting (a known limitation).",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// fmt deliberately accepts only filesystem paths, not
			// git-revision specs: the --write semantics on a git
			// revision are nonsensical, and the stdout path would
			// just be `git show <rev>:<path> | declare fmt -` if
			// we grew stdin support, which we haven't.
			var failed bool
			for _, path := range args {
				out, err := formatFile(path)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err)
					failed = true
					continue
				}
				if write {
					if err := os.WriteFile(path, out, 0o644); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err)
						failed = true
						continue
					}
				} else {
					_, _ = cmd.OutOrStdout().Write(out)
				}
			}
			if failed {
				return fmt.Errorf("fmt failed")
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&write, "write", "w", false,
		"overwrite each input file in place instead of writing to stdout")
	return c
}

// formatFile reads the named file, lints it (refusing to format an
// invalid spec -- formatting a broken file would silently change its
// shape and likely make the diagnosis harder), and returns the
// canonicalized bytes.
func formatFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	res := lint.Lint(path, data)
	if !res.OK() {
		// Surface the issues to stderr-of-the-process here would
		// duplicate them across multi-file calls; let the caller
		// report a single combined error.
		var buf bytes.Buffer
		for _, i := range res.Issues {
			fmt.Fprintln(&buf, i)
		}
		return nil, fmt.Errorf("refusing to format file with lint issues:\n%s", buf.String())
	}
	return canonical.Marshal(res.Declaration, canonical.Options{
		StripComments: false,
		SourceNode:    res.Declaration.Node,
	})
}

func newExportCmd() *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "export <source>",
		Short: "Emit the AST in an agent-optimized format",
		Long: "Emits a canonical projection of the .dx file suitable " +
			"for ingestion by another agent. Comments are stripped; " +
			"top-level keys appear in SPEC §2 canonical order; map " +
			"entries are sorted alphabetically. The output is " +
			"byte-stable for the same AST -- two agents can hash the " +
			"export and compare.\n\n" +
			"Source may be a filesystem path or a git revision spec " +
			"(see `declare diff --help`).",
		Args: cobra.ExactArgs(1),
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
	c.Flags().StringVarP(&format, "format", "f", string(export.FormatYAML),
		"output format (yaml or json)")
	return c
}
