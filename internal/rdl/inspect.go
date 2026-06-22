package rdl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
)

// This file holds the read-only inspection helpers backed by the XML tree.
// Each returns plain structs that JSON-serialise cleanly for MCP output.

// ── Overview ───────────────────────────────────────────────────────────────

// Overview is the high-level fingerprint of a report. Returned by `rdl_inspect`.
type Overview struct {
	ReportID        string `json:"reportId,omitempty"`
	Description     string `json:"description,omitempty"`
	Language        string `json:"language,omitempty"`
	Author          string `json:"author,omitempty"`
	Orientation     string `json:"orientation,omitempty"` // Portrait or Landscape
	PageWidth       string `json:"pageWidth,omitempty"`
	PageHeight      string `json:"pageHeight,omitempty"`
	DataSourceCount int    `json:"dataSourceCount"`
	DataSetCount    int    `json:"dataSetCount"`
	ParameterCount  int    `json:"parameterCount"`
	TablixCount     int    `json:"tablixCount"`
}

// Overview returns a top-level summary of the report. Always non-nil.
func (d *Document) Overview() *Overview {
	o := &Overview{}
	if n := xmlquery.FindOne(d.root, "//*[local-name()='ReportID']"); n != nil {
		o.ReportID = strings.TrimSpace(n.InnerText())
	}
	o.Description = textOf(d.root, "//Description")
	o.Language = textOf(d.root, "//Language")
	o.Author = textOf(d.root, "//Author")
	o.PageWidth = textOf(d.root, "//PageWidth")
	o.PageHeight = textOf(d.root, "//PageHeight")
	o.Orientation = orientationOf(o.PageWidth, o.PageHeight)
	o.DataSourceCount = countByPath(d.root, "//DataSource")
	o.DataSetCount = countByPath(d.root, "//DataSet")
	o.ParameterCount = countByPath(d.root, "//ReportParameter")
	o.TablixCount = countByPath(d.root, "//Tablix")
	return o
}

// ── DataSources ────────────────────────────────────────────────────────────

// DataSourceSummary describes one <DataSource> entry.
type DataSourceSummary struct {
	Name          string `json:"name"`
	Provider      string `json:"provider,omitempty"`
	ConnectString string `json:"connectString,omitempty"`
	SecurityType  string `json:"securityType,omitempty"`
	DataSourceID  string `json:"dataSourceId,omitempty"`
}

// ListDataSources returns every DataSource declared in the report.
func (d *Document) ListDataSources() []DataSourceSummary {
	nodes := xmlquery.Find(d.root, "//DataSource")
	out := make([]DataSourceSummary, 0, len(nodes))
	for _, n := range nodes {
		s := DataSourceSummary{Name: attr(n, "Name")}
		if cp := child(n, "ConnectionProperties"); cp != nil {
			s.Provider = textOf(cp, "DataProvider")
			s.ConnectString = textOf(cp, "ConnectString")
		}
		// SecurityType and DataSourceID are rd: prefixed — match by local name.
		s.SecurityType = localNameText(n, "SecurityType")
		s.DataSourceID = localNameText(n, "DataSourceID")
		out = append(out, s)
	}
	return out
}

// ── DataSets ───────────────────────────────────────────────────────────────

// DataSetSummary describes one <DataSet> entry with its fields and query.
type DataSetSummary struct {
	Name        string         `json:"name"`
	DataSource  string         `json:"dataSource,omitempty"`
	CommandText string         `json:"commandText,omitempty"`
	Fields      []FieldSummary `json:"fields,omitempty"`
	FilterCount int            `json:"filterCount"`
}

// FieldSummary describes one <Field> in a DataSet.
type FieldSummary struct {
	Name      string `json:"name"`
	DataField string `json:"dataField,omitempty"`
}

// ListDataSets returns every DataSet declared in the report.
func (d *Document) ListDataSets() []DataSetSummary {
	nodes := xmlquery.Find(d.root, "//DataSet")
	out := make([]DataSetSummary, 0, len(nodes))
	for _, n := range nodes {
		s := DataSetSummary{Name: attr(n, "Name")}
		if q := child(n, "Query"); q != nil {
			s.DataSource = textOf(q, "DataSourceName")
			s.CommandText = textOf(q, "CommandText")
		}
		if f := child(n, "Fields"); f != nil {
			for _, fd := range xmlquery.Find(f, "Field") {
				s.Fields = append(s.Fields, FieldSummary{
					Name:      attr(fd, "Name"),
					DataField: textOf(fd, "DataField"),
				})
			}
		}
		if filters := child(n, "Filters"); filters != nil {
			s.FilterCount = len(xmlquery.Find(filters, "Filter"))
		}
		out = append(out, s)
	}
	return out
}

// ── Parameters ─────────────────────────────────────────────────────────────

// ParameterSummary describes one <ReportParameter> entry.
type ParameterSummary struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType,omitempty"`
	Nullable   bool   `json:"nullable"`
	AllowBlank bool   `json:"allowBlank"`
	MultiValue bool   `json:"multiValue"`
	Hidden     bool   `json:"hidden"`
	Prompt     string `json:"prompt,omitempty"`
	Default    string `json:"default,omitempty"`
}

// ListParameters returns every ReportParameter in the report.
func (d *Document) ListParameters() []ParameterSummary {
	nodes := xmlquery.Find(d.root, "//ReportParameter")
	out := make([]ParameterSummary, 0, len(nodes))
	for _, n := range nodes {
		s := ParameterSummary{
			Name:       attr(n, "Name"),
			DataType:   textOf(n, "DataType"),
			Nullable:   hasChild(n, "Nullable"),
			AllowBlank: hasChild(n, "AllowBlank"),
			MultiValue: hasChild(n, "MultiValue"),
			Hidden:     hasChild(n, "Hidden"),
			Prompt:     textOf(n, "Prompt"),
		}
		if dv := child(n, "DefaultValue"); dv != nil {
			// Default value lives at DefaultValue/Values/Value — use descendant axis.
			s.Default = strings.TrimSpace(descendantText(dv, "Value"))
		}
		out = append(out, s)
	}
	return out
}

// ── Tablixes ───────────────────────────────────────────────────────────────

// TablixSummary describes one <Tablix>: its shape and cell contents.
type TablixSummary struct {
	Name    string          `json:"name,omitempty"`
	DataSet string          `json:"dataSet,omitempty"`
	Columns []ColumnSummary `json:"columns,omitempty"`
	Rows    []RowSummary    `json:"rows,omitempty"`
}

// ColumnSummary is one <TablixColumn> width.
type ColumnSummary struct {
	Width string `json:"width,omitempty"`
}

// RowSummary is one <TablixRow> with its cells.
type RowSummary struct {
	Height string        `json:"height,omitempty"`
	Cells  []CellSummary `json:"cells,omitempty"`
}

// CellSummary is one <TablixCell>'s textbox content.
type CellSummary struct {
	Textbox string `json:"textbox,omitempty"`
	Value   string `json:"value"`           // always present so consumers can index safely
	Colspan int    `json:"colspan,omitempty"`
}

// ListTablixes returns every Tablix in the report.
func (d *Document) ListTablixes() []TablixSummary {
	nodes := xmlquery.Find(d.root, "//Tablix")
	out := make([]TablixSummary, 0, len(nodes))
	for _, n := range nodes {
		s := TablixSummary{Name: attr(n, "Name")}
		s.DataSet = textOf(n, "DataSetName")

		if body := child(n, "TablixBody"); body != nil {
			if cols := child(body, "TablixColumns"); cols != nil {
				for _, c := range xmlquery.Find(cols, "TablixColumn") {
					s.Columns = append(s.Columns, ColumnSummary{Width: textOf(c, "Width")})
				}
			}
			if rows := child(body, "TablixRows"); rows != nil {
				for _, r := range xmlquery.Find(rows, "TablixRow") {
					rs := RowSummary{Height: textOf(r, "Height")}
					// TablixCell is nested inside TablixCells — use descendant axis.
					for _, cell := range xmlquery.Find(r, ".//TablixCell") {
						cs := CellSummary{}
						if cc := child(cell, "CellContents"); cc != nil {
							if tb := child(cc, "Textbox"); tb != nil {
								cs.Textbox = attr(tb, "Name")
								cs.Value = strings.TrimSpace(descendantText(tb, "Value"))
							}
							if col := child(cc, "ColSpan"); col != nil {
								cs.Colspan = atoiSafe(col.InnerText())
							}
						}
						rs.Cells = append(rs.Cells, cs)
					}
					s.Rows = append(s.Rows, rs)
				}
			}
		}
		out = append(out, s)
	}
	return out
}

// ── Metadata ───────────────────────────────────────────────────────────────

// MetadataSummary is the report-level metadata that update_metadata touches.
// Returned by `rdl_get_metadata` so the agent can see what it can change.
type MetadataSummary struct {
	ReportID    string  `json:"reportId,omitempty"`
	Description string  `json:"description,omitempty"`
	Language    string  `json:"language,omitempty"`
	Author      string  `json:"author,omitempty"`
	PageWidth   string  `json:"pageWidth,omitempty"`
	PageHeight  string  `json:"pageHeight,omitempty"`
	Orientation string  `json:"orientation,omitempty"`
	Margins     Margins `json:"margins"`
}

// Margins holds the four page margin values.
type Margins struct {
	Left   string `json:"left,omitempty"`
	Right  string `json:"right,omitempty"`
	Top    string `json:"top,omitempty"`
	Bottom string `json:"bottom,omitempty"`
}

// GetMetadata returns the page-level metadata.
func (d *Document) GetMetadata() *MetadataSummary {
	m := &MetadataSummary{
		ReportID:    strings.TrimSpace(localNameText(d.root, "ReportID")),
		Description: textOf(d.root, "//Description"),
		Language:    textOf(d.root, "//Language"),
		Author:      textOf(d.root, "//Author"),
		PageWidth:   textOf(d.root, "//PageWidth"),
		PageHeight:  textOf(d.root, "//PageHeight"),
		Margins: Margins{
			Left:   textOf(d.root, "//LeftMargin"),
			Right:  textOf(d.root, "//RightMargin"),
			Top:    textOf(d.root, "//TopMargin"),
			Bottom: textOf(d.root, "//BottomMargin"),
		},
	}
	m.Orientation = orientationOf(m.PageWidth, m.PageHeight)
	return m
}

// ── Tiny tree helpers (local file, used by every inspect function) ─────────

// textOf returns the trimmed inner text of the first node matching expr.
// Empty string if not found.
func textOf(parent *xmlquery.Node, expr string) string {
	n := xmlquery.FindOne(parent, expr)
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.InnerText())
}

// descendantText finds the first descendant element with the given local name
// and returns its trimmed text. Use for nested-but-direct cases where bare
// XPath would only match direct children.
func descendantText(parent *xmlquery.Node, name string) string {
	if parent == nil {
		return ""
	}
	for _, n := range xmlquery.Find(parent, ".//"+name) {
		return strings.TrimSpace(n.InnerText())
	}
	return ""
}

// localNameText finds the first descendant whose local name matches (ignoring
// XML prefix). Use for rd: prefixed elements like ReportID, SecurityType,
// DataSourceID — xmlquery's bare XPath won't match prefixed names.
func localNameText(parent *xmlquery.Node, name string) string {
	n := findByLocalName(parent, name)
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.InnerText())
}

// findByLocalName returns the first descendant whose LocalName equals name.
func findByLocalName(parent *xmlquery.Node, name string) *xmlquery.Node {
	if parent == nil {
		return nil
	}
	for _, n := range xmlquery.Find(parent, fmt.Sprintf(".//*[local-name()='%s']", name)) {
		return n
	}
	return nil
}

func attr(n *xmlquery.Node, name string) string {
	if n == nil {
		return ""
	}
	return n.SelectAttr(name)
}

// child returns the first direct child element of n with the given name, or nil.
func child(parent *xmlquery.Node, name string) *xmlquery.Node {
	if parent == nil {
		return nil
	}
	return parent.SelectElement(name)
}

func hasChild(parent *xmlquery.Node, name string) bool {
	return child(parent, name) != nil
}

func countByPath(parent *xmlquery.Node, expr string) int {
	return len(xmlquery.Find(parent, expr))
}

// orientationOf infers "Portrait" or "Landscape" from page dimensions in cm.
// Returns "" if either dimension is missing or unparseable.
func orientationOf(width, height string) string {
	w := parseCm(width)
	h := parseCm(height)
	if w == 0 || h == 0 {
		return ""
	}
	if w > h {
		return "Landscape"
	}
	return "Portrait"
}

// parseCm parses a value like "29.7cm" into 29.7. Returns 0 on failure.
func parseCm(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "cm")
	return atofSafe(s)
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atofSafe(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
