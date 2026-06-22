package rdl

import (
	"strings"

	"github.com/antchfx/xmlquery"
)

// SwapMacros replaces strings within <ConnectString> elements.
// Returns the total number of occurrences replaced.
func SwapMacros(path string, pairs []RenamePair, dryRun bool) (int, error) {
	doc, err := Load(path)
	if err != nil {
		return 0, err
	}
	count := doc.SwapInElement("ConnectString", pairs)
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return 0, err
	}
	return count, nil
}

// SwapFields replaces Fields!X.Value references within <Value> elements.
// Returns the total number of occurrences replaced.
func SwapFields(path string, pairs []RenamePair, dryRun bool) (int, error) {
	doc, err := Load(path)
	if err != nil {
		return 0, err
	}
	expanded := make([]RenamePair, len(pairs))
	for i, p := range pairs {
		expanded[i] = RenamePair{
			Old: "Fields!" + p.Old + ".Value",
			New: "Fields!" + p.New + ".Value",
		}
	}
	count := doc.SwapInElement("Value", expanded)
	if _, err := maybeSave(doc, path, "", dryRun); err != nil {
		return 0, err
	}
	return count, nil
}

// SwapInElement replaces old→new within every <tag> element in the document.
// Returns the total number of replacements made.
func (d *Document) SwapInElement(tag string, pairs []RenamePair) int {
	total := 0
	for _, p := range pairs {
		for _, n := range xmlquery.Find(d.root, "//"+tag) {
			total += replaceInNodeText(n, p.Old, p.New)
		}
	}
	return total
}

// replaceInNodeText replaces old with new inside the text content of n.
// Walks all TextNode descendants. Returns the number of replacements.
func replaceInNodeText(n *xmlquery.Node, old, new string) int {
	count := 0
	for _, tn := range xmlquery.Find(n, ".//text()") {
		if tn.Type != xmlquery.TextNode {
			continue
		}
		c := strings.Count(tn.Data, old)
		if c > 0 {
			tn.Data = strings.ReplaceAll(tn.Data, old, new)
			count += c
		}
	}
	return count
}
