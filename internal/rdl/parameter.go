package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// ParameterOps bundles Parameter mutations for a single Manage call.
type ParameterOps struct {
	Remove []string       `json:"remove,omitempty"`
	Add    []ParameterAdd `json:"add,omitempty"`
}

// ManageParameters applies the operations to the file.
func ManageParameters(path string, ops ParameterOps, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary := doc.ManageParameters(ops)
	return maybeSave(doc, path, summary, dryRun)
}

// ManageParameters applies ops to the document, returning a summary.
// Removing a parameter also strips any ReportParametersLayout CellDefinition
// that references it (orphaned layout references break SSRS rendering).
func (d *Document) ManageParameters(ops ParameterOps) string {
	var b strings.Builder

	for _, name := range ops.Remove {
		n := d.findNamedElement("ReportParameter", name)
		if n == nil {
			fmt.Fprintf(&b, "Removed parameter '%s' (not found)\n", name)
			continue
		}
		xmlquery.RemoveFromTree(n)
		// Also remove layout cells referencing this parameter.
		layoutCount := 0
		for _, ref := range xmlquery.Find(d.root, "//CellDefinition") {
			pn := child(ref, "ParameterName")
			if pn != nil && strings.TrimSpace(pn.InnerText()) == name {
				xmlquery.RemoveFromTree(ref)
				layoutCount++
			}
		}
		extra := ""
		if layoutCount > 0 {
			extra = fmt.Sprintf(" (+ %d layout cell(s))", layoutCount)
		}
		fmt.Fprintf(&b, "Removed parameter '%s'%s\n", name, extra)
	}

	// Compact the parameter layout grid: reassign RowIndex values to be
	// contiguous (no gaps) and update NumberOfRows. Visual Studio's SSRS
	// designer throws "Index was out of range" on sparse grids.
	d.compactParameterGrid()

	for _, a := range ops.Add {
		if d.exists("ReportParameter", a.Name) {
			fmt.Fprintf(&b, "Added parameter '%s' (already exists, skipped)\n", a.Name)
			continue
		}
		d.addParameter(a)
		fmt.Fprintf(&b, "Added parameter '%s' (type=%s)\n", a.Name, a.Type)
	}

	// After all add/remove operations, validate the grid. If the grid has
	// fewer cells than total parameters (e.g. hidden params not in grid),
	// remove the entire ReportParametersLayout so Visual Studio regenerates
	// it. Visual Studio crashes with "Index was out of range" on mismatched grids.
	d.sanitizeParameterGrid(&b)

	return strings.TrimRight(b.String(), "\n")
}

// addParameter constructs a <ReportParameter> element and appends it to
// <ReportParameters>.
func (d *Document) addParameter(spec ParameterAdd) {
	container := xmlquery.FindOne(d.root, "//ReportParameters")
	if container == nil {
		return
	}

	prompt := spec.Prompt
	if prompt == "" {
		prompt = spec.Name
	}

	p := createElement("ReportParameter", [2]string{"Name", spec.Name})
	xmlquery.AddChild(p, elementWithText("DataType", spec.Type))

	if spec.Default != "" {
		dv := createElement("DefaultValue")
		vals := createElement("Values")
		xmlquery.AddChild(vals, elementWithText("Value", spec.Default))
		xmlquery.AddChild(dv, vals)
		xmlquery.AddChild(p, dv)
	}
	if spec.Hidden {
		xmlquery.AddChild(p, elementWithText("Hidden", "true"))
	}
	xmlquery.AddChild(p, elementWithText("Prompt", prompt))

	childIndent := detectChildIndent(container)
	containerIndent := detectContainerIndent(container)
	appendIndentedWithSuffix(container, p, childIndent, containerIndent)
}

// compactParameterGrid reassigns RowIndex values in the parameter layout grid
// to be contiguous (no gaps) and updates NumberOfRows. Visual Studio's SSRS
// designer throws "Index was out of range" when the grid has sparse rows.
func (d *Document) compactParameterGrid() {
	grid := xmlquery.FindOne(d.root, "//GridLayoutDefinition")
	if grid == nil {
		return
	}

	// Collect all remaining CellDefinitions with their RowIndex.
	cells := xmlquery.Find(grid, ".//CellDefinition")
	if len(cells) == 0 {
		return
	}

	// Build a sorted list of unique RowIndex values and create a mapping.
	rowSet := make(map[int]bool)
	for _, cell := range cells {
		ri := child(cell, "RowIndex")
		if ri == nil {
			continue
		}
		var idx int
		fmt.Sscanf(strings.TrimSpace(ri.InnerText()), "%d", &idx)
		rowSet[idx] = true
	}

	// Create old→new row index mapping (contiguous).
	oldRows := make([]int, 0, len(rowSet))
	for r := range rowSet {
		oldRows = append(oldRows, r)
	}
	// Sort ascending.
	for i := range oldRows {
		for j := i + 1; j < len(oldRows); j++ {
			if oldRows[i] > oldRows[j] {
				oldRows[i], oldRows[j] = oldRows[j], oldRows[i]
			}
		}
	}
	rowMap := make(map[int]int)
	for newIdx, oldIdx := range oldRows {
		rowMap[oldIdx] = newIdx
	}

	// Update each CellDefinition's RowIndex.
	for _, cell := range cells {
		ri := child(cell, "RowIndex")
		if ri == nil {
			continue
		}
		var oldIdx int
		fmt.Sscanf(strings.TrimSpace(ri.InnerText()), "%d", &oldIdx)
		if newIdx, ok := rowMap[oldIdx]; ok && newIdx != oldIdx {
			setNodeText(ri, fmt.Sprintf("%d", newIdx))
		}
	}

	// Update NumberOfRows.
	nr := xmlquery.FindOne(grid, ".//NumberOfRows")
	if nr != nil {
		setNodeText(nr, fmt.Sprintf("%d", len(oldRows)))
	}
}

// sanitizeParameterGrid checks if the parameter layout grid is consistent
// with the actual parameters. If the grid has fewer cells than total
// parameters (e.g. hidden params not in grid), the entire
// ReportParametersLayout is removed so Visual Studio regenerates it.
// Visual Studio's SSRS designer crashes with "Index was out of range" when
// the grid doesn't cover all parameters.
func (d *Document) sanitizeParameterGrid(log *strings.Builder) {
	grid := xmlquery.FindOne(d.root, "//GridLayoutDefinition")
	if grid == nil {
		return
	}

	paramCount := len(xmlquery.Find(d.root, "//ReportParameter"))
	cellCount := len(xmlquery.Find(grid, ".//CellDefinition"))

	if cellCount < paramCount {
		layout := xmlquery.FindOne(d.root, "//ReportParametersLayout")
		if layout != nil {
			xmlquery.RemoveFromTree(layout)
			fmt.Fprintf(log, "Removed ReportParametersLayout (had %d cells for %d params — grid would crash VS designer)\n", cellCount, paramCount)
		}
	}
}
