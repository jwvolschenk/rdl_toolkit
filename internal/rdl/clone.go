package rdl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/antchfx/xmlquery"
)

// Clone copies a source RDL to a target with a fresh ReportID.
// When dryRun is true, the target is not written; the returned UUID is the
// ID that would have been used.
func Clone(source, target string, dryRun bool) (string, error) {
	doc, err := Load(source)
	if err != nil {
		return "", fmt.Errorf("reading source: %w", err)
	}
	newID := newUUID()
	doc.SetReportID(newID)

	if dryRun {
		return newID, nil
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating target directory: %w", err)
	}
	if err := doc.Save(target); err != nil {
		return "", fmt.Errorf("writing target: %w", err)
	}
	return newID, nil
}

// SetReportID sets the <rd:ReportID> value, creating the element if missing.
func (d *Document) SetReportID(id string) {
	if n := findByLocalName(d.root, "ReportID"); n != nil {
		setNodeText(n, id)
		return
	}
	if report := d.reportRoot(); report != nil {
		appendIndented(report, elementWithText("rd:ReportID", id), depthOf(report))
	}
}

// ReportID returns the current <rd:ReportID> value, or empty if missing.
func (d *Document) ReportID() string {
	return localNameText(d.root, "ReportID")
}

// reportRoot returns the root <Report> element, or nil if missing.
func (d *Document) reportRoot() *xmlquery.Node {
	for n := d.root.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == xmlquery.ElementNode && n.Data == "Report" {
			return n
		}
	}
	return nil
}

// depthOf returns the depth of n from the document root, with the document
// root being depth -1 (so the root element is depth 0).
func depthOf(n *xmlquery.Node) int {
	d := -1
	for p := n; p != nil; p = p.Parent {
		d++
	}
	return d
}
