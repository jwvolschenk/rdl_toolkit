package rdl

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// Clone copies a source RDL to a target with a new ReportID.
func Clone(source, target string) (string, error) {
	raw, hasBOM, err := ReadRDL(source)
	if err != nil {
		return "", fmt.Errorf("reading source: %w", err)
	}

	newID := generateGUID()

	// Replace ReportID - it's in <rd:ReportID>guid</rd:ReportID>
	reportIDTag := []byte("<rd:ReportID>")
	reportIDEnd := []byte("</rd:ReportID>")
	start := bytes.Index(raw, reportIDTag)
	if start != -1 {
		start += len(reportIDTag)
		end := bytes.Index(raw[start:], reportIDEnd)
		if end != -1 {
			var buf []byte
			buf = append(buf, raw[:start]...)
			buf = append(buf, []byte(newID)...)
			buf = append(buf, raw[start+end:]...)
			raw = buf
		}
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating target directory: %w", err)
	}

	if hasBOM {
		raw = append(bomBytes, raw...)
	}
	raw = normalizeCRLF(raw)

	if err := os.WriteFile(target, raw, 0644); err != nil {
		return "", fmt.Errorf("writing target: %w", err)
	}

	return newID, nil
}

func generateGUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
