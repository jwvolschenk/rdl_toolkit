package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/rdl-toolkit/internal/rdl"
)

func Execute() error {
	root := &cobra.Command{
		Use:   "rdl-tool",
		Short: "CLI tool for manipulating SSRS RDL files",
	}

	// clone
	var cloneSource, cloneTarget string
	cloneCmd := &cobra.Command{
		Use:   "clone",
		Short: "Copy RDL file with new ReportID",
		RunE: func(cmd *cobra.Command, args []string) error {
			newID, err := rdl.Clone(cloneSource, cloneTarget)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Printf("Cloned %s -> %s\nNew ReportID: %s\n", cloneSource, cloneTarget, newID)
			return nil
		},
	}
	cloneCmd.Flags().StringVar(&cloneSource, "source", "", "Source RDL file")
	cloneCmd.Flags().StringVar(&cloneTarget, "target", "", "Target RDL file")
	cloneCmd.MarkFlagRequired("source")
	cloneCmd.MarkFlagRequired("target")
	root.AddCommand(cloneCmd)

	// update-metadata
	var metaDesc, metaTitle, metaOrientation string
	metaCmd := &cobra.Command{
		Use:   "update-metadata FILE",
		Short: "Update report metadata (description, title, orientation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			count, err := rdl.UpdateMetadata(args[0], metaDesc, metaTitle, metaOrientation)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Printf("Updated %d metadata field(s) in %s\n", count, args[0])
			return nil
		},
	}
	metaCmd.Flags().StringVar(&metaDesc, "description", "", "Pipe-delimited description")
	metaCmd.Flags().StringVar(&metaTitle, "title", "", "Report title")
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
			count, err := rdl.SwapMacros(args[0], pairs)
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
			count, err := rdl.SwapFields(args[0], pairs)
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
	var dsAdd, dsRemove, dsRename []string
	dsCmd := &cobra.Command{
		Use:   "manage-datasources FILE",
		Short: "Add, remove, or rename DataSources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseTriple(dsAdd)
			if err != nil {
				return err
			}
			renames, err := parsePairs(dsRename)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageDataSources(args[0], adds, dsRemove, renames)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	dsCmd.Flags().StringSliceVar(&dsAdd, "add", nil, "NAME:PROVIDER:CONNECTSTRING")
	dsCmd.Flags().StringSliceVar(&dsRemove, "remove", nil, "DataSource name to remove")
	dsCmd.Flags().StringSliceVar(&dsRename, "rename", nil, "OLD:NEW")
	root.AddCommand(dsCmd)

	// manage-datasets
	var dtAdd, dtRemove, dtRename, dtAddField []string
	dtCmd := &cobra.Command{
		Use:   "manage-datasets FILE",
		Short: "Add, remove, or rename DataSets",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseDatasetAdd(dtAdd)
			if err != nil {
				return err
			}
			renames, err := parsePairs(dtRename)
			if err != nil {
				return err
			}
			addFields, err := parsePairs(dtAddField)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageDataSets(args[0], adds, dtRemove, renames, addFields)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	dtCmd.Flags().StringSliceVar(&dtAdd, "add", nil, "NAME:DATASOURCE:CMDTEXT:F1,F2,...")
	dtCmd.Flags().StringSliceVar(&dtRemove, "remove", nil, "DataSet name to remove")
	dtCmd.Flags().StringSliceVar(&dtRename, "rename", nil, "OLD:NEW")
	dtCmd.Flags().StringSliceVar(&dtAddField, "add-field", nil, "DATASET:FIELD")
	root.AddCommand(dtCmd)

	// manage-parameters
	var paramAdd, paramRemove []string
	paramCmd := &cobra.Command{
		Use:   "manage-parameters FILE",
		Short: "Add or remove ReportParameters",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adds, err := parseParamAdd(paramAdd)
			if err != nil {
				return err
			}
			summary, err := rdl.ManageParameters(args[0], adds, paramRemove)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	paramCmd.Flags().StringSliceVar(&paramAdd, "add", nil, "NAME:TYPE:DEFAULT:HIDDEN")
	paramCmd.Flags().StringSliceVar(&paramRemove, "remove", nil, "Parameter name to remove")
	root.AddCommand(paramCmd)

	// rebuild-tablix
	tablixCmd := &cobra.Command{
		Use:   "rebuild-tablix FILE SPEC.json",
		Short: "Rebuild the first Tablix from a JSON spec",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.RebuildTablix(args[0], args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	root.AddCommand(tablixCmd)

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
	valCmd := &cobra.Command{
		Use:   "validate FILE",
		Short: "Validate RDL structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			summary, err := rdl.Validate(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return err
			}
			fmt.Println(summary)
			return nil
		},
	}
	root.AddCommand(valCmd)

	return root.Execute()
}

func parsePairs(items []string) ([][2]string, error) {
	var pairs [][2]string
	for _, item := range items {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid pair format %q, expected OLD:NEW", item)
		}
		pairs = append(pairs, [2]string{parts[0], parts[1]})
	}
	return pairs, nil
}

func parseTriple(items []string) ([][3]string, error) {
	var result [][3]string
	for _, item := range items {
		parts := strings.SplitN(item, ":", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid format %q, expected NAME:PROVIDER:CONNECTSTRING", item)
		}
		result = append(result, [3]string{parts[0], parts[1], parts[2]})
	}
	return result, nil
}

func parseDatasetAdd(items []string) ([]rdl.DatasetAddInfo, error) {
	var result []rdl.DatasetAddInfo
	for _, item := range items {
		parts := strings.SplitN(item, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid format %q, expected NAME:DATASOURCE:CMDTEXT:F1,F2,...", item)
		}
		var fields []string
		if parts[3] != "" {
			fields = strings.Split(parts[3], ",")
		}
		result = append(result, rdl.DatasetAddInfo{Name: parts[0], DS: parts[1], CmdText: parts[2], Fields: fields})
	}
	return result, nil
}

func parseParamAdd(items []string) ([]rdl.ParamAddInfo, error) {
	var result []rdl.ParamAddInfo
	for _, item := range items {
		parts := strings.SplitN(item, ":", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid format %q, expected NAME:TYPE:DEFAULT:HIDDEN", item)
		}
		result = append(result, rdl.ParamAddInfo{Name: parts[0], Type: parts[1], Default: parts[2], Hidden: parts[3]})
	}
	return result, nil
}
