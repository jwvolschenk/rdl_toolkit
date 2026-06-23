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
	// fewer cells than visible parameters (e.g. hidden params not in grid),
	// remove the entire ReportParametersLayout so Visual Studio regenerates
	// it. Visual Studio crashes with "Index was out of range" on mismatched grids.
	d.sanitizeParameterGrid(&b)

	// If the layout was removed (or never existed), auto-generate one for
	// visible parameters so the report opens in Visual Studio without manual edits.
	d.ensureParameterLayout(&b)

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
// with the actual parameters. If the grid has fewer cells than VISIBLE
// parameters, the entire ReportParametersLayout is removed so Visual Studio
// regenerates it. Hidden parameters don't need grid cells.
// Visual Studio's SSRS designer crashes with "Index was out of range" when
// the grid doesn't cover all visible parameters.
func (d *Document) sanitizeParameterGrid(log *strings.Builder) {
	grid := xmlquery.FindOne(d.root, "//GridLayoutDefinition")
	if grid == nil {
		return
	}

	// Count only visible parameters (hidden ones don't need grid cells).
	visibleParamCount := 0
	for _, rp := range xmlquery.Find(d.root, "//ReportParameter") {
		hn := child(rp, "Hidden")
		if hn == nil || strings.TrimSpace(hn.InnerText()) != "true" {
			visibleParamCount++
		}
	}
	cellCount := len(xmlquery.Find(grid, ".//CellDefinition"))

	if cellCount < visibleParamCount {
		layout := xmlquery.FindOne(d.root, "//ReportParametersLayout")
		if layout != nil {
			xmlquery.RemoveFromTree(layout)
			fmt.Fprintf(log, "Removed ReportParametersLayout (had %d cells for %d visible params — grid would crash VS designer)\n", cellCount, visibleParamCount)
		}
	}
}

// ensureParameterLayout generates a ReportParametersLayout grid if one is
// missing and there are visible parameters. Called after all parameter
// operations to ensure the report opens in Visual Studio.
func (d *Document) ensureParameterLayout(log *strings.Builder) {
	// Don't generate if layout already exists.
	if xmlquery.FindOne(d.root, "//ReportParametersLayout") != nil {
		return
	}

	// Collect visible parameters.
	var visible []string
	for _, rp := range xmlquery.Find(d.root, "//ReportParameter") {
		hn := child(rp, "Hidden")
		if hn == nil || strings.TrimSpace(hn.InnerText()) != "true" {
			if name := rp.SelectAttr("Name"); name != "" {
				visible = append(visible, name)
			}
		}
	}
	if len(visible) == 0 {
		return
	}

	// Generate a 2-column grid layout.
	cols := 2
	rows := (len(visible) + cols - 1) / cols

	layout := createElement("ReportParametersLayout")
	grid := createElement("GridLayoutDefinition")
	xmlquery.AddChild(grid, elementWithText("NumberOfColumns", fmt.Sprintf("%d", cols)))
	xmlquery.AddChild(grid, elementWithText("NumberOfRows", fmt.Sprintf("%d", rows)))

	cellDefs := createElement("CellDefinitions")
	for i, name := range visible {
		cd := createElement("CellDefinition")
		xmlquery.AddChild(cd, elementWithText("ColumnIndex", fmt.Sprintf("%d", i%cols)))
		xmlquery.AddChild(cd, elementWithText("RowIndex", fmt.Sprintf("%d", i/cols)))
		xmlquery.AddChild(cd, elementWithText("ParameterName", name))
		xmlquery.AddChild(cellDefs, cd)
	}
	xmlquery.AddChild(grid, cellDefs)
	xmlquery.AddChild(layout, grid)

	// Insert after ReportParameters (or before rd:ReportID).
	reportParams := xmlquery.FindOne(d.root, "//ReportParameters")
	if reportParams != nil {
		insertAfter(reportParams, layout)
	} else {
		report := d.reportRoot()
		if report != nil {
			appendIndented(report, layout, depthOf(report))
		}
	}
	fmt.Fprintf(log, "Generated ReportParametersLayout (%d cols × %d rows for %d visible params)\n", cols, rows, len(visible))
}
