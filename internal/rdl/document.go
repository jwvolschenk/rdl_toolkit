package rdl

import (
	"bytes"
	"fmt"
	"os"

	"github.com/antchfx/xmlquery"
)

// Namespace constants, BOM bytes, and other file-format assumptions live in
// conventions.go. Document below is the parsed-tree handle.

// Document is a parsed RDL XML tree. All operations go through this type.
type Document struct {
	root   *xmlquery.Node
	hasBOM bool
}

// Load reads an RDL file, strips any BOM, and parses it into an XML tree.
// The hasBOM flag is remembered so Save can re-add it.
func Load(path string) (*Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	hasBOM := bytes.HasPrefix(raw, bomBytes)
	if hasBOM {
		raw = raw[3:]
	}
	root, err := xmlquery.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}
	return &Document{root: root, hasBOM: hasBOM}, nil
}

// Save serializes the tree back to path with UTF-8 BOM and CRLF endings.
// Idempotent: a file produced by Save will round-trip through Load unchanged.
func (d *Document) Save(path string) error {
	var buf bytes.Buffer
	// WithEmptyTagSupport preserves `<Foo />` instead of expanding to `<Foo></Foo>`.
	if err := d.root.WriteWithOptions(&buf, xmlquery.WithEmptyTagSupport()); err != nil {
		return fmt.Errorf("serializing XML: %w", err)
	}
	out := normalizeCRLF(buf.Bytes())
	if d.hasBOM {
		out = append(bomBytes, out...)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

// Root returns the underlying xmlquery document node. Inspect and mutation
// helpers use this to navigate the tree.
func (d *Document) Root() *xmlquery.Node { return d.root }

// normalizeCRLF ensures all line endings are CRLF.
func normalizeCRLF(data []byte) []byte {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	return data
}
