package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// ApplyTheme copies visual theming from a source report to a target report.
// This includes: HeaderTheme shared dataset, PageHeader, PageFooter, margins,
// and page dimensions. Data sources, datasets, parameters, and tablix are NOT touched.
//
// If the target already has a HeaderTheme dataset, it is replaced.
// If the target already has a PageHeader/PageFooter, it is replaced.
// Parameters pvc_Theme and vc_ReportPack are added if missing.
func ApplyTheme(source, target string, dryRun bool) (string, error) {
	srcDoc, err := Load(source)
	if err != nil {
		return "", fmt.Errorf("reading source: %w", err)
	}
	tgtDoc, err := Load(target)
	if err != nil {
		return "", fmt.Errorf("reading target: %w", err)
	}

	var changes []string

	// 1. Copy HeaderTheme DataSet.
	if copied := copyHeaderTheme(srcDoc, tgtDoc); copied {
		changes = append(changes, "HeaderTheme dataset")
	}

	// 2. Copy PageHeader.
	if copied := copyPageSection(srcDoc, tgtDoc, "PageHeader"); copied {
		changes = append(changes, "PageHeader")
	}

	// 3. Copy PageFooter (if source has one).
	if copied := copyPageSection(srcDoc, tgtDoc, "PageFooter"); copied {
		changes = append(changes, "PageFooter")
	}

	// 4. Copy page dimensions and margins.
	if copied := copyPageDimensions(srcDoc, tgtDoc); copied {
		changes = append(changes, "page dimensions/margins")
	}

	// 5. Ensure pvc_Theme and vc_ReportPack parameters exist.
	for _, p := range []ParameterAdd{
		{Name: "pvc_Theme", Type: "String", Prompt: "Theme"},
		{Name: "vc_ReportPack", Type: "String", Hidden: true},
	} {
		if !tgtDoc.exists("ReportParameter", p.Name) {
			tgtDoc.addParameter(p)
			changes = append(changes, fmt.Sprintf("parameter %s", p.Name))
		}
	}

	// 6. Regenerate parameter layout after adding parameters.
	var log strings.Builder
	tgtDoc.sanitizeParameterGrid(&log)
	tgtDoc.ensureParameterLayout(&log)

	if len(changes) == 0 {
		return "No theme elements found in source", nil
	}

	summary := fmt.Sprintf("Applied theme from %s: %s", source, strings.Join(changes, ", "))

	if dryRun {
		return summary, nil
	}
	if err := tgtDoc.Save(target); err != nil {
		return "", fmt.Errorf("writing target: %w", err)
	}
	return summary, nil
}

// copyHeaderTheme copies the HeaderTheme shared dataset from source to target.
// Returns true if a HeaderTheme was found and copied.
func copyHeaderTheme(src, tgt *Document) bool {
	srcDS := xmlquery.FindOne(src.root, "//DataSet[@Name='HeaderTheme']")
	if srcDS == nil {
		return false
	}

	// Remove existing HeaderTheme in target (if any).
	if existing := xmlquery.FindOne(tgt.root, "//DataSet[@Name='HeaderTheme']"); existing != nil {
		xmlquery.RemoveFromTree(existing)
	}

	// Serialize from source, parse into target context.
	xml := srcDS.OutputXML(true)
	fragment, err := parseFragment(xml)
	if err != nil {
		return false
	}

	// Insert into target's DataSets container.
	container := xmlquery.FindOne(tgt.root, "//DataSets")
	if container == nil {
		report := tgt.reportRoot()
		if report == nil {
			return false
		}
		container = createElement("DataSets")
		// Insert before ReportSections.
		sections := findByLocalName(report, "ReportSections")
		if sections != nil {
			insertNodeBefore(container, sections)
		} else {
			appendIndented(report, container, depthOf(report))
		}
	}
	appendIndented(container, fragment, depthOf(container)+1)
	return true
}

// copyPageSection copies a PageHeader or PageFooter from source to target.
// Returns true if the section was found and copied.
func copyPageSection(src, tgt *Document, sectionName string) bool {
	srcSection := findByLocalName(src.root, sectionName)
	if srcSection == nil {
		return false
	}

	// Serialize from source.
	xml := srcSection.OutputXML(true)
	fragment, err := parseFragment(xml)
	if err != nil {
		return false
	}

	// Find target's Page element.
	tgtPage := findByLocalName(tgt.root, "Page")
	if tgtPage == nil {
		return false
	}

	// Remove existing section in target (if any).
	if existing := findByLocalName(tgtPage, sectionName); existing != nil {
		xmlquery.RemoveFromTree(existing)
	}

	// Insert before PageHeight (or PageWidth if no PageHeight).
	ref := findByLocalName(tgtPage, "PageHeight")
	if ref == nil {
		ref = findByLocalName(tgtPage, "PageWidth")
	}
	if ref != nil {
		insertNodeBefore(fragment, ref)
	} else {
		appendIndented(tgtPage, fragment, depthOf(tgtPage)+1)
	}
	return true
}

// copyPageDimensions copies page size and margins from source to target.
// Returns true if any dimension was changed.
func copyPageDimensions(src, tgt *Document) bool {
	srcPage := findByLocalName(src.root, "Page")
	tgtPage := findByLocalName(tgt.root, "Page")
	if srcPage == nil || tgtPage == nil {
		return false
	}

	dims := []string{"PageHeight", "PageWidth", "LeftMargin", "RightMargin", "TopMargin", "BottomMargin"}
	changed := false
	for _, dim := range dims {
		srcVal := localNameText(srcPage, dim)
		if srcVal == "" {
			continue
		}
		tgtNode := findByLocalName(tgtPage, dim)
		if tgtNode == nil {
			appendIndented(tgtPage, elementWithText(dim, srcVal), depthOf(tgtPage)+1)
			changed = true
		} else if strings.TrimSpace(tgtNode.InnerText()) != srcVal {
			setNodeText(tgtNode, srcVal)
			changed = true
		}
	}
	return changed
}

// insertNodeBefore inserts newNode immediately before refNode in the tree.
// Preserves any preceding whitespace text node.
func insertNodeBefore(newNode, refNode *xmlquery.Node) {
	parent := refNode.Parent
	if parent == nil {
		return
	}
	// If refNode is the first child, make newNode the first child.
	if parent.FirstChild == refNode {
		newNode.Parent = parent
		newNode.NextSibling = parent.FirstChild
		parent.FirstChild.PrevSibling = newNode
		parent.FirstChild = newNode
		return
	}
	// Otherwise, insert after refNode's previous sibling.
	xmlquery.AddImmediateSibling(refNode.PrevSibling, newNode)
}
