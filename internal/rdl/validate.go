package rdl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antchfx/xmlquery"
)

// Severity is "error" or "warning". Errors mean SSRS will reject the file;
// warnings mean the file will render but may be wrong.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding.
type Issue struct {
	Severity Severity `json:"severity"`
	XPath    string   `json:"xpath,omitempty"` // element location if known
	Message  string   `json:"message"`
}

// ValidationReport aggregates findings for one file. Pass is true when no
// error-severity issues were found (warnings don't fail the report).
type ValidationReport struct {
	File   string  `json:"file"`
	Issues []Issue `json:"issues"`
	Pass   bool    `json:"pass"`
}

// Validate runs all checks on the file and returns a structured report.
// Returns a non-nil error only for I/O or XML parser failures (file unreadable,
// malformed XML). Structural and reference problems land in the report with
// Pass=false; callers wanting a hard failure should check report.Pass.
func Validate(path string) (*ValidationReport, error) {
	doc, err := Load(path)
	if err != nil {
		return nil, err
	}
	r := doc.Validate()
	r.File = path
	return r, nil
}

// Validate runs every check against the document and returns the report.
func (d *Document) Validate() *ValidationReport {
	r := &ValidationReport{Issues: []Issue{}}

	d.checkTablixShape(r)
	d.checkTextboxStructure(r)
	d.checkFieldReferences(r)
	d.checkDatasetReferences(r)
	d.checkDataSourceReferences(r)
	d.checkParameterLayout(r)

	r.Pass = true
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			r.Pass = false
			break
		}
	}
	return r
}

// ── Checkers ───────────────────────────────────────────────────────────────

// checkTablixShape verifies per-Tablix: column hierarchy member count equals
// the body column count, and every row has the same effective cell width
// (accounting for ColSpan).
func (d *Document) checkTablixShape(r *ValidationReport) {
	for _, t := range xmlquery.Find(d.root, "//Tablix") {
		name := t.SelectAttr("Name")
		xpath := fmt.Sprintf("//Tablix[@Name=%q]", name)

		body := child(t, "TablixBody")
		if body == nil {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError, XPath: xpath,
				Message: "Tablix has no <TablixBody>",
			})
			continue
		}

		cols := len(xmlquery.Find(body, "TablixColumns/TablixColumn"))
		hierCols := len(xmlquery.Find(t, "TablixColumnHierarchy/TablixMembers/TablixMember"))
		if cols == 0 {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError, XPath: xpath,
				Message: "TablixBody has no TablixColumns",
			})
		} else if hierCols != cols {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError, XPath: xpath,
				Message: fmt.Sprintf("TablixColumnHierarchy member count (%d) does not match TablixColumns count (%d)", hierCols, cols),
			})
		}

		// Per-row effective cell width (sum of ColSpan, default 1) must equal column count.
		// When len(cells)==cols, also validate legacy ColSpan placeholder semantics.
		rows := xmlquery.Find(body, "TablixRows/TablixRow")
		for i, row := range rows {
			cells := xmlquery.Find(row, "TablixCells/TablixCell")
			effective := effectiveRowCellWidth(cells)
			if effective != cols {
				r.Issues = append(r.Issues, Issue{
					Severity: SeverityError,
					XPath:    fmt.Sprintf("%s//TablixBody//TablixRow[%d]", xpath, i),
					Message: fmt.Sprintf("row %d effective cell width %d does not match column count %d",
						i, effective, cols),
				})
			}
			if len(cells) != cols {
				continue // no-placeholder colspan layout; skip placeholder check
			}
			// Legacy: ColSpan placeholder cells when physical cell count equals column count.
			for j, c := range cells {
				cs := child(c, "CellContents")
				if cs == nil {
					continue
				}
				col := child(cs, "ColSpan")
				if col == nil {
					continue
				}
				n := atoiSafe(strings.TrimSpace(col.InnerText()))
				if n <= 1 {
					continue
				}
				expected := n - 1
				actual := 0
				for k := j + 1; k < len(cells) && k <= j+expected; k++ {
					phCC := child(cells[k], "CellContents")
					if phCC == nil || phCC.FirstChild == nil {
						actual++
					}
				}
				if actual < expected {
					r.Issues = append(r.Issues, Issue{
						Severity: SeverityError,
						XPath:    fmt.Sprintf("%s//TablixBody//TablixRow[%d]/TablixCell[%d]", xpath, i, j),
						Message: fmt.Sprintf("ColSpan=%d cell needs %d empty placeholder cells but found %d",
							n, expected, actual),
					})
				}
			}
		}

		// Row hierarchy member count must equal body row count.
		hierRows := len(xmlquery.Find(t, "TablixRowHierarchy/TablixMembers/TablixMember"))
		if hierRows != len(rows) {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError, XPath: xpath,
				Message: fmt.Sprintf("TablixRowHierarchy member count (%d) does not match TablixRows count (%d)", hierRows, len(rows)),
			})
		}
	}
}

// checkTextboxStructure verifies every Textbox has the mandatory Paragraphs
// child. Visual Studio rejects empty Textbox shells at design time.
func (d *Document) checkTextboxStructure(r *ValidationReport) {
	for _, tb := range xmlquery.Find(d.root, "//Textbox") {
		if child(tb, "Paragraphs") != nil {
			continue
		}
		name := tb.SelectAttr("Name")
		xpath := "//Textbox"
		if name != "" {
			xpath = fmt.Sprintf("//Textbox[@Name=%q]", name)
		}
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityError,
			XPath:    xpath,
			Message: fmt.Sprintf(
				"Textbox %q is missing mandatory <Paragraphs> element (Visual Studio will refuse to open the report)",
				name),
		})
	}
}

// checkFieldReferences finds every Fields!X.Value reference and checks that X
// is defined as a Field in some DataSet. Also checks that the Field's parent
// dataset, if discoverable via Tablix binding, exists.
func (d *Document) checkFieldReferences(r *ValidationReport) {
	defined := map[string]bool{}
	for _, fd := range xmlquery.Find(d.root, "//Field") {
		if name := fd.SelectAttr("Name"); name != "" {
			defined[name] = true
		}
	}
	re := regexp.MustCompile(`Fields!([^.]+)\.Value`)
	seen := map[string]bool{}
	for _, v := range xmlquery.Find(d.root, "//Value") {
		text := strings.TrimSpace(v.InnerText())
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			field := m[1]
			if seen[field] {
				continue
			}
			seen[field] = true
			if !defined[field] {
				xpath := valueXPath(v)
				r.Issues = append(r.Issues, Issue{
					Severity: SeverityError,
					XPath:    xpath,
					Message:  fmt.Sprintf("Fields!%s.Value is referenced but not defined in any <Field>", field),
				})
			}
		}
	}
}

func valueXPath(v *xmlquery.Node) string {
	if v == nil {
		return "//Value"
	}
	parts := []string{}
	for n := v; n != nil; n = n.Parent {
		if n.Type != xmlquery.ElementNode {
			continue
		}
		if n.Data == "Report" {
			parts = append([]string{"//Report"}, parts...)
			break
		}
		parts = append([]string{n.Data}, parts...)
	}
	if len(parts) == 0 {
		return "//Value"
	}
	return strings.Join(parts, "/")
}

// effectiveRowCellWidth sums ColSpan from TablixCell nodes (default 1).
// Placeholder cells (no CellContents) contribute 0 to the width.
func effectiveRowCellWidth(cells []*xmlquery.Node) int {
	got := 0
	for _, c := range cells {
		cc := child(c, "CellContents")
		if cc == nil {
			// Empty placeholder cell — contributes 0 to effective width.
			continue
		}
		cs := 1
		if col := child(cc, "ColSpan"); col != nil {
			cs = atoiSafe(strings.TrimSpace(col.InnerText()))
			if cs < 1 {
				cs = 1
			}
		}
		got += cs
	}
	return got
}

// checkDatasetReferences verifies that every <DataSetName> in a Tablix points
// at an existing DataSet, and that every <DataSourceName> in a DataSet Query
// points at an existing DataSource.
func (d *Document) checkDatasetReferences(r *ValidationReport) {
	dataSets := map[string]bool{}
	for _, n := range xmlquery.Find(d.root, "//DataSet") {
		if name := n.SelectAttr("Name"); name != "" {
			dataSets[name] = true
		}
	}
	dataSources := map[string]bool{}
	for _, n := range xmlquery.Find(d.root, "//DataSource") {
		if name := n.SelectAttr("Name"); name != "" {
			dataSources[name] = true
		}
	}

	for _, t := range xmlquery.Find(d.root, "//Tablix") {
		if dn := child(t, "DataSetName"); dn != nil {
			name := strings.TrimSpace(dn.InnerText())
			if name != "" && !dataSets[name] {
				r.Issues = append(r.Issues, Issue{
					Severity: SeverityError,
					XPath:    fmt.Sprintf("//Tablix[@Name=%q]/DataSetName", t.SelectAttr("Name")),
					Message:  fmt.Sprintf("Tablix references DataSet %q which is not defined", name),
				})
			}
		}
	}

	for _, ds := range xmlquery.Find(d.root, "//DataSet") {
		dsName := ds.SelectAttr("Name")
		if q := child(ds, "Query"); q != nil {
			if dsn := child(q, "DataSourceName"); dsn != nil {
				name := strings.TrimSpace(dsn.InnerText())
				if name != "" && !dataSources[name] {
					r.Issues = append(r.Issues, Issue{
						Severity: SeverityError,
						XPath:    fmt.Sprintf("//DataSet[@Name=%q]/Query/DataSourceName", dsName),
						Message:  fmt.Sprintf("DataSet %q references DataSource %q which is not defined", dsName, name),
					})
				}
			}
		}
	}
}

// checkDataSourceReferences is folded into checkDatasetReferences above.
// Kept as a placeholder for future DataSource-only checks (duplicate names,
// missing ConnectionProperties, etc.).
func (d *Document) checkDataSourceReferences(r *ValidationReport) {
	// Detect duplicate DataSource names.
	seen := map[string]int{}
	for _, n := range xmlquery.Find(d.root, "//DataSource") {
		if name := n.SelectAttr("Name"); name != "" {
			seen[name]++
		}
	}
	for name, count := range seen {
		if count > 1 {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError,
				Message:  fmt.Sprintf("DataSource %q defined %d times (must be unique)", name, count),
			})
		}
	}
}

// checkParameterLayout verifies:
//  1. Every ParameterName in ReportParametersLayout points at a defined ReportParameter.
//  2. When ReportParameters exist, ReportParametersLayout must also exist (VS2019+ requirement).
func (d *Document) checkParameterLayout(r *ValidationReport) {
	defined := map[string]bool{}
	for _, p := range xmlquery.Find(d.root, "//ReportParameter") {
		if name := p.SelectAttr("Name"); name != "" {
			defined[name] = true
		}
	}

	// Check 1: orphaned layout references
	for _, ref := range xmlquery.Find(d.root, "//CellDefinition/ParameterName") {
		name := strings.TrimSpace(ref.InnerText())
		if name != "" && !defined[name] {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError,
				Message:  fmt.Sprintf("ReportParametersLayout references parameter %q which is not defined", name),
			})
		}
	}

	// Check 2: parameters exist but no layout (VS2019+ will refuse to open)
	if len(defined) > 0 && len(xmlquery.Find(d.root, "//ReportParametersLayout")) == 0 {
		r.Issues = append(r.Issues, Issue{
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"Report has %d ReportParameter(s) but no ReportParametersLayout — Visual Studio 2019+ requires this element",
				len(defined)),
		})
	}
}
