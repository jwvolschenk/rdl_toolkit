package rdl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
)

// This file holds per-element Tablix edit operations: set one cell, add/remove
// a row, add/remove a column. Each operation targets a Tablix by name.
// Use rdl_list_tablixes to discover names and current cell layout first.
//
// Colspan caveat: column add/remove walks each row's TablixCells by index
// and ignores ColSpan semantics. If a row has merged cells, manually fix up
// the ColSpan values after a column edit.

// ── SetCell ────────────────────────────────────────────────────────────────

// CellValue is the new content for a cell. By default the Value is written
// verbatim (suitable for static text like "Header"). Set Expression=true to
// mark it as a VB expression — SSRS will then evaluate it at render time.
type CellValue struct {
	Value      string `json:"value"`
	Expression bool   `json:"expression,omitempty"`
	Format     string `json:"format,omitempty"`
}

// TablixSetCell sets the value of a single cell at (row, col). Indexes are
// 0-based; col indexes the cell position within the row (NOT the logical
// column — colspans are not expanded).
func TablixSetCell(path, tablixName string, row, col int, value CellValue, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary, err := doc.SetTablixCell(tablixName, row, col, value)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, path, summary, dryRun)
}

// SetTablixCell edits one cell in place.
func (d *Document) SetTablixCell(tablixName string, row, col int, value CellValue) (string, error) {
	t := d.findTablix(tablixName)
	if t == nil {
		return "", fmt.Errorf("tablix %q not found", tablixName)
	}
	cell, err := locateCell(t, row, col)
	if err != nil {
		return "", err
	}

	// Find or create the Textbox > Paragraphs > Paragraph > TextRuns > TextRun > Value chain.
	tb := child(cell, "CellContents")
	if tb == nil {
		tb = createElement("CellContents")
		xmlquery.AddChild(cell, tb)
	}
	textbox := child(tb, "Textbox")
	if textbox == nil {
		// synthesise a textbox with a unique-ish name
		textbox = createElement("Textbox", [2]string{"Name", fmt.Sprintf("%s_R%dC%d", tablixName, row, col)})
		xmlquery.AddChild(tb, textbox)
	}
	setTextboxValue(textbox, value)

	return fmt.Sprintf("Set cell (%d,%d) in '%s' to %q", row, col, tablixName, value.Value), nil
}

// setTextboxValue ensures the textbox has a single TextRun holding value.
func setTextboxValue(textbox *xmlquery.Node, value CellValue) {
	// Ensure Structure: Paragraphs > Paragraph > TextRuns > TextRun
	paragraphs := child(textbox, "Paragraphs")
	if paragraphs == nil {
		paragraphs = createElement("Paragraphs")
		xmlquery.AddChild(textbox, paragraphs)
	}
	paragraph := child(paragraphs, "Paragraph")
	if paragraph == nil {
		paragraph = createElement("Paragraph")
		xmlquery.AddChild(paragraphs, paragraph)
	}
	textruns := child(paragraph, "TextRuns")
	if textruns == nil {
		textruns = createElement("TextRuns")
		xmlquery.AddChild(paragraph, textruns)
	}
	textrun := child(textruns, "TextRun")
	if textrun == nil {
		textrun = createElement("TextRun")
		xmlquery.AddChild(textruns, textrun)
	}
	// Replace existing Value/Format with the new ones.
	for _, name := range []string{"Value", "Format"} {
		for {
			c := child(textrun, name)
			if c == nil {
				break
			}
			xmlquery.RemoveFromTree(c)
		}
	}
	xmlquery.AddChild(textrun, elementWithText("Value", value.Value))
	if value.Format != "" {
		xmlquery.AddChild(textrun, elementWithText("Format", value.Format))
	}
}

// locateCell returns the (row,col)-th TablixCell of the tablix.
func locateCell(t *xmlquery.Node, row, col int) (*xmlquery.Node, error) {
	rows := xmlquery.Find(t, ".//TablixBody/TablixRows/TablixRow")
	if row < 0 || row >= len(rows) {
		return nil, fmt.Errorf("row index %d out of range (have %d rows)", row, len(rows))
	}
	cells := xmlquery.Find(rows[row], "TablixCells/TablixCell")
	if col < 0 || col >= len(cells) {
		return nil, fmt.Errorf("col index %d out of range (row %d has %d cells)", col, row, len(cells))
	}
	return cells[col], nil
}

// ── AddRow / RemoveRow ─────────────────────────────────────────────────────

// RowSpec is one row for Add. Cells may be empty (a row of empty textboxes is
// still added so the row has the right number of cells).
type RowSpec struct {
	Height string      `json:"height,omitempty"`
	Cells  []TablixCell `json:"cells,omitempty"`
}

// TablixAddRow appends (atIndex=-1) or inserts a row into the named Tablix.
func TablixAddRow(path, tablixName string, row RowSpec, atIndex int, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary, err := doc.AddTablixRow(tablixName, row, atIndex)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, path, summary, dryRun)
}

// AddTablixRow inserts a row. atIndex=-1 (or >= current row count) appends.
// A matching <TablixMember /> is added to TablixRowHierarchy so the row is visible.
func (d *Document) AddTablixRow(tablixName string, spec RowSpec, atIndex int) (string, error) {
	t := d.findTablix(tablixName)
	if t == nil {
		return "", fmt.Errorf("tablix %q not found", tablixName)
	}

	// Determine target column count from existing rows or columns.
	rows := xmlquery.Find(t, ".//TablixBody/TablixRows/TablixRow")
	colCount := 0
	if len(rows) > 0 {
		cells := xmlquery.Find(rows[0], "TablixCells/TablixCell")
		colCount = len(cells)
	} else {
		colCount = len(xmlquery.Find(t, ".//TablixBody/TablixColumns/TablixColumn"))
	}
	if colCount == 0 {
		return "", fmt.Errorf("tablix %q has no columns", tablixName)
	}

	height := spec.Height
	if height == "" {
		height = "0.5cm"
	}

	// Build the TablixRow XML.
	cells := spec.Cells
	if len(cells) == 0 {
		// empty cells: synthesize placeholders
		cells = make([]TablixCell, colCount)
	}
	// Build via the same path as Rebuild for consistency.
	rowSpec := TablixRow{Height: height, Cells: cells}
	xmlStr := buildTablixRowXML(rowSpec, tablixName)
	newNode, err := parseFragmentRow(xmlStr)
	if err != nil {
		return "", fmt.Errorf("building row: %w", err)
	}

	rowsContainer := xmlquery.FindOne(t, ".//TablixBody/TablixRows")
	if rowsContainer == nil {
		return "", fmt.Errorf("tablix %q has no TablixRows container", tablixName)
	}

	idx := atIndex
	if idx < 0 || idx >= len(rows) {
		xmlquery.AddChild(rowsContainer, newTextNode("\n            "))
		xmlquery.AddChild(rowsContainer, newNode)
		xmlquery.AddChild(rowsContainer, newTextNode("\n          "))
		idx = len(rows)
	} else {
		xmlquery.AddSibling(rows[idx], newTextNode("\n            "))
		xmlquery.AddSibling(rows[idx], newNode)
		xmlquery.AddSibling(rows[idx], newTextNode("\n          "))
	}

	// Add a TablixMember to row hierarchy.
	hier := xmlquery.FindOne(t, ".//TablixRowHierarchy/TablixMembers")
	if hier != nil {
		members := xmlquery.Find(hier, "TablixMember")
		member := createElement("TablixMember")
		if idx < 0 || idx >= len(members) {
			xmlquery.AddChild(hier, member)
		} else {
			xmlquery.AddSibling(members[idx], member)
		}
	}

	return fmt.Sprintf("Added row at index %d to '%s'", idx, tablixName), nil
}

// buildTablixRowXML produces a <TablixRow>...</TablixRow> fragment.
func buildTablixRowXML(row TablixRow, tablixName string) string {
	var b strings.Builder
	h := row.Height
	if h == "" {
		h = "0.5cm"
	}
	fmt.Fprintf(&b, `<TablixRow><Height>%s</Height><TablixCells>`, h)
	for i, cell := range row.Cells {
		tbName := cell.Textbox
		if tbName == "" {
			tbName = fmt.Sprintf("%s_R%d", tablixName, i)
		}
		b.WriteString(`<TablixCell><CellContents>`)
		fmt.Fprintf(&b, `<Textbox Name=%q>`, tbName)
		if cell.Value != "" {
			fmt.Fprintf(&b, `<Paragraphs><Paragraph><TextRuns><TextRun><Value>%s</Value></TextRun></TextRuns></Paragraph></Paragraphs>`,
				escapeXMLText(cell.Value))
		}
		b.WriteString(`</Textbox>`)
		if cell.Colspan > 1 {
			fmt.Fprintf(&b, `<ColSpan>%d</ColSpan>`, cell.Colspan)
		}
		b.WriteString(`</CellContents></TablixCell>`)
	}
	b.WriteString(`</TablixCells></TablixRow>`)
	return b.String()
}

// parseFragmentRow parses a <TablixRow> fragment.
func parseFragmentRow(xmlStr string) (*xmlquery.Node, error) {
	return parseFragment(xmlStr)
}

// TablixRemoveRow removes the row at index from the named Tablix.
func TablixRemoveRow(path, tablixName string, index int, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary, err := doc.RemoveTablixRow(tablixName, index)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, path, summary, dryRun)
}

// RemoveTablixRow removes a row and its corresponding TablixMember.
func (d *Document) RemoveTablixRow(tablixName string, index int) (string, error) {
	t := d.findTablix(tablixName)
	if t == nil {
		return "", fmt.Errorf("tablix %q not found", tablixName)
	}
	rows := xmlquery.Find(t, ".//TablixBody/TablixRows/TablixRow")
	if index < 0 || index >= len(rows) {
		return "", fmt.Errorf("row index %d out of range (have %d rows)", index, len(rows))
	}
	xmlquery.RemoveFromTree(rows[index])

	// Match in hierarchy.
	if members := xmlquery.Find(t, ".//TablixRowHierarchy/TablixMembers/TablixMember"); len(members) > index {
		xmlquery.RemoveFromTree(members[index])
	}
	return fmt.Sprintf("Removed row %d from '%s'", index, tablixName), nil
}

// ── AddColumn / RemoveColumn ───────────────────────────────────────────────

// TablixAddColumn appends (atIndex=-1) or inserts a column.
// A new empty cell is added to every row; column hierarchy gets a new member.
func TablixAddColumn(path, tablixName, width string, atIndex int, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary, err := doc.AddTablixColumn(tablixName, width, atIndex)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, path, summary, dryRun)
}

// AddTablixColumn inserts a column. width must be a valid RDL size like "2.5cm".
func (d *Document) AddTablixColumn(tablixName, width string, atIndex int) (string, error) {
	if width == "" {
		width = "2.5cm"
	}
	t := d.findTablix(tablixName)
	if t == nil {
		return "", fmt.Errorf("tablix %q not found", tablixName)
	}

	// 1. Insert <TablixColumn> into TablixColumns.
	colsContainer := xmlquery.FindOne(t, ".//TablixBody/TablixColumns")
	if colsContainer == nil {
		return "", fmt.Errorf("tablix %q has no TablixColumns", tablixName)
	}
	newCol := createElement("TablixColumn")
	xmlquery.AddChild(newCol, elementWithText("Width", width))

	existingCols := xmlquery.Find(colsContainer, "TablixColumn")
	idx := atIndex
	if idx < 0 || idx >= len(existingCols) {
		xmlquery.AddChild(colsContainer, newCol)
		idx = len(existingCols)
	} else {
		xmlquery.AddSibling(existingCols[idx], newCol)
	}

	// 2. Add an empty cell to each row.
	rows := xmlquery.Find(t, ".//TablixBody/TablixRows/TablixRow")
	for ri, row := range rows {
		cellsContainer := child(row, "TablixCells")
		if cellsContainer == nil {
			continue
		}
		newCell := createElement("TablixCell")
		cc := createElement("CellContents")
		xmlquery.AddChild(cc, createElement("Textbox", [2]string{"Name", fmt.Sprintf("%s_Col%d_Row%d", tablixName, idx, ri)}))
		xmlquery.AddChild(newCell, cc)

		existingCells := xmlquery.Find(cellsContainer, "TablixCell")
		if idx >= len(existingCells) {
			xmlquery.AddChild(cellsContainer, newCell)
		} else {
			xmlquery.AddSibling(existingCells[idx], newCell)
		}
	}

	// 3. Add a TablixMember to column hierarchy.
	if hier := xmlquery.FindOne(t, ".//TablixColumnHierarchy/TablixMembers"); hier != nil {
		members := xmlquery.Find(hier, "TablixMember")
		member := createElement("TablixMember")
		if idx >= len(members) {
			xmlquery.AddChild(hier, member)
		} else {
			xmlquery.AddSibling(members[idx], member)
		}
	}

	return fmt.Sprintf("Added column at index %d (width %s) to '%s'", idx, width, tablixName), nil
}

// TablixRemoveColumn removes a column by index. Each row loses its cell at
// that index. ColSpan values are NOT adjusted — fix manually if needed.
func TablixRemoveColumn(path, tablixName string, index int, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary, err := doc.RemoveTablixColumn(tablixName, index)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, path, summary, dryRun)
}

// RemoveTablixColumn removes the column at index.
func (d *Document) RemoveTablixColumn(tablixName string, index int) (string, error) {
	t := d.findTablix(tablixName)
	if t == nil {
		return "", fmt.Errorf("tablix %q not found", tablixName)
	}

	// 1. Remove TablixColumn.
	cols := xmlquery.Find(t, ".//TablixBody/TablixColumns/TablixColumn")
	if index < 0 || index >= len(cols) {
		return "", fmt.Errorf("column index %d out of range (have %d columns)", index, len(cols))
	}
	xmlquery.RemoveFromTree(cols[index])

	// 2. Remove cell at index from each row.
	for _, row := range xmlquery.Find(t, ".//TablixBody/TablixRows/TablixRow") {
		cells := xmlquery.Find(row, "TablixCells/TablixCell")
		if index < len(cells) {
			xmlquery.RemoveFromTree(cells[index])
		}
	}

	// 3. Remove column hierarchy member.
	if members := xmlquery.Find(t, ".//TablixColumnHierarchy/TablixMembers/TablixMember"); index < len(members) {
		xmlquery.RemoveFromTree(members[index])
	}

	return fmt.Sprintf("Removed column %d from '%s'", index, tablixName), nil
}

// readTablixSpec reads and unmarshals a TablixSpec from a JSON file path.
func readTablixSpec(specPath string) (TablixSpec, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return TablixSpec{}, fmt.Errorf("reading spec file: %w", err)
	}
	var spec TablixSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return TablixSpec{}, fmt.Errorf("parsing spec JSON: %w", err)
	}
	return spec, nil
}
