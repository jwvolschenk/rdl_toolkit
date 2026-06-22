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

	for _, a := range ops.Add {
		if d.exists("ReportParameter", a.Name) {
			fmt.Fprintf(&b, "Added parameter '%s' (already exists, skipped)\n", a.Name)
			continue
		}
		d.addParameter(a)
		fmt.Fprintf(&b, "Added parameter '%s' (type=%s)\n", a.Name, a.Type)
	}

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

	xmlquery.AddChild(container, newTextNode("\n  "))
	xmlquery.AddChild(container, p)
	xmlquery.AddChild(container, newTextNode("\n  "))
}
