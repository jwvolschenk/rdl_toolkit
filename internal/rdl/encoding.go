package rdl

import "fmt"

// FixEncoding ensures the file has UTF-8 BOM and CRLF line endings on next save.
// CRLF is normalised in Document.Save automatically; this only forces BOM.
func FixEncoding(path string) (string, error) {
	doc, err := Load(path)
	if err != nil {
		return "", err
	}
	addedBOM := !doc.hasBOM
	doc.hasBOM = true
	if err := doc.Save(path); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	status := "already correct"
	if addedBOM {
		status = "added BOM"
	}
	return fmt.Sprintf("Fixed encoding for %s (%s, CRLF ensured)", path, status), nil
}
