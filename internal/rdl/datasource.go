package rdl

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// DataSourceOps bundles all DataSource mutations for a single Manage call.
// Fields left zero are no-ops. Operations are applied in this order:
// removes → renames → setConnectStrings → adds.
type DataSourceOps struct {
	Remove           []string              `json:"remove,omitempty"`
	Rename           []RenamePair          `json:"rename,omitempty"`
	SetConnectString []ConnectStringUpdate `json:"setConnectString,omitempty"`
	Add              []DataSourceAdd       `json:"add,omitempty"`
}

// ManageDataSources applies the given operations to the file.
func ManageDataSources(path string, ops DataSourceOps, dryRun bool) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	summary := doc.ManageDataSources(ops)
	return maybeSave(doc, path, summary, dryRun)
}

// ManageDataSources applies ops to the document in place, returning a
// human-readable summary of what changed.
func (d *Document) ManageDataSources(ops DataSourceOps) string {
	var b strings.Builder

	for _, name := range ops.Remove {
		n := d.findNamedElement("DataSource", name)
		if n == nil {
			fmt.Fprintf(&b, "Removed DataSource '%s' (not found)\n", name)
			continue
		}
		xmlquery.RemoveFromTree(n)
		fmt.Fprintf(&b, "Removed DataSource '%s'\n", name)
	}

	for _, r := range ops.Rename {
		if n := d.findNamedElement("DataSource", r.Old); n != nil {
			n.SetAttr("Name", r.New)
			// Also update any <DataSourceName> references in DataSets.
			for _, ref := range xmlquery.Find(d.root, "//DataSourceName") {
				if strings.TrimSpace(ref.InnerText()) == r.Old {
					setNodeText(ref, r.New)
				}
			}
			fmt.Fprintf(&b, "Renamed DataSource '%s' -> '%s'\n", r.Old, r.New)
		} else {
			fmt.Fprintf(&b, "Renamed DataSource '%s' (not found)\n", r.Old)
		}
	}

	for _, u := range ops.SetConnectString {
		n := d.findNamedElement("DataSource", u.Name)
		if n == nil {
			fmt.Fprintf(&b, "Set ConnectString for '%s' (datasource not found)\n", u.Name)
			continue
		}
		if cp := child(n, "ConnectionProperties"); cp != nil {
			if setSimpleChildText(cp, "ConnectString", u.ConnectString) {
				fmt.Fprintf(&b, "Set ConnectString for DataSource '%s'\n", u.Name)
			}
		}
	}

	for _, a := range ops.Add {
		if d.exists("DataSource", a.Name) {
			fmt.Fprintf(&b, "Added DataSource '%s' (already exists, skipped)\n", a.Name)
			continue
		}
		if err := d.addDataSource(a); err != nil {
			fmt.Fprintf(&b, "Add DataSource '%s' failed: %v\n", a.Name, err)
			continue
		}
		fmt.Fprintf(&b, "Added DataSource '%s'\n", a.Name)
	}

	return strings.TrimRight(b.String(), "\n")
}

// addDataSource constructs a <DataSource> element and appends it to <DataSources>.
func (d *Document) addDataSource(spec DataSourceAdd) error {
	container := xmlquery.FindOne(d.root, "//DataSources")
	if container == nil {
		return fmt.Errorf("no <DataSources> element in document")
	}
	security := spec.SecurityType
	if security == "" {
		security = "None"
	}

	ds := createElement("DataSource", [2]string{"Name", spec.Name})
	cp := createElement("ConnectionProperties")
	xmlquery.AddChild(cp, elementWithText("DataProvider", spec.Provider))
	xmlquery.AddChild(cp, elementWithText("ConnectString", spec.ConnectString))
	xmlquery.AddChild(ds, cp)
	xmlquery.AddChild(ds, elementWithText("rd:SecurityType", security))
	xmlquery.AddChild(ds, elementWithText("rd:DataSourceID", newUUID()))

	// Detect indentation from existing siblings, then add the new element.
	childIndent := detectChildIndent(container)
	containerIndent := detectContainerIndent(container)
	appendIndentedWithSuffix(container, ds, childIndent, containerIndent)
	return nil
}

// setSimpleChildText replaces the inner text of the first direct child element
// named `tag` under parent. Returns false if not found.
func setSimpleChildText(parent *xmlquery.Node, tag, value string) bool {
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode && c.Data == tag {
			setNodeText(c, value)
			return true
		}
	}
	return false
}
