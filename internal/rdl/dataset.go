package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// DataSetOps bundles all DataSet mutations for a single Manage call.
// Operations are applied in this order: remove → rename → clearField →
// clearFilter → setCommandText → add → addField.
type DataSetOps struct {
	Remove         []string        `json:"remove,omitempty"`
	Rename         []RenamePair    `json:"rename,omitempty"`
	ClearFields    []string        `json:"clearFields,omitempty"`
	ClearFilters   []string        `json:"clearFilters,omitempty"`
	SetCommandText []CmdTextUpdate `json:"setCommandText,omitempty"`
	Add            []DataSetAdd    `json:"add,omitempty"`
	AddField       []FieldAdd      `json:"addField,omitempty"`
}

// ManageDataSets applies the operations to the file.
func ManageDataSets(path string, ops DataSetOps, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary := doc.ManageDataSets(ops)
	return maybeSave(doc, path, summary, dryRun)
}

// ManageDataSets applies ops to the document, returning a summary.
func (d *Document) ManageDataSets(ops DataSetOps) string {
	var b strings.Builder

	for _, name := range ops.Remove {
		n := d.findNamedElement("DataSet", name)
		if n == nil {
			fmt.Fprintf(&b, "Removed DataSet '%s' (not found)\n", name)
			continue
		}
		xmlquery.RemoveFromTree(n)
		fmt.Fprintf(&b, "Removed DataSet '%s'\n", name)
	}

	for _, r := range ops.Rename {
		n := d.findNamedElement("DataSet", r.Old)
		if n == nil {
			fmt.Fprintf(&b, "Renamed DataSet '%s' (not found)\n", r.Old)
			continue
		}
		n.SetAttr("Name", r.New)
		// Update <DataSetName> references in Tablixes etc.
		for _, ref := range xmlquery.Find(d.root, "//DataSetName") {
			if strings.TrimSpace(ref.InnerText()) == r.Old {
				setNodeText(ref, r.New)
			}
		}
		fmt.Fprintf(&b, "Renamed DataSet '%s' -> '%s'\n", r.Old, r.New)
	}

	for _, name := range ops.ClearFields {
		n := d.findNamedElement("DataSet", name)
		if n == nil {
			continue
		}
		c := clearChildElements(n, "Fields", "Field")
		fmt.Fprintf(&b, "Cleared %d field(s) from DataSet '%s'\n", c, name)
	}

	for _, name := range ops.ClearFilters {
		n := d.findNamedElement("DataSet", name)
		if n == nil {
			continue
		}
		if fields := child(n, "Filters"); fields != nil {
			xmlquery.RemoveFromTree(fields)
			fmt.Fprintf(&b, "Cleared filters from DataSet '%s'\n", name)
		}
	}

	for _, u := range ops.SetCommandText {
		n := d.findNamedElement("DataSet", u.DataSet)
		if n == nil {
			continue
		}
		if q := child(n, "Query"); q != nil {
			if setSimpleChildText(q, "CommandText", u.CmdText) {
				fmt.Fprintf(&b, "Updated CommandText for DataSet '%s'\n", u.DataSet)
			}
		}
	}

	for _, a := range ops.Add {
		if d.exists("DataSet", a.Name) {
			fmt.Fprintf(&b, "Added DataSet '%s' (already exists, skipped)\n", a.Name)
			continue
		}
		if err := d.addDataSet(a); err != nil {
			fmt.Fprintf(&b, "Add DataSet '%s' failed: %v\n", a.Name, err)
			continue
		}
		fmt.Fprintf(&b, "Added DataSet '%s'\n", a.Name)
	}

	for _, af := range ops.AddField {
		n := d.findNamedElement("DataSet", af.DataSet)
		if n == nil {
			fmt.Fprintf(&b, "Add field '%s' to '%s' (dataset not found)\n", af.Field, af.DataSet)
			continue
		}
		fields := child(n, "Fields")
		if fields == nil {
			fields = createElement("Fields")
			xmlquery.AddChild(n, fields)
		}
		f := createElement("Field", [2]string{"Name", af.Field})
		xmlquery.AddChild(f, elementWithText("DataField", af.Field))
		childIndent := detectChildIndent(fields)
		parentIndent := detectContainerIndent(fields)
		appendIndentedWithSuffix(fields, f, childIndent, parentIndent)
		fmt.Fprintf(&b, "Added field '%s' to DataSet '%s'\n", af.Field, af.DataSet)
	}

	return strings.TrimRight(b.String(), "\n")
}

// addDataSet constructs a <DataSet> element and appends it to <DataSets>.
func (d *Document) addDataSet(spec DataSetAdd) error {
	container := xmlquery.FindOne(d.root, "//DataSets")
	if container == nil {
		return fmt.Errorf("no <DataSets> element in document")
	}

	ds := createElement("DataSet", [2]string{"Name", spec.Name})
	query := createElement("Query")
	xmlquery.AddChild(query, elementWithText("DataSourceName", spec.DataSource))
	xmlquery.AddChild(query, elementWithText("CommandText", spec.CmdText))
	xmlquery.AddChild(ds, query)

	fields := createElement("Fields")
	fieldIndent := detectChildIndent(container) + "  "
	for _, f := range spec.Fields {
		fld := createElement("Field", [2]string{"Name", f})
		xmlquery.AddChild(fld, elementWithText("DataField", f))
		xmlquery.AddChild(fields, newTextNode("\n"+fieldIndent))
		xmlquery.AddChild(fields, fld)
	}
	xmlquery.AddChild(fields, newTextNode("\n"+detectChildIndent(container)))
	xmlquery.AddChild(ds, fields)

	childIndent := detectChildIndent(container)
	containerIndent := detectContainerIndent(container)
	appendIndentedWithSuffix(container, ds, childIndent, containerIndent)
	return nil
}

// clearChildElements removes all <child> elements from the <containerTag>
// child of parent. Returns the count removed. Container is left in place
// (empty <Container></Container>).
func clearChildElements(parent *xmlquery.Node, containerTag, childTag string) int {
	container := child(parent, containerTag)
	if container == nil {
		return 0
	}
	count := 0
	// Snapshot children — we'll mutate during iteration.
	var toRemove []*xmlquery.Node
	for c := container.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode && c.Data == childTag {
			toRemove = append(toRemove, c)
		}
	}
	for _, n := range toRemove {
		xmlquery.RemoveFromTree(n)
		count++
	}
	return count
}
