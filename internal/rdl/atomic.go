package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// MutationOutcome is the structured result of a single atomic mutation.
type MutationOutcome struct {
	Action  string `json:"action"`
	Name    string `json:"name,omitempty"`
	OldName string `json:"oldName,omitempty"`
	NewName string `json:"newName,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// ── DataSource atomic ops ───────────────────────────────────────────────────

func AddDataSource(path string, spec DataSourceAdd, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.AddDataSourceOp(spec)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) AddDataSourceOp(spec DataSourceAdd) (MutationOutcome, error) {
	if d.exists("DataSource", spec.Name) {
		return MutationOutcome{Action: "add", Name: spec.Name, Skipped: true}, nil
	}
	if err := d.addDataSource(spec); err != nil {
		return MutationOutcome{}, err
	}
	return MutationOutcome{Action: "added", Name: spec.Name}, nil
}

func RemoveDataSource(path, name string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.RemoveDataSourceOp(name)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) RemoveDataSourceOp(name string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSource", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSource", name, d.dataSourceNames())
	}
	xmlquery.RemoveFromTree(n)
	return MutationOutcome{Action: "removed", Name: name}, nil
}

func RenameDataSource(path, old, new string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.RenameDataSourceOp(old, new)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) RenameDataSourceOp(old, new string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSource", old)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSource", old, d.dataSourceNames())
	}
	n.SetAttr("Name", new)
	for _, ref := range xmlquery.Find(d.root, "//DataSourceName") {
		if strings.TrimSpace(ref.InnerText()) == old {
			setNodeText(ref, new)
		}
	}
	return MutationOutcome{Action: "renamed", OldName: old, NewName: new}, nil
}

func SetDataSourceConnectString(path, name, connectString string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.SetDataSourceConnectStringOp(name, connectString)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) SetDataSourceConnectStringOp(name, connectString string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSource", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSource", name, d.dataSourceNames())
	}
	cp := child(n, "ConnectionProperties")
	if cp == nil {
		return MutationOutcome{}, fmt.Errorf("DataSource %q has no ConnectionProperties", name)
	}
	if !setSimpleChildText(cp, "ConnectString", connectString) {
		return MutationOutcome{}, fmt.Errorf("DataSource %q has no ConnectString element", name)
	}
	return MutationOutcome{Action: "updated", Name: name}, nil
}

// ── DataSet atomic ops ──────────────────────────────────────────────────────

func AddDataSet(path string, spec DataSetAdd, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.AddDataSetOp(spec)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) AddDataSetOp(spec DataSetAdd) (MutationOutcome, error) {
	if d.exists("DataSet", spec.Name) {
		return MutationOutcome{Action: "add", Name: spec.Name, Skipped: true}, nil
	}
	if err := d.addDataSet(spec); err != nil {
		return MutationOutcome{}, err
	}
	return MutationOutcome{Action: "added", Name: spec.Name}, nil
}

func RemoveDataSet(path, name string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.RemoveDataSetOp(name)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) RemoveDataSetOp(name string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", name, d.dataSetNames())
	}
	xmlquery.RemoveFromTree(n)
	return MutationOutcome{Action: "removed", Name: name}, nil
}

func RenameDataSet(path, old, new string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.RenameDataSetOp(old, new)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) RenameDataSetOp(old, new string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", old)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", old, d.dataSetNames())
	}
	n.SetAttr("Name", new)
	for _, ref := range xmlquery.Find(d.root, "//DataSetName") {
		if strings.TrimSpace(ref.InnerText()) == old {
			setNodeText(ref, new)
		}
	}
	return MutationOutcome{Action: "renamed", OldName: old, NewName: new}, nil
}

func AddDataSetField(path, dataset, field string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.AddDataSetFieldOp(dataset, field)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) AddDataSetFieldOp(dataset, field string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", dataset)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", dataset, d.dataSetNames())
	}
	fields := child(n, "Fields")
	if fields == nil {
		fields = createElement("Fields")
		xmlquery.AddChild(n, fields)
	}
	f := createElement("Field", [2]string{"Name", field})
	xmlquery.AddChild(f, elementWithText("DataField", field))
	childIndent := detectChildIndent(fields)
	parentIndent := detectContainerIndent(fields)
	appendIndentedWithSuffix(fields, f, childIndent, parentIndent)
	return MutationOutcome{Action: "field_added", Name: field, NewName: dataset}, nil
}

func ClearDataSetFields(path, name string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.ClearDataSetFieldsOp(name)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) ClearDataSetFieldsOp(name string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", name, d.dataSetNames())
	}
	count := clearChildElements(n, "Fields", "Field")
	return MutationOutcome{Action: "fields_cleared", Name: name, Count: count}, nil
}

func ClearDataSetFilters(path, name string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.ClearDataSetFiltersOp(name)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) ClearDataSetFiltersOp(name string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", name, d.dataSetNames())
	}
	if filters := child(n, "Filters"); filters != nil {
		xmlquery.RemoveFromTree(filters)
		return MutationOutcome{Action: "filters_cleared", Name: name}, nil
	}
	return MutationOutcome{Action: "filters_cleared", Name: name, Skipped: true}, nil
}

func SetDataSetCommandText(path, dataset, cmdText string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.SetDataSetCommandTextOp(dataset, cmdText)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) SetDataSetCommandTextOp(dataset, cmdText string) (MutationOutcome, error) {
	n := d.findNamedElement("DataSet", dataset)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("DataSet", dataset, d.dataSetNames())
	}
	q := child(n, "Query")
	if q == nil {
		return MutationOutcome{}, fmt.Errorf("DataSet %q has no Query element", dataset)
	}
	if !setSimpleChildText(q, "CommandText", cmdText) {
		return MutationOutcome{}, fmt.Errorf("DataSet %q has no CommandText element", dataset)
	}
	return MutationOutcome{Action: "command_updated", Name: dataset}, nil
}

// ── Parameter atomic ops ─────────────────────────────────────────────────────

func AddParameter(path string, spec ParameterAdd, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.AddParameterOp(spec)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) AddParameterOp(spec ParameterAdd) (MutationOutcome, error) {
	if d.exists("ReportParameter", spec.Name) {
		return MutationOutcome{Action: "add", Name: spec.Name, Skipped: true}, nil
	}
	container := xmlquery.FindOne(d.root, "//ReportParameters")
	if container == nil {
		return MutationOutcome{}, fmt.Errorf("no <ReportParameters> element in document")
	}
	d.addParameter(spec)
	return MutationOutcome{Action: "added", Name: spec.Name}, nil
}

func RemoveParameter(path, name string, dryRun bool) (MutationOutcome, error) {
	doc, err := Load(path)
	if err != nil {
		return MutationOutcome{}, MapLoadError(err, path)
	}
	out, err := doc.RemoveParameterOp(name)
	if err != nil {
		return MutationOutcome{}, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return MutationOutcome{}, err
	}
	return out, nil
}

func (d *Document) RemoveParameterOp(name string) (MutationOutcome, error) {
	n := d.findNamedElement("ReportParameter", name)
	if n == nil {
		return MutationOutcome{}, NewNotFoundError("ReportParameter", name, d.parameterNames())
	}
	xmlquery.RemoveFromTree(n)
	layoutCount := 0
	for _, ref := range xmlquery.Find(d.root, "//CellDefinition") {
		pn := child(ref, "ParameterName")
		if pn != nil && strings.TrimSpace(pn.InnerText()) == name {
			xmlquery.RemoveFromTree(ref)
			layoutCount++
		}
	}
	d.compactParameterGrid()
	var b strings.Builder
	d.sanitizeParameterGrid(&b)
	return MutationOutcome{Action: "removed", Name: name, Count: layoutCount}, nil
}
