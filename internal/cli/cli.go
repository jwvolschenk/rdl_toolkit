package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/rdl-toolkit/internal/rdl"
)

func Execute() error {
	var dryRun bool
	root := &cobra.Command{
		Use:   "rdl-tool",
		Short: "CLI tool for manipulating SSRS RDL files",
	}
	// Single persistent flag — inherited by every mutation subcommand.
	// Inspect commands ignore it (they're already read-only).
	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false,
		"Preview mutation results without writing the file (summary is prefixed with [DRY RUN])")

	// ── inspect commands (read-only) ───────────────────────────────────
	root.AddCommand(inspectCmd("inspect", "Show a top-level summary of the report",
		func(doc *rdl.Document) any { return doc.Overview() }))
	root.AddCommand(inspectCmd("list-datasources", "List all DataSources in the report",
		func(doc *rdl.Document) any { return doc.ListDataSources() }))
	root.AddCommand(inspectCmd("list-datasets", "List all DataSets with their fields and filters",
		func(doc *rdl.Document) any { return doc.ListDataSets() }))
	root.AddCommand(inspectCmd("list-parameters", "List all ReportParameters",
		func(doc *rdl.Document) any { return doc.ListParameters() }))
	root.AddCommand(inspectCmd("list-tablixes", "List all Tablixes with their cells and values",
		func(doc *rdl.Document) any { return doc.ListTablixes() }))
	root.AddCommand(inspectCmd("get-metadata", "Show report metadata (page size, margins, language, etc.)",
		func(doc *rdl.Document) any { return doc.GetMetadata() }))

	// clone
	var cloneSource, cloneTarget string
	cloneCmd := &cobra.Command{
		Use:   "clone",
		Short: "Copy RDL file with new ReportID",
		RunE: func(cmd *cobra.Command, args []string) error {
			newID, err := rdl.Clone(cloneSource, cloneTarget, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			prefix := ""
			if dryRun {
				prefix = "[DRY RUN] "
			}
			fmt.Printf("%sCloned %s -> %s\nNew ReportID: %s\n", prefix, cloneSource, cloneTarget, newID)
			return nil
		},
	}
	cloneCmd.Flags().StringVar(&cloneSource, "source", "", "Source RDL file")
	cloneCmd.Flags().StringVar(&cloneTarget, "target", "", "Target RDL file")
	cloneCmd.MarkFlagRequired("source")
	cloneCmd.MarkFlagRequired("target")
	root.AddCommand(cloneCmd)

	// update-metadata
	var metaDesc, metaTitle, metaTitleTextbox, metaOrientation string
	metaCmd := &cobra.Command{
		Use:   "update-metadata FILE",
		Short: "Update report metadata (description, title, orientation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec := rdl.MetadataUpdate{
				Description:  metaDesc,
				Title:        metaTitle,
				TitleTextbox: metaTitleTextbox,
				Orientation:  metaOrientation,
			}
			count, err := rdl.UpdateMetadata(args[0], spec, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Printf("Updated %d metadata field(s) in %s\n", count, args[0])
			return nil
		},
	}
	metaCmd.Flags().StringVar(&metaDesc, "description", "", "Description text (pipe-delimited conventions preserved)")
	metaCmd.Flags().StringVar(&metaTitle, "title", "", "New title text (requires --title-textbox)")
	metaCmd.Flags().StringVar(&metaTitleTextbox, "title-textbox", "", "Name attribute of the Textbox to update with --title")
	metaCmd.Flags().StringVar(&metaOrientation, "orientation", "", "Portrait or Landscape")
	root.AddCommand(metaCmd)

	// swap-macros
	swapMacrosCmd := &cobra.Command{
		Use:   "swap-macros FILE OLD:NEW [OLD2:NEW2 ...]",
		Short: "Replace strings in ConnectString elements",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs, err := parsePairs(args[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			count, err := rdl.SwapMacros(args[0], pairs, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Printf("Replaced %d occurrence(s) in ConnectString elements of %s\n", count, args[0])
			return nil
		},
	}
	root.AddCommand(swapMacrosCmd)

	// swap-fields
	swapFieldsCmd := &cobra.Command{
		Use:   "swap-fields FILE OLD:NEW [OLD2:NEW2 ...]",
		Short: "Replace Fields!X.Value references in Value elements",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pairs, err := parsePairs(args[1:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			count, err := rdl.SwapFields(args[0], pairs, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Printf("Replaced %d occurrence(s) in Value elements of %s\n", count, args[0])
			return nil
		},
	}
	root.AddCommand(swapFieldsCmd)

	// manage-datasources
	var dsAddJSON, dsRemove, dsRename, dsSetConn []string
	dsCmd := &cobra.Command{
		Use:   "manage-datasources FILE",
		Short: "Add, remove, rename DataSources, or set their ConnectStrings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseJSONList[rdl.DataSourceAdd](dsAddJSON)
			if err != nil {
				return err
			}
			renames, err := parsePairs(dsRename)
			if err != nil {
				return err
			}
			setConns, err := parseJSONList[rdl.ConnectStringUpdate](dsSetConn)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageDataSources(args[0], rdl.DataSourceOps{
				Add:              adds,
				Remove:           dsRemove,
				Rename:           renames,
				SetConnectString: setConns,
			}, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	dsCmd.Flags().StringArrayVar(&dsAddJSON, "add", nil, `JSON object: '{"name":"X","provider":"SQL","connectString":"Data Source=...","securityType":"Integrated"}'`)
	dsCmd.Flags().StringArrayVar(&dsRemove, "remove", nil, "DataSource name to remove")
	dsCmd.Flags().StringArrayVar(&dsRename, "rename", nil, "OLD:NEW")
	dsCmd.Flags().StringArrayVar(&dsSetConn, "set-connectstring", nil, `JSON object: '{"name":"X","connectString":"..."}'`)
	root.AddCommand(dsCmd)

	// manage-datasets
	var dtAddJSON, dtRemove, dtRename, dtAddFieldJSON, dtClearFields, dtClearFilters, dtSetCmdJSON []string
	dtCmd := &cobra.Command{
		Use:   "manage-datasets FILE",
		Short: "Add, remove, rename DataSets, edit fields, or update CommandText",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseJSONList[rdl.DataSetAdd](dtAddJSON)
			if err != nil {
				return err
			}
			renames, err := parsePairs(dtRename)
			if err != nil {
				return err
			}
			addFields, err := parseJSONList[rdl.FieldAdd](dtAddFieldJSON)
			if err != nil {
				return err
			}
			setCmds, err := parseJSONList[rdl.CmdTextUpdate](dtSetCmdJSON)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageDataSets(args[0], rdl.DataSetOps{
				Add:            adds,
				Remove:         dtRemove,
				Rename:         renames,
				ClearFields:    dtClearFields,
				ClearFilters:   dtClearFilters,
				SetCommandText: setCmds,
				AddField:       addFields,
			}, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	dtCmd.Flags().StringArrayVar(&dtAddJSON, "add", nil, `JSON object: '{"name":"X","datasource":"DS","cmdText":"SELECT * FROM t WHERE x:y","fields":["a","b"]}'`)
	dtCmd.Flags().StringArrayVar(&dtRemove, "remove", nil, "DataSet name to remove")
	dtCmd.Flags().StringArrayVar(&dtRename, "rename", nil, "OLD:NEW")
	dtCmd.Flags().StringArrayVar(&dtAddFieldJSON, "add-field", nil, `JSON object: '{"dataset":"X","field":"F"}'`)
	dtCmd.Flags().StringArrayVar(&dtClearFields, "clear-fields", nil, "DataSet name whose fields should be emptied")
	dtCmd.Flags().StringArrayVar(&dtClearFilters, "clear-filters", nil, "DataSet name whose filters should be removed")
	dtCmd.Flags().StringArrayVar(&dtSetCmdJSON, "set-command-text", nil, `JSON object: '{"dataset":"X","cmdText":"SELECT ..."}'`)
	root.AddCommand(dtCmd)

	// manage-parameters
	var paramAddJSON, paramRemove []string
	paramCmd := &cobra.Command{
		Use:   "manage-parameters FILE",
		Short: "Add or remove ReportParameters",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseJSONList[rdl.ParameterAdd](paramAddJSON)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageParameters(args[0], rdl.ParameterOps{
				Add:    adds,
				Remove: paramRemove,
			}, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	paramCmd.Flags().StringArrayVar(&paramAddJSON, "add", nil, `JSON object: '{"name":"X","type":"String","default":"ALL","prompt":"X","hidden":false}'`)
	paramCmd.Flags().StringArrayVar(&paramRemove, "remove", nil, "Parameter name to remove")
	root.AddCommand(paramCmd)

	// rebuild-tablix
	tablixCmd := &cobra.Command{
		Use:   "rebuild-tablix FILE SPEC.json",
		Short: "Rebuild a Tablix from a JSON spec (targeted by Name in spec, or first if absent)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.RebuildTablix(args[0], args[1], dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	root.AddCommand(tablixCmd)

	// tablix-set-cell
	var tscTablix string
	var tscRow, tscCol int
	var tscValue, tscFormat string
	var tscExpr bool
	tablixSetCellCmd := &cobra.Command{
		Use:   "tablix-set-cell FILE",
		Short: "Set the value of a single cell in a Tablix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.TablixSetCell(args[0], tscTablix, tscRow, tscCol, rdl.CellValue{
				Value:      tscValue,
				Expression: tscExpr,
				Format:     tscFormat,
			}, dryRun)
			if err != nil {
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	tablixSetCellCmd.Flags().StringVar(&tscTablix, "tablix", "", "Tablix Name (required)")
	tablixSetCellCmd.Flags().IntVar(&tscRow, "row", 0, "Row index (0-based)")
	tablixSetCellCmd.Flags().IntVar(&tscCol, "col", 0, "Column index (0-based)")
	tablixSetCellCmd.Flags().StringVar(&tscValue, "value", "", "Cell value (required)")
	tablixSetCellCmd.Flags().StringVar(&tscFormat, "format", "", "Optional format string (e.g. N2)")
	tablixSetCellCmd.Flags().BoolVar(&tscExpr, "expression", false, "Treat value as a VB expression")
	tablixSetCellCmd.MarkFlagRequired("tablix")
	tablixSetCellCmd.MarkFlagRequired("value")
	root.AddCommand(tablixSetCellCmd)

	// tablix-add-row
	var tarTablix, tarHeight, tarCellsJSON string
	var tarIndex int
	tablixAddRowCmd := &cobra.Command{
		Use:   "tablix-add-row FILE",
		Short: "Append (or insert) a row into a Tablix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			row := rdl.RowSpec{Height: tarHeight}
			if tarCellsJSON != "" {
				if err := json.Unmarshal([]byte(tarCellsJSON), &row.Cells); err != nil {
					return fmt.Errorf("invalid --cells JSON: %w", err)
				}
			}
			summary, err := rdl.TablixAddRow(args[0], tarTablix, row, tarIndex, dryRun)
			if err != nil {
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	tablixAddRowCmd.Flags().StringVar(&tarTablix, "tablix", "", "Tablix Name (required)")
	tablixAddRowCmd.Flags().StringVar(&tarHeight, "height", "0.5cm", "Row height")
	tablixAddRowCmd.Flags().StringVar(&tarCellsJSON, "cells", "", `JSON array of cells: [{"textbox":"X","value":"Y","colspan":2}]`)
	tablixAddRowCmd.Flags().IntVar(&tarIndex, "index", -1, "Insert position (-1 = append)")
	tablixAddRowCmd.MarkFlagRequired("tablix")
	root.AddCommand(tablixAddRowCmd)

	// tablix-remove-row
	var trrTablix string
	var trrIndex int
	tablixRemoveRowCmd := &cobra.Command{
		Use:   "tablix-remove-row FILE",
		Short: "Remove a row from a Tablix by index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.TablixRemoveRow(args[0], trrTablix, trrIndex, dryRun)
			if err != nil {
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	tablixRemoveRowCmd.Flags().StringVar(&trrTablix, "tablix", "", "Tablix Name (required)")
	tablixRemoveRowCmd.Flags().IntVar(&trrIndex, "index", 0, "Row index to remove (0-based)")
	tablixRemoveRowCmd.MarkFlagRequired("tablix")
	root.AddCommand(tablixRemoveRowCmd)

	// tablix-add-column
	var tacTablix, tacWidth string
	var tacIndex int
	tablixAddColumnCmd := &cobra.Command{
		Use:   "tablix-add-column FILE",
		Short: "Append (or insert) a column into a Tablix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.TablixAddColumn(args[0], tacTablix, tacWidth, tacIndex, dryRun)
			if err != nil {
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	tablixAddColumnCmd.Flags().StringVar(&tacTablix, "tablix", "", "Tablix Name (required)")
	tablixAddColumnCmd.Flags().StringVar(&tacWidth, "width", "2.5cm", "Column width")
	tablixAddColumnCmd.Flags().IntVar(&tacIndex, "index", -1, "Insert position (-1 = append)")
	tablixAddColumnCmd.MarkFlagRequired("tablix")
	root.AddCommand(tablixAddColumnCmd)

	// tablix-remove-column
	var trcTablix string
	var trcIndex int
	tablixRemoveColumnCmd := &cobra.Command{
		Use:   "tablix-remove-column FILE",
		Short: "Remove a column from a Tablix by index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.TablixRemoveColumn(args[0], trcTablix, trcIndex, dryRun)
			if err != nil {
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	tablixRemoveColumnCmd.Flags().StringVar(&trcTablix, "tablix", "", "Tablix Name (required)")
	tablixRemoveColumnCmd.Flags().IntVar(&trcIndex, "index", 0, "Column index to remove (0-based)")
	tablixRemoveColumnCmd.MarkFlagRequired("tablix")
	root.AddCommand(tablixRemoveColumnCmd)

	// fix-encoding
	encCmd := &cobra.Command{
		Use:   "fix-encoding FILE",
		Short: "Fix UTF-8 BOM and CRLF line endings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.FixEncoding(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	root.AddCommand(encCmd)

	// register
	regCmd := &cobra.Command{
		Use:   "register FILE RPTPROJ",
		Short: "Register an RDL in a .rptproj file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.Register(args[0], args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	root.AddCommand(regCmd)

	// validate
	var validateJSON bool
	valCmd := &cobra.Command{
		Use:   "validate FILE",
		Short: "Validate RDL structure (XML well-formedness + reference integrity)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := rdl.Validate(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			if validateJSON {
				b, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			fmt.Println(formatReportText(report))
			if !report.Pass {
				os.Exit(1)
			}
			return nil
		},
	}
	valCmd.Flags().BoolVar(&validateJSON, "json", false, "Emit the report as JSON instead of text")
	root.AddCommand(valCmd)

	return root.Execute()
}

// formatReportText renders the report for human reading. Errors first, then
// warnings; both prefixed with severity tag.
func formatReportText(r *rdl.ValidationReport) string {
	if r.Pass && len(r.Issues) == 0 {
		return fmt.Sprintf("Validation PASSED for %s (no issues)", r.File)
	}
	var b strings.Builder
	if r.Pass {
		fmt.Fprintf(&b, "Validation PASSED for %s with %d warning(s):\n", r.File, len(r.Issues))
	} else {
		errs := 0
		for _, i := range r.Issues {
			if i.Severity == rdl.SeverityError {
				errs++
			}
		}
		fmt.Fprintf(&b, "Validation FAILED for %s: %d error(s), %d warning(s):\n",
			r.File, errs, len(r.Issues)-errs)
	}
	for _, i := range r.Issues {
		loc := ""
		if i.XPath != "" {
			loc = " at " + i.XPath
		}
		fmt.Fprintf(&b, "  [%s]%s %s\n", i.Severity, loc, i.Message)
	}
	return strings.TrimRight(b.String(), "\n")
}

func parsePairs(items []string) ([]rdl.RenamePair, error) {
	pairs := make([]rdl.RenamePair, 0, len(items))
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid pair format %q, expected OLD:NEW", item)
		}
		pairs = append(pairs, rdl.RenamePair{Old: parts[0], New: parts[1]})
	}
	return pairs, nil
}

// parseJSONList unmarshals each item as a JSON object of type T.
// Used for --add flags that may contain colons or other special chars in
// values (SQL command text, connection strings). Identifiers don't need this
// -- the simpler OLD:NEW format is fine for renames.
func parseJSONList[T any](items []string) ([]T, error) {
	result := make([]T, 0, len(items))
	for _, item := range items {
		var v T
		if err := json.Unmarshal([]byte(item), &v); err != nil {
			return nil, fmt.Errorf("invalid JSON %q: %w", item, err)
		}
		result = append(result, v)
	}
	return result, nil
}

// inspectCmd builds a read-only cobra subcommand that loads an RDL file and
// prints the result of fn as indented JSON.
func inspectCmd(use, short string, fn func(*rdl.Document) any) *cobra.Command {
	return &cobra.Command{
		Use:   use + " FILE",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := rdl.Load(args[0])
			if err != nil {
				return err
			}
			out, err := json.MarshalIndent(fn(doc), "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
}
