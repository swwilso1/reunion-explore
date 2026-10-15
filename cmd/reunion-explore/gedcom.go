package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kedoco/reunion-explore/gedcom"
)

var gedcomCmd = &cobra.Command{
	Use:     "gedcom <bundle>",
	Short:   "Export as GEDCOM 5.5.1 (-o for a file, default stdout)",
	Args:    cobra.ExactArgs(1),
	PreRunE: loadBundleFromArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath, _ := cmd.Flags().GetString("output")
		noPictures, _ := cmd.Flags().GetBool("no-pictures")

		var w io.Writer = os.Stdout
		if outPath != "" && outPath != "-" {
			f, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}

		if err := gedcom.Write(w, ff, gedcom.Options{ProgramVersion: version, NoPictures: noPictures}); err != nil {
			return fmt.Errorf("writing GEDCOM: %w", err)
		}
		if outPath != "" && outPath != "-" {
			fmt.Fprintf(os.Stderr, "Wrote %d persons to %s\n", len(ff.Persons), outPath)
		}
		return nil
	},
}

func init() {
	gedcomCmd.Flags().StringP("output", "o", "", "Output file (default stdout)")
	gedcomCmd.Flags().Bool("no-pictures", false, "Omit picture links")
	rootCmd.AddCommand(gedcomCmd)
}
