package rdl

import (
	"fmt"

	"github.com/antchfx/xmlquery"
)

// UpdateMetadata updates description, title, and/or page orientation in the RDL.
// Only fields set in the spec are touched; zero-value fields are left alone.
// When dryRun is true, no file is written.
//
// Title requires TitleTextbox to be set explicitly — there is no heuristic for
// finding "the title". Use rdl_list_tablixes or rdl_get_metadata to discover
// the textbox name, or rdl_inspect to see the report's PageHeader content.
//
// Orientation swaps page width/height using A4 dimensions (21cm × 29.7cm).
// It does NOT touch margins.
func UpdateMetadata(path string, spec MetadataUpdate, dryRun bool) (int, error) {
	doc, err := Load(path)
	if err != nil {
		return 0, MapLoadError(err, path)
	}
	count, err := doc.UpdateMetadata(spec)
	if err != nil {
		return 0, err
	}
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateMetadata applies the spec to the document in place. Returns the count
// of fields actually changed, or an error when a requested change could not apply.
func (d *Document) UpdateMetadata(spec MetadataUpdate) (int, error) {
	count := 0
	if spec.Description != "" {
		if setSimpleElementText(d.root, "Description", spec.Description) {
			count++
		}
	}
	if spec.Title != "" && spec.TitleTextbox != "" {
		if !d.setPageHeaderTextboxValue(spec.TitleTextbox, spec.Title) {
			return count, NewNotFoundError("Textbox", spec.TitleTextbox, nil)
		}
		count++
	}
	if spec.Orientation == "Portrait" || spec.Orientation == "Landscape" {
		applyOrientation(d.root, spec.Orientation)
		count++
	}
	return count, nil
}

// setSimpleElementText finds the first direct-child <tag> under <Report> and
// replaces its inner text. Returns false if the element doesn't exist.
func setSimpleElementText(doc *xmlquery.Node, tag, value string) bool {
	report := doc.FirstChild
	for report != nil && (report.Type != xmlquery.ElementNode || report.Data != "Report") {
		report = report.NextSibling
	}
	if report == nil {
		return false
	}
	for c := report.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode && c.Data == tag {
			setNodeText(c, value)
			return true
		}
	}
	return false
}

// setPageHeaderTextboxValue finds the Textbox with the given Name attribute
// inside <PageHeader> and replaces its first <Value> with newTitle.
// Returns false if the textbox or PageHeader is missing.
func (d *Document) setPageHeaderTextboxValue(textboxName, newTitle string) bool {
	// Find <Textbox Name="textboxName"> inside <PageHeader>.
	ph := xmlquery.FindOne(d.root, "//PageHeader")
	if ph == nil {
		return false
	}
	expr := fmt.Sprintf(".//Textbox[@Name=%q]", textboxName)
	tb := xmlquery.FindOne(ph, expr)
	if tb == nil {
		return false
	}
	val := xmlquery.FindOne(tb, ".//Value")
	if val != nil {
		setNodeText(val, newTitle)
		return true
	}
	// Value missing — create it.
	xmlquery.AddChild(tb, elementWithText("Value", newTitle))
	return true
}

// applyOrientation swaps <PageWidth> and <PageHeight> to match orientation,
// using A4 dimensions (21cm × 29.7cm).
func applyOrientation(doc *xmlquery.Node, orientation string) {
	var w, h string
	if orientation == "Landscape" {
		w, h = "29.7cm", "21cm"
	} else {
		w, h = "21cm", "29.7cm"
	}
	setSimpleElementText(doc, "PageWidth", w)
	setSimpleElementText(doc, "PageHeight", h)
}
