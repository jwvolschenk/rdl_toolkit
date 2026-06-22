package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/google/uuid"
)

// This file holds shared tree-mutation helpers used by every ported mutation
// file (clone, datasource, dataset, parameter, metadata, swap, register).
// They hide xmlquery's raw node construction behind a small, named API.

// ── Spec structs ───────────────────────────────────────────────────────────

// DataSourceAdd describes a DataSource to add. SecurityType defaults to "None"
// when empty (matching the most common RDL pattern).
type DataSourceAdd struct {
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ConnectString string `json:"connectString"`
	SecurityType  string `json:"securityType,omitempty"`
}

// DataSetAdd describes a DataSet to add.
type DataSetAdd struct {
	Name       string   `json:"name"`
	DataSource string   `json:"datasource"`
	CmdText    string   `json:"cmdText"`
	Fields     []string `json:"fields,omitempty"`
}

// ParameterAdd describes a ReportParameter to add. Prompt defaults to Name
// when empty (SSRS requires a Prompt element).
type ParameterAdd struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default string `json:"default,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
}

// RenamePair is an old→new name pair for rename operations.
type RenamePair struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// ConnectStringUpdate sets a DataSource's ConnectString to a new value.
type ConnectStringUpdate struct {
	Name          string `json:"name"`
	ConnectString string `json:"connectString"`
}

// FieldAdd adds a single Field to a DataSet.
type FieldAdd struct {
	DataSet string `json:"dataset"`
	Field   string `json:"field"`
}

// CmdTextUpdate sets a DataSet's CommandText.
type CmdTextUpdate struct {
	DataSet string `json:"dataset"`
	CmdText string `json:"cmdText"`
}

// MetadataUpdate is the set of metadata fields that may be changed.
// Any field left zero is left untouched in the document.
type MetadataUpdate struct {
	Description  string `json:"description,omitempty"`
	Title        string `json:"title,omitempty"`
	TitleTextbox string `json:"titleTextbox,omitempty"` // explicit textbox name; if empty, UpdateMetadata will not change the title.
	Orientation  string `json:"orientation,omitempty"`  // "Portrait" or "Landscape"
}

// ── Node construction helpers ──────────────────────────────────────────────

// createElement builds an ElementNode with the given name (may include a
// namespace prefix like "rd:ReportID") and optional attribute pairs.
func createElement(name string, attrs ...[2]string) *xmlquery.Node {
	n := &xmlquery.Node{Type: xmlquery.ElementNode, Data: name}
	if i := strings.IndexByte(name, ':'); i > 0 {
		n.Prefix = name[:i]
		n.Data = name[i+1:]
	}
	for _, a := range attrs {
		// Attr key may also be prefixed ("xmlns:rd", "rd:Type")
		xmlquery.AddAttr(n, a[0], a[1])
	}
	return n
}

// elementWithText builds an ElementNode with a single TextNode child.
func elementWithText(name, text string, attrs ...[2]string) *xmlquery.Node {
	n := createElement(name, attrs...)
	xmlquery.AddChild(n, &xmlquery.Node{Type: xmlquery.TextNode, Data: text})
	return n
}

// newTextNode is a shorthand for a TextNode with the given content.
func newTextNode(text string) *xmlquery.Node {
	return &xmlquery.Node{Type: xmlquery.TextNode, Data: text}
}

// parseFragment parses an XML fragment (possibly using namespaces) and returns
// its single root element. The fragment is wrapped in a temporary root that
// declares the RDL + rd: namespaces so prefixes resolve correctly.
func parseFragment(xmlStr string) (*xmlquery.Node, error) {
	wrapped := `<root xmlns="` + RDLNamespace + `" xmlns:rd="` + RDNamespace + `">` + xmlStr + `</root>`
	doc, err := xmlquery.Parse(strings.NewReader(wrapped))
	if err != nil {
		return nil, fmt.Errorf("parsing fragment: %w", err)
	}
	// Walk to the wrapping <root>, return its first element child.
	for n := doc.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == xmlquery.ElementNode {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == xmlquery.ElementNode {
					return c, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no element found in fragment")
}

// ── Finders ────────────────────────────────────────────────────────────────

// findNamedElement returns the first element with the given tag name and Name
// attribute value anywhere in the document. Used for <DataSource Name="X">,
// <DataSet Name="X">, <ReportParameter Name="X">, <Tablix Name="X">.
func (d *Document) findNamedElement(tag, name string) *xmlquery.Node {
	expr := fmt.Sprintf("//%s[@Name=%q]", tag, name)
	return xmlquery.FindOne(d.root, expr)
}

// findUniqueNamedElement is like findNamedElement but returns an error if the
// element is missing. The verb describes the operation for the error message
// (e.g. "remove", "rename", "find").
func (d *Document) findUniqueNamedElement(verb, tag, name string) (*xmlquery.Node, error) {
	n := d.findNamedElement(tag, name)
	if n == nil {
		return nil, fmt.Errorf("%s: no <%s Name=%q> found", verb, tag, name)
	}
	return n, nil
}

// setElementText sets the inner text of the first descendant element matching
// name. Returns true if changed. Creates the element if missing and create is
// true; the new element is appended as the last child of parent.
func setElementText(parent *xmlquery.Node, name, value string, create bool) bool {
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode && c.Data == name {
			setNodeText(c, value)
			return true
		}
	}
	if !create {
		return false
	}
	xmlquery.AddChild(parent, elementWithText(name, value))
	return true
}

// setNodeText replaces all text content of n with value.
func setNodeText(n *xmlquery.Node, value string) {
	// Replace the first text child if present, otherwise prepend a new one.
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.TextNode {
			c.Data = value
			// Remove any subsequent text/CDATA siblings (rare in RDL).
			for s := c.NextSibling; s != nil; {
				next := s.NextSibling
				if s.Type == xmlquery.TextNode || s.Type == xmlquery.CharDataNode {
					xmlquery.RemoveFromTree(s)
				}
				s = next
			}
			return
		}
	}
	xmlquery.AddChild(n, newTextNode(value))
}

// appendIndented adds n as the last child of parent, preceded by a newline
// plus indent (2 spaces per depth). Depth is the parent's depth from the
// document root.
func appendIndented(parent, n *xmlquery.Node, depth int) {
	xmlquery.AddChild(parent, newTextNode("\n"+strings.Repeat("  ", depth+1)))
	xmlquery.AddChild(parent, n)
}

// ── Idempotency check ──────────────────────────────────────────────────────

// errExists is returned by Add operations when an element with the same Name
// already exists. Callers wrap the message into their summary.
func (d *Document) exists(tag, name string) bool {
	return d.findNamedElement(tag, name) != nil
}

// newUUID generates a fresh UUID v4 string. Used for ReportID/DataSourceID.
func newUUID() string { return uuid.NewString() }

// maybeSave persists doc to path unless dryRun is true. When dryRun, the
// returned summary is prefixed with "[DRY RUN] " so callers can see at a
// glance that nothing was written.
//
// All file-based mutation wrappers route their Save through this helper so
// they get dry-run support for free.
func maybeSave(doc *Document, path, summary string, dryRun bool) (string, error) {
	if dryRun {
		return "[DRY RUN] " + summary, nil
	}
	if err := doc.Save(path); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return summary, nil
}
