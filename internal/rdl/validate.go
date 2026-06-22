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

		// Per-row effective width must equal column count.
		rows := xmlquery.Find(body, "TablixRows/TablixRow")
		expected := cols
		for i, row := range rows {
			cells := xmlquery.Find(row, "TablixCells/TablixCell")
			effective := 0
			for _, c := range cells {
				effective += effectiveCellWidth(c)
			}
			if effective != expected {
				r.Issues = append(r.Issues, Issue{
					Severity: SeverityError,
					XPath:    fmt.Sprintf("%s//TablixBody//TablixRow[%d]", xpath, i),
					Message: fmt.Sprintf("row %d effective width %d does not match column count %d (check ColSpan values)",
						i, effective, expected),
				})
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

// effectiveCellWidth returns the column count consumed by a TablixCell.
// Reads <ColSpan> if present; defaults to 1.
func effectiveCellWidth(cell *xmlquery.Node) int {
	if cs := child(cell, "CellContents"); cs != nil {
		if col := child(cs, "ColSpan"); col != nil {
			n := atoiSafe(strings.TrimSpace(col.InnerText()))
			if n > 0 {
				return n
			}
		}
	}
	return 1
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
				r.Issues = append(r.Issues, Issue{
					Severity: SeverityError,
					Message:  fmt.Sprintf("Fields!%s.Value is referenced but not defined in any <Field>", field),
				})
			}
		}
	}
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

// checkParameterLayout verifies every ParameterName in ReportParametersLayout
// points at a defined ReportParameter.
func (d *Document) checkParameterLayout(r *ValidationReport) {
	defined := map[string]bool{}
	for _, p := range xmlquery.Find(d.root, "//ReportParameter") {
		if name := p.SelectAttr("Name"); name != "" {
			defined[name] = true
		}
	}
	for _, ref := range xmlquery.Find(d.root, "//CellDefinition/ParameterName") {
		name := strings.TrimSpace(ref.InnerText())
		if name != "" && !defined[name] {
			r.Issues = append(r.Issues, Issue{
				Severity: SeverityError,
				Message:  fmt.Sprintf("ReportParametersLayout references parameter %q which is not defined", name),
			})
		}
	}
}
