package rdl

// Legacy byte-level helpers retained solely for tablix.go and validate.go,
// which Phase 4 (tablix rework) and Phase 5 (XSD validator) will retire.
// New code must use the Document type from document.go instead.

import (
	"bytes"
	"fmt"
	"os"
)

// ReadRDL reads an RDL file and returns raw bytes (BOM stripped) plus a flag
// indicating whether a BOM was present.
func ReadRDL(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("reading file: %w", err)
	}
	hasBOM := bytes.HasPrefix(raw, bomBytes)
	if hasBOM {
		raw = raw[3:]
	}
	return raw, hasBOM, nil
}

// WriteRDL writes RDL content with UTF-8 BOM and CRLF line endings.
func WriteRDL(path string, content []byte) error {
	content = normalizeCRLF(content)
	out := append(bomBytes, content...)
	return os.WriteFile(path, out, 0644)
}

// RDLDocument is the legacy byte-bag representation. Prefer Document.
type RDLDocument struct {
	Content []byte
}

// LoadRDL loads an RDL file into an RDLDocument.
func LoadRDL(path string) (*RDLDocument, error) {
	raw, _, err := ReadRDL(path)
	if err != nil {
		return nil, err
	}
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	raw = bytes.ReplaceAll(raw, []byte("\r"), []byte("\n"))
	return &RDLDocument{Content: raw}, nil
}

// Save writes the document with BOM and CRLF.
func (d *RDLDocument) Save(path string) error {
	return WriteRDL(path, d.Content)
}

// indexByte finds the first occurrence of sep in data. Thin wrapper over
// bytes.Index, kept because tablix.go/validate.go still call it.
func indexByte(data, sep []byte) int { return bytes.Index(data, sep) }
