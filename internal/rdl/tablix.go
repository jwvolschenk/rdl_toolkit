package rdl

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// TablixSpec defines a full Tablix rebuild. Replaces every child of an
// existing <Tablix>: body (columns + rows), column hierarchy, row hierarchy,
// and dataset binding. The Name attribute of the existing Tablix is preserved
// unless the spec supplies one.
type TablixSpec struct {
	// Name targets a specific Tablix. If empty, the first Tablix in the
	// document is rebuilt and keeps its existing Name.
	Name    string       `json:"name,omitempty"`
	Columns []float64    `json:"columns"`
	Dataset string       `json:"dataset,omitempty"`
	Rows    []TablixRow  `json:"rows"`
}

// TablixRow is one row of the rebuild spec.
type TablixRow struct {
	Height string       `json:"height,omitempty"`
	Cells  []TablixCell `json:"cells"`
}

// TablixCell is one cell. Colspan consumes additional column hierarchy
// members; do NOT add placeholder cells for colspan — SSRS does not want them.
type TablixCell struct {
	Textbox string     `json:"textbox,omitempty"`
	Value   string     `json:"value,omitempty"`
	Colspan int        `json:"colspan,omitempty"`
	Style   *CellStyle `json:"style,omitempty"`
	Format  string     `json:"format,omitempty"`
}

// CellStyle is a small subset of RDL style. Zero-value fields are omitted.
type CellStyle struct {
	FontSize   string `json:"fontSize,omitempty"`
	FontWeight string `json:"fontWeight,omitempty"`
	FontColor  string `json:"fontColor,omitempty"`
	TextAlign  string `json:"textAlign,omitempty"`
	BgColor    string `json:"bgColor,omitempty"`
}

// RebuildTablix replaces the named (or first) Tablix with one built from spec.
// The spec is read from specPath as JSON. File-based wrapper.
func RebuildTablix(rdlPath, specPath string, dryRun bool) (string, error) {
	doc, err := Load(rdlPath)
	if err != nil {
		return "", err
	}
	spec, err := readTablixSpec(specPath)
	if err != nil {
		return "", err
	}
	summary, err := doc.RebuildTablix(spec)
	if err != nil {
		return "", err
	}
	return maybeSave(doc, rdlPath, summary, dryRun)
}

// RebuildTablix rebuilds the targeted Tablix in place.
func (d *Document) RebuildTablix(spec TablixSpec) (string, error) {
	if len(spec.Columns) == 0 {
		return "", fmt.Errorf("spec must declare at least one column")
	}
	existing := d.findTablix(spec.Name)
	if existing == nil {
		if spec.Name != "" {
			return "", fmt.Errorf("tablix %q not found", spec.Name)
		}
		return "", fmt.Errorf("no tablix in document")
	}
	preserveName := spec.Name
	if preserveName == "" {
		preserveName = existing.SelectAttr("Name")
	}

	// Validate cell counts: each row's effective column count must equal len(Columns).
	for i, r := range spec.Rows {
		got := 0
		for _, c := range r.Cells {
			cs := c.Colspan
			if cs < 1 {
				cs = 1
			}
			got += cs
		}
		if got != len(spec.Columns) {
			return "", fmt.Errorf("row %d: effective cell width %d does not match column count %d", i, got, len(spec.Columns))
		}
	}

	xmlStr := buildTablixXML(spec, preserveName)
	newNode, err := parseFragment(xmlStr)
	if err != nil {
		return "", fmt.Errorf("building tablix: %w", err)
	}
	xmlquery.AddSibling(existing, newNode)
	xmlquery.RemoveFromTree(existing)

	totalCells := 0
	for _, r := range spec.Rows {
		totalCells += len(r.Cells)
	}
	return fmt.Sprintf("Rebuilt Tablix '%s': %d columns, %d rows, %d cells",
		preserveName, len(spec.Columns), len(spec.Rows), totalCells), nil
}

// buildTablixXML returns the XML string for a Tablix element matching spec.
// Every row gets a static <TablixMember /> (no groups). Cell-level ColSpan
// is emitted as <ColSpan>N</ColSpan> inside CellContents — no placeholder cells.
func buildTablixXML(spec TablixSpec, name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<Tablix Name=%q>`, name)
	b.WriteString(`<TablixBody><TablixColumns>`)
	for _, w := range spec.Columns {
		fmt.Fprintf(&b, `<TablixColumn><Width>%vcm</Width></TablixColumn>`, w)
	}
	b.WriteString(`</TablixColumns><TablixRows>`)
	for _, row := range spec.Rows {
		h := row.Height
		if h == "" {
			h = "0.5cm"
		}
		fmt.Fprintf(&b, `<TablixRow><Height>%s</Height><TablixCells>`, h)
		for _, cell := range row.Cells {
			b.WriteString(`<TablixCell><CellContents>`)
			tbName := cell.Textbox
			if tbName == "" {
				tbName = name + "_Cell"
			}
			fmt.Fprintf(&b, `<Textbox Name=%q>`, tbName)
			writeTextboxBody(&b, cell)
			b.WriteString(`</Textbox>`)
			if cell.Colspan > 1 {
				fmt.Fprintf(&b, `<ColSpan>%d</ColSpan>`, cell.Colspan)
			}
			b.WriteString(`</CellContents></TablixCell>`)
		}
		b.WriteString(`</TablixCells></TablixRow>`)
	}
	b.WriteString(`</TablixRows></TablixBody>`)

	// Column hierarchy: one static member per column.
	b.WriteString(`<TablixColumnHierarchy><TablixMembers>`)
	for range spec.Columns {
		b.WriteString(`<TablixMember />`)
	}
	b.WriteString(`</TablixMembers></TablixColumnHierarchy>`)

	// Row hierarchy: one static member per row. No grouping — agents that
	// need groups can edit the hierarchy after rebuild.
	b.WriteString(`<TablixRowHierarchy><TablixMembers>`)
	for range spec.Rows {
		b.WriteString(`<TablixMember />`)
	}
	b.WriteString(`</TablixMembers></TablixRowHierarchy>`)

	if spec.Dataset != "" {
		fmt.Fprintf(&b, `<DataSetName>%s</DataSetName>`, spec.Dataset)
	}
	b.WriteString(`</Tablix>`)
	return b.String()
}

// writeTextboxBody writes the mandatory Paragraphs subtree and optional
// textbox-level Style. SSRS requires every Textbox to contain Paragraphs,
// even when the cell value is empty.
func writeTextboxBody(b *strings.Builder, cell TablixCell) {
	b.WriteString(`<Paragraphs><Paragraph><TextRuns><TextRun>`)
	fmt.Fprintf(b, `<Value>%s</Value>`, escapeXMLText(cell.Value))
	if cell.Format != "" || hasTextRunStyle(cell.Style) {
		b.WriteString(`<Style>`)
		if cell.Format != "" {
			fmt.Fprintf(b, `<Format>%s</Format>`, escapeXMLText(cell.Format))
		}
		if cell.Style != nil {
			writeTextRunStyle(b, cell.Style)
		}
		b.WriteString(`</Style>`)
	}
	b.WriteString(`</TextRun></TextRuns>`)
	if cell.Style != nil && cell.Style.TextAlign != "" {
		fmt.Fprintf(b, `<Style><TextAlign>%s</TextAlign></Style>`, cell.Style.TextAlign)
	}
	b.WriteString(`</Paragraph></Paragraphs>`)
	if cell.Style != nil && cell.Style.BgColor != "" {
		fmt.Fprintf(b, `<Style><BackgroundColor>%s</BackgroundColor></Style>`, cell.Style.BgColor)
	}
}

func hasTextRunStyle(s *CellStyle) bool {
	return s != nil && (s.FontWeight != "" || s.FontSize != "" || s.FontColor != "")
}

func writeTextRunStyle(b *strings.Builder, s *CellStyle) {
	if s.FontWeight != "" {
		fmt.Fprintf(b, `<FontWeight>%s</FontWeight>`, s.FontWeight)
	}
	if s.FontSize != "" {
		fmt.Fprintf(b, `<FontSize>%s</FontSize>`, s.FontSize)
	}
	if s.FontColor != "" {
		fmt.Fprintf(b, `<Color>%s</Color>`, s.FontColor)
	}
}

// escapeXMLText uses stdlib for XML character escaping. Replaces the
// hand-rolled xmlEscapeText from the byte-hacking era.
func escapeXMLText(s string) string {
	var buf strings.Builder
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}

// findTablix returns the named Tablix, or the first Tablix if name is empty.
func (d *Document) findTablix(name string) *xmlquery.Node {
	if name == "" {
		return xmlquery.FindOne(d.root, "//Tablix")
	}
	return xmlquery.FindOne(d.root, fmt.Sprintf(`//Tablix[@Name=%q]`, name))
}
