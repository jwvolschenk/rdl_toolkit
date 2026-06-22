package rdl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/antchfx/xmlquery"
)

// Register adds an RDL file entry to a .rptproj project file, alphabetically
// sorted among existing <Report Include="..." /> entries.
func Register(rdlPath, projPath string) (string, error) {
	rdlName := filepath.Base(rdlPath)

	projData, err := os.ReadFile(projPath)
	if err != nil {
		return "", fmt.Errorf("reading project file: %w", err)
	}
	root, err := xmlquery.Parse(bytes.NewReader(projData))
	if err != nil {
		return "", fmt.Errorf("parsing project file: %w", err)
	}

	// Idempotency: skip if already registered.
	if n := xmlquery.FindOne(root, fmt.Sprintf(`//Report[@Include=%q]`, rdlName)); n != nil {
		return fmt.Sprintf("'%s' is already registered in %s", rdlName, projPath), nil
	}

	// Find an existing <Report Include="..." /> entry to insert alongside.
	existing := xmlquery.Find(root, "//Report")
	existingNames := make([]string, 0, len(existing))
	for _, n := range existing {
		if v := n.SelectAttr("Include"); v != "" {
			existingNames = append(existingNames, v)
		}
	}
	sort.Strings(existingNames)

	newNode := createElement("Report", [2]string{"Include", rdlName})

	if len(existingNames) > 0 {
		// Insert in alphabetical position.
		var anchorName string
		var before bool
		for _, name := range existingNames {
			if rdlName < name {
				anchorName = name
				before = true
				break
			}
		}
		if anchorName == "" {
			// Goes after the last existing entry.
			anchorName = existingNames[len(existingNames)-1]
			before = false
		}
		anchor := xmlquery.FindOne(root, fmt.Sprintf(`//Report[@Include=%q]`, anchorName))
		if anchor == nil {
			return "", fmt.Errorf("internal: anchor %q not found", anchorName)
		}
		if before {
			insertBefore(anchor, newNode)
		} else {
			insertAfter(anchor, newNode)
		}
	} else {
		// No existing reports — insert inside the last <ItemGroup>.
		itemGroups := xmlquery.Find(root, "//ItemGroup")
		if len(itemGroups) == 0 {
			return "", fmt.Errorf("no <ItemGroup> found in project file")
		}
		last := itemGroups[len(itemGroups)-1]
		appendIndented(last, newNode, depthOf(last))
	}

	var buf bytes.Buffer
	if err := root.WriteWithOptions(&buf, xmlquery.WithEmptyTagSupport()); err != nil {
		return "", fmt.Errorf("serialising project file: %w", err)
	}
	if err := os.WriteFile(projPath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("writing project file: %w", err)
	}
	return fmt.Sprintf("Registered '%s' in %s (alphabetically)", rdlName, projPath), nil
}

// insertBefore inserts newNode as the previous sibling of anchor, copying
// the indent text that precedes anchor.
func insertBefore(anchor, newNode *xmlquery.Node) {
	prev := anchor.PrevSibling
	indent := ""
	if prev != nil && prev.Type == xmlquery.TextNode {
		indent = prev.Data
	}
	if indent != "" {
		xmlquery.AddSibling(anchor, newTextNode(indent))
	}
	xmlquery.AddSibling(anchor, newNode)
}

// insertAfter inserts newNode as the next sibling of anchor, copying the
// indent text that follows anchor.
func insertAfter(anchor, newNode *xmlquery.Node) {
	next := anchor.NextSibling
	indent := ""
	if next != nil && next.Type == xmlquery.TextNode {
		indent = next.Data
	}
	if indent != "" {
		xmlquery.AddImmediateSibling(anchor, newTextNode(indent))
	}
	xmlquery.AddImmediateSibling(anchor, newNode)
}
